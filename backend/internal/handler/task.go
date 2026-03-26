package handler

import (
	"context"
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

type TaskHandler struct {
	store     *store.Store
	publisher *clawsynapse.Client
	log       *zap.Logger
}

func NewTaskHandler(s *store.Store, publisher *clawsynapse.Client, log *zap.Logger) *TaskHandler {
	return &TaskHandler{store: s, publisher: publisher, log: log}
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
		Content string `json:"content"`
		TodoID  string `json:"todo_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid request body"))
		return
	}

	comment, appErr := h.store.AddTaskComment(userID, c.Param("id"), store.TaskCommentInput{
		TaskID:  c.Param("id"),
		TodoID:  body.TodoID,
		Content: body.Content,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusCreated, comment)
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
