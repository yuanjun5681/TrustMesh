package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/protocol"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type addTaskCommentResponse struct {
	Comment           model.Comment                  `json:"comment"`
	MentionDeliveries []model.CommentMentionDelivery `json:"mention_deliveries,omitempty"`
}

type TaskHandler struct {
	store     *store.Store
	publisher *clawsynapse.Client
	log       *zap.Logger
}

func NewTaskHandler(s *store.Store, publisher *clawsynapse.Client, log *zap.Logger) *TaskHandler {
	return &TaskHandler{store: s, publisher: publisher, log: log}
}

func (h *TaskHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var body struct {
		Title           string `json:"title"`
		Description     string `json:"description"`
		Priority        string `json:"priority"`
		AssigneeAgentID string `json:"assignee_agent_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid request body"))
		return
	}

	task, appErr := h.store.CreateTaskByUser(userID, store.UserTaskCreateInput{
		ProjectID:       c.Param("projectId"),
		Title:           body.Title,
		Description:     body.Description,
		Priority:        body.Priority,
		AssigneeAgentID: body.AssigneeAgentID,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	// Auto-dispatch the todo to the agent
	todo := &task.Todos[0]
	if h.publisher != nil {
		payload := protocol.TodoAssignedPayload{
			TaskID:      task.ID,
			TodoID:      todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Content:     "你收到了一个新的 Todo 任务。请使用 /tm-task-exec skill 执行此任务，按要求回报进度和结果。",
			ExecBrief: &protocol.TodoExecBrief{
				Objective:    "执行分派的 Todo 任务；及时回报进度；完成后提交结果，失败时说明原因。",
				MustUseSkill: "tm-task-exec",
			},
		}
		if _, err := h.publisher.Publish(c.Request.Context(), todo.Assignee.NodeID, "todo.assigned", payload, task.ID, map[string]any{"source": "user_created"}); err != nil {
			if h.log != nil {
				h.log.Warn("auto dispatch todo failed for user-created task", zap.String("task_id", task.ID), zap.String("todo_id", todo.ID), zap.Error(err))
			}
			// Task is created but dispatch failed — return the task anyway
			transport.WriteData(c, http.StatusCreated, task)
			return
		}

		dispatched, dispatchErr := h.store.RecordTodoDispatch(userID, task.ID, todo.ID)
		if dispatchErr != nil {
			if h.log != nil {
				h.log.Warn("record todo dispatch failed", zap.String("task_id", task.ID), zap.String("todo_id", todo.ID), zap.Error(dispatchErr))
			}
			transport.WriteData(c, http.StatusCreated, task)
			return
		}
		task = dispatched
	}

	transport.WriteData(c, http.StatusCreated, task)
}

func (h *TaskHandler) ListByProject(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	status := c.Query("status")
	items, appErr := h.store.ListTasks(userID, c.Param("projectId"), status)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, items, len(items))
}

func (h *TaskHandler) Get(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	task, appErr := h.store.GetTask(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, task)
}

func (h *TaskHandler) ListEvents(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	events, appErr := h.store.ListTaskEvents(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, events, len(events))
}

func (h *TaskHandler) DispatchTodo(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	task, appErr := h.store.GetTask(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	if appErr := h.store.CheckTaskProjectActive(task.ID); appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	todo := findTaskTodo(task, c.Param("todoId"))
	if todo == nil {
		transport.WriteError(c, transport.NotFound("todo not found"))
		return
	}
	if todo.Status != "pending" {
		transport.WriteError(c, transport.Conflict("TODO_NOT_PENDING", "todo is not pending"))
		return
	}
	if !task.CanDispatchTodo(todo.ID) {
		transport.WriteError(c, transport.Conflict("TODO_BLOCKED_BY_PREVIOUS", "todo is blocked by previous todos"))
		return
	}
	if h.publisher == nil {
		transport.WriteError(c, transport.NewError(http.StatusServiceUnavailable, "CLAWSYNAPSE_DISABLED", "clawsynapse client is disabled"))
		return
	}

	payload := protocol.TodoAssignedPayload{
		TaskID:      task.ID,
		TodoID:      todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Content:     "你收到了一个新的 Todo 任务。请使用 /tm-task-exec skill 执行此任务，按要求回报进度和结果。",
		ExecBrief: &protocol.TodoExecBrief{
			Objective:    "执行分派的 Todo 任务；及时回报进度；完成后提交结果，失败时说明原因。",
			MustUseSkill: "tm-task-exec",
		},
	}
	if _, err := h.publisher.Publish(context.Background(), todo.Assignee.NodeID, "todo.assigned", payload, task.ID, map[string]any{"source": "manual_dispatch"}); err != nil {
		if h.log != nil {
			h.log.Warn("manual todo dispatch failed", zap.String("task_id", task.ID), zap.String("todo_id", todo.ID), zap.String("target_node", todo.Assignee.NodeID), zap.Error(err))
		}
		transport.WriteError(c, transport.NewError(http.StatusBadGateway, "TODO_DISPATCH_FAILED", "failed to dispatch todo to agent"))
		return
	}

	task, appErr = h.store.RecordTodoDispatch(userID, task.ID, todo.ID)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, task)
}

func (h *TaskHandler) AddComment(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var body struct {
		Content  string `json:"content"`
		TodoID   string `json:"todo_id"`
		Mentions []struct {
			AgentID string `json:"agent_id"`
		} `json:"mentions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid request body"))
		return
	}

	mentions := make([]store.TaskCommentMentionInput, 0, len(body.Mentions))
	for _, mention := range body.Mentions {
		mentions = append(mentions, store.TaskCommentMentionInput{AgentID: mention.AgentID})
	}

	comment, appErr := h.store.AddTaskComment(userID, c.Param("id"), store.TaskCommentInput{
		TaskID:   c.Param("id"),
		TodoID:   body.TodoID,
		Content:  body.Content,
		Mentions: mentions,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	task, appErr := h.store.GetTask(userID, c.Param("id"))
	if appErr != nil {
		if h.log != nil {
			h.log.Warn("load task after comment failed", zap.String("task_id", c.Param("id")), zap.Error(appErr))
		}
		transport.WriteData(c, http.StatusCreated, addTaskCommentResponse{Comment: *comment})
		return
	}

	transport.WriteData(c, http.StatusCreated, addTaskCommentResponse{
		Comment:           *comment,
		MentionDeliveries: h.publishTaskCommentMentions(c.Request.Context(), task, comment),
	})
}

func (h *TaskHandler) Cancel(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid request body"))
		return
	}

	task, appErr := h.store.CancelTask(userID, store.TaskCancelInput{
		TaskID: c.Param("id"),
		Reason: strings.TrimSpace(body.Reason),
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, task)
}

func (h *TaskHandler) ListComments(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	comments, appErr := h.store.ListTaskComments(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, comments, len(comments))
}

func findTaskTodo(task *model.TaskDetail, todoID string) *model.Todo {
	for i := range task.Todos {
		if task.Todos[i].ID == todoID {
			return &task.Todos[i]
		}
	}
	return nil
}

func (h *TaskHandler) publishTaskCommentMentions(ctx context.Context, task *model.TaskDetail, comment *model.Comment) []model.CommentMentionDelivery {
	if comment == nil || len(comment.Mentions) == 0 {
		return nil
	}

	deliveries := make([]model.CommentMentionDelivery, 0, len(comment.Mentions))
	for _, mention := range comment.Mentions {
		delivery := model.CommentMentionDelivery{
			AgentID:   mention.AgentID,
			AgentName: mention.AgentName,
			Status:    "sent",
		}

		if h.publisher == nil {
			delivery.Status = "failed"
			delivery.Error = "clawsynapse client is disabled"
			deliveries = append(deliveries, delivery)
			continue
		}
		if strings.TrimSpace(mention.NodeID) == "" {
			delivery.Status = "failed"
			delivery.Error = "target agent node is missing"
			deliveries = append(deliveries, delivery)
			continue
		}

		payload := h.buildTaskMentionPayload(task, comment, mention)
		if _, err := h.publisher.Publish(ctx, mention.NodeID, "task.mention", payload, task.ID, map[string]any{
			"source":     "task_comment_mention",
			"task_id":    task.ID,
			"comment_id": comment.ID,
		}); err != nil {
			delivery.Status = "failed"
			delivery.Error = err.Error()
			if h.log != nil {
				h.log.Warn(
					"publish task mention failed",
					zap.String("task_id", task.ID),
					zap.String("comment_id", comment.ID),
					zap.String("target_agent_id", mention.AgentID),
					zap.String("target_node", mention.NodeID),
					zap.Error(err),
				)
			}
		}

		deliveries = append(deliveries, delivery)
	}

	return deliveries
}

func (h *TaskHandler) buildTaskMentionPayload(task *model.TaskDetail, comment *model.Comment, mention model.CommentMention) protocol.TaskMentionPayload {
	payload := protocol.TaskMentionPayload{
		TaskID:          task.ID,
		ProjectID:       task.ProjectID,
		ConversationID:  task.ConversationID,
		CommentID:       comment.ID,
		TodoID:          comment.TodoID,
		TaskTitle:       task.Title,
		TaskDescription: task.Description,
		TaskStatus:      task.Status,
		TaskPriority:    task.Priority,
		AuthorName:      comment.ActorName,
		UserContent:     comment.Content,
		Content: fmt.Sprintf(
			"用户在任务评论中 @%s。请结合任务上下文阅读这条评论；如需回应，请通过 task.comment 回复并携带 task_id=%s。",
			mention.AgentName,
			task.ID,
		),
	}

	if todo := findTaskTodo(task, comment.TodoID); todo != nil {
		payload.TodoTitle = todo.Title
	}

	return payload
}
