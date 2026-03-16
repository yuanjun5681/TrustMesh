package nats

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	natslib "github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"trustmesh/backend/internal/config"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/store"
)

type Service struct {
	nc    *natslib.Conn
	store *store.Store
	log   *zap.Logger
	pub   *Publisher
}

type envelope struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	NodeID    string          `json:"node_id"`
	Payload   json.RawMessage `json:"payload"`
}

type rpcReply struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Start(cfg config.Config, log *zap.Logger, st *store.Store) (*Service, error) {
	nc, err := natslib.Connect(
		cfg.NATSURL,
		natslib.Name(cfg.NATSClient),
		natslib.Timeout(cfg.NATSTimeout),
		natslib.ReconnectWait(time.Second),
		natslib.MaxReconnects(-1),
		natslib.DisconnectErrHandler(func(_ *natslib.Conn, err error) {
			if err != nil {
				log.Warn("nats disconnected", zap.Error(err))
			} else {
				log.Warn("nats disconnected")
			}
		}),
		natslib.ReconnectHandler(func(conn *natslib.Conn) {
			log.Info("nats reconnected", zap.String("url", conn.ConnectedUrl()))
		}),
	)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		nc:    nc,
		store: st,
		log:   log,
		pub:   NewPublisher(nc, log),
	}

	if _, err := nc.QueueSubscribe("agent.*.*.*", "trustmesh-backend", svc.handleAgentMessage); err != nil {
		nc.Close()
		return nil, err
	}
	if _, err := nc.QueueSubscribe("rpc.*.*.*", "trustmesh-backend", svc.handleRPCMessage); err != nil {
		nc.Close()
		return nil, err
	}
	if err := nc.Flush(); err != nil {
		nc.Close()
		return nil, err
	}

	log.Info("nats service started", zap.String("url", cfg.NATSURL))
	return svc, nil
}

func (s *Service) Publisher() *Publisher {
	return s.pub
}

func (s *Service) Close() error {
	if s == nil || s.nc == nil {
		return nil
	}
	return s.nc.Drain()
}

type heartbeatPayload struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type conversationReplyPayload struct {
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
}

type taskCreatePayload struct {
	ProjectID      string                  `json:"project_id"`
	ConversationID string                  `json:"conversation_id"`
	Title          string                  `json:"title"`
	Description    string                  `json:"description"`
	Todos          []taskCreateTodoPayload `json:"todos"`
}

type taskCreateTodoPayload struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	AssigneeNodeID string `json:"assignee_node_id"`
}

type todoProgressPayload struct {
	TaskID  string `json:"task_id"`
	TodoID  string `json:"todo_id"`
	Message string `json:"message"`
}

type todoCompletePayload struct {
	TaskID string           `json:"task_id"`
	TodoID string           `json:"todo_id"`
	Result model.TodoResult `json:"result"`
}

type todoFailPayload struct {
	TaskID string `json:"task_id"`
	TodoID string `json:"todo_id"`
	Error  string `json:"error"`
}

type rpcTaskGetPayload struct {
	TaskID string `json:"task_id"`
}

type rpcProjectSummaryPayload struct {
	ProjectID string `json:"project_id"`
}

type rpcTaskByConversationPayload struct {
	ConversationID string `json:"conversation_id"`
}

type rpcAgentListPayload struct {
	ProjectID string `json:"project_id,omitempty"`
}

func (s *Service) handleAgentMessage(msg *natslib.Msg) {
	subject, err := ParseSubject(msg.Subject)
	if err != nil {
		s.log.Warn("invalid agent subject", zap.String("subject", msg.Subject), zap.Error(err))
		return
	}
	if subject.Namespace != "agent" {
		return
	}

	env, err := decodeEnvelope(msg.Data, subject.NodeID)
	if err != nil {
		s.log.Warn("invalid agent envelope", zap.String("subject", msg.Subject), zap.Error(err))
		return
	}

	switch subject.Domain + "." + subject.Action {
	case "system.heartbeat":
		s.handleHeartbeat(subject, env)
	case "conversation.reply":
		s.handleConversationReply(subject, env)
	case "task.create":
		s.handleTaskCreate(subject, env)
	case "todo.progress":
		s.handleTodoProgress(subject, env)
	case "todo.complete":
		s.handleTodoComplete(subject, env)
	case "todo.fail":
		s.handleTodoFail(subject, env)
	default:
		s.log.Warn("unsupported agent action", zap.String("subject", msg.Subject))
	}
}

func (s *Service) handleRPCMessage(msg *natslib.Msg) {
	subject, err := ParseSubject(msg.Subject)
	if err != nil {
		s.log.Warn("invalid rpc subject", zap.String("subject", msg.Subject), zap.Error(err))
		return
	}
	if subject.Namespace != "rpc" {
		return
	}

	env, err := decodeEnvelope(msg.Data, subject.NodeID)
	if err != nil {
		s.log.Warn("invalid rpc envelope", zap.String("subject", msg.Subject), zap.Error(err))
		s.replyRPC(msg.Reply, rpcReply{Success: false, Error: "BAD_ENVELOPE"})
		return
	}

	switch subject.Domain + "." + subject.Action {
	case "task.get":
		var payload rpcTaskGetPayload
		if err := decodePayload(env.Payload, &payload); err != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: "BAD_PAYLOAD"})
			return
		}
		task, appErr := s.store.GetTaskForNode(subject.NodeID, payload.TaskID)
		if appErr != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: appErr.Code})
			return
		}
		s.replyRPC(msg.Reply, rpcReply{Success: true, Data: task})
	case "todo.assigned":
		items, appErr := s.store.ListAssignedTodosForNode(subject.NodeID)
		if appErr != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: appErr.Code})
			return
		}
		s.replyRPC(msg.Reply, rpcReply{Success: true, Data: map[string]any{"items": items}})
	case "project.summary":
		var payload rpcProjectSummaryPayload
		if err := decodePayload(env.Payload, &payload); err != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: "BAD_PAYLOAD"})
			return
		}
		summary, appErr := s.store.GetProjectSummaryForPMNode(subject.NodeID, payload.ProjectID)
		if appErr != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: appErr.Code})
			return
		}
		s.replyRPC(msg.Reply, rpcReply{Success: true, Data: summary})
	case "task.by_conversation":
		var payload rpcTaskByConversationPayload
		if err := decodePayload(env.Payload, &payload); err != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: "BAD_PAYLOAD"})
			return
		}
		task, appErr := s.store.GetTaskByConversationForPMNode(subject.NodeID, payload.ConversationID)
		if appErr != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: appErr.Code})
			return
		}
		s.replyRPC(msg.Reply, rpcReply{Success: true, Data: task})
	case "agent.list":
		var payload rpcAgentListPayload
		if err := decodePayload(env.Payload, &payload); err != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: "BAD_PAYLOAD"})
			return
		}
		items, appErr := s.store.ListCandidateAgentsForPMNode(subject.NodeID, payload.ProjectID)
		if appErr != nil {
			s.replyRPC(msg.Reply, rpcReply{Success: false, Error: appErr.Code})
			return
		}
		s.replyRPC(msg.Reply, rpcReply{Success: true, Data: map[string]any{"items": items}})
	default:
		s.replyRPC(msg.Reply, rpcReply{Success: false, Error: "UNSUPPORTED_RPC"})
	}
}

func (s *Service) handleHeartbeat(subject Subject, env envelope) {
	var payload heartbeatPayload
	if err := decodePayload(env.Payload, &payload); err != nil {
		s.log.Warn("invalid heartbeat payload", zap.String("node_id", subject.NodeID), zap.Error(err))
		return
	}
	if strings.TrimSpace(payload.Status) == "" {
		payload.Status = "online"
	}
	ts := payload.Timestamp
	if ts.IsZero() {
		ts = env.Timestamp
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	if _, appErr := s.store.UpdateAgentHeartbeat(subject.NodeID, payload.Status, ts); appErr != nil {
		s.log.Warn("heartbeat rejected", zap.String("node_id", subject.NodeID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}
	s.log.Debug("heartbeat accepted", zap.String("node_id", subject.NodeID), zap.String("status", payload.Status))
}

func (s *Service) handleConversationReply(subject Subject, env envelope) {
	var payload conversationReplyPayload
	if err := decodePayload(env.Payload, &payload); err != nil {
		s.log.Warn("invalid conversation.reply payload", zap.String("node_id", subject.NodeID), zap.Error(err))
		return
	}
	if _, appErr := s.store.AppendPMReplyByNode(subject.NodeID, payload.ConversationID, payload.Content); appErr != nil {
		s.log.Warn("conversation.reply rejected", zap.String("node_id", subject.NodeID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}
	s.log.Info("conversation reply accepted", zap.String("node_id", subject.NodeID), zap.String("conversation_id", payload.ConversationID))
}

func (s *Service) handleTaskCreate(subject Subject, env envelope) {
	var payload taskCreatePayload
	if err := decodePayload(env.Payload, &payload); err != nil {
		s.log.Warn("invalid task.create payload", zap.String("node_id", subject.NodeID), zap.Error(err))
		return
	}
	in := store.TaskCreateInput{
		ProjectID:      payload.ProjectID,
		ConversationID: payload.ConversationID,
		Title:          payload.Title,
		Description:    payload.Description,
		Todos:          make([]store.TaskCreateTodoInput, 0, len(payload.Todos)),
	}
	for _, td := range payload.Todos {
		in.Todos = append(in.Todos, store.TaskCreateTodoInput{
			ID:             td.ID,
			Title:          td.Title,
			Description:    td.Description,
			AssigneeNodeID: td.AssigneeNodeID,
		})
	}
	task, appErr := s.store.CreateTaskByPMNodeWithMessageID(subject.NodeID, env.ID, in)
	if appErr != nil {
		s.log.Warn("task.create rejected", zap.String("node_id", subject.NodeID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}

	if err := s.pub.NotifyTaskCreated(task.PMAgent.NodeID, TaskCreatedPayload{
		TaskID:         task.ID,
		ProjectID:      task.ProjectID,
		ConversationID: task.ConversationID,
		Title:          task.Title,
	}); err != nil {
		s.log.Warn("notify task.created failed", zap.Error(err), zap.String("task_id", task.ID))
	}
	for _, todo := range task.Todos {
		if err := s.pub.NotifyTodoAssigned(todo.Assignee.NodeID, TodoAssignedPayload{
			TaskID:      task.ID,
			TodoID:      todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
		}); err != nil {
			s.log.Warn("notify todo.assigned failed", zap.Error(err), zap.String("task_id", task.ID), zap.String("todo_id", todo.ID))
		}
	}
	s.log.Info("task.create accepted", zap.String("task_id", task.ID), zap.String("pm_node_id", subject.NodeID))
}

func (s *Service) handleTodoProgress(subject Subject, env envelope) {
	var payload todoProgressPayload
	if err := decodePayload(env.Payload, &payload); err != nil {
		s.log.Warn("invalid todo.progress payload", zap.String("node_id", subject.NodeID), zap.Error(err))
		return
	}
	task, appErr := s.store.UpdateTodoProgressByNode(subject.NodeID, store.TodoProgressInput{
		TaskID:  payload.TaskID,
		TodoID:  payload.TodoID,
		Message: payload.Message,
	})
	if appErr != nil {
		s.log.Warn("todo.progress rejected", zap.String("node_id", subject.NodeID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}
	s.publishTaskAndTodoUpdates(task, payload.TodoID, payload.Message)
}

func (s *Service) handleTodoComplete(subject Subject, env envelope) {
	var payload todoCompletePayload
	if err := decodePayload(env.Payload, &payload); err != nil {
		s.log.Warn("invalid todo.complete payload", zap.String("node_id", subject.NodeID), zap.Error(err))
		return
	}
	task, appErr := s.store.CompleteTodoByNodeWithMessageID(subject.NodeID, env.ID, store.TodoCompleteInput{
		TaskID: payload.TaskID,
		TodoID: payload.TodoID,
		Result: payload.Result,
	})
	if appErr != nil {
		s.log.Warn("todo.complete rejected", zap.String("node_id", subject.NodeID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}
	s.publishTaskAndTodoUpdates(task, payload.TodoID, "completed")
}

func (s *Service) handleTodoFail(subject Subject, env envelope) {
	var payload todoFailPayload
	if err := decodePayload(env.Payload, &payload); err != nil {
		s.log.Warn("invalid todo.fail payload", zap.String("node_id", subject.NodeID), zap.Error(err))
		return
	}
	task, appErr := s.store.FailTodoByNode(subject.NodeID, store.TodoFailInput{
		TaskID: payload.TaskID,
		TodoID: payload.TodoID,
		Error:  payload.Error,
	})
	if appErr != nil {
		s.log.Warn("todo.fail rejected", zap.String("node_id", subject.NodeID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}
	s.publishTaskAndTodoUpdates(task, payload.TodoID, payload.Error)
}

func (s *Service) publishTaskAndTodoUpdates(task *model.TaskDetail, todoID, message string) {
	if err := s.pub.NotifyTaskUpdated(task.PMAgent.NodeID, TaskUpdatedPayload{TaskID: task.ID, Status: task.Status}); err != nil {
		s.log.Warn("notify task.updated failed", zap.Error(err), zap.String("task_id", task.ID))
	}
	todo := findTodo(task, todoID)
	if todo == nil {
		return
	}
	payload := TodoUpdatedPayload{TaskID: task.ID, TodoID: todo.ID, Status: todo.Status, Message: message}
	if err := s.pub.NotifyTodoUpdated(todo.Assignee.NodeID, payload); err != nil {
		s.log.Warn("notify todo.updated failed", zap.Error(err), zap.String("task_id", task.ID), zap.String("todo_id", todo.ID))
	}
	if task.PMAgent.NodeID != todo.Assignee.NodeID {
		if err := s.pub.NotifyTodoUpdated(task.PMAgent.NodeID, payload); err != nil {
			s.log.Warn("notify pm todo.updated failed", zap.Error(err), zap.String("task_id", task.ID), zap.String("todo_id", todo.ID))
		}
	}
}

func (s *Service) replyRPC(replySubject string, payload rpcReply) {
	if replySubject == "" {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		s.log.Warn("marshal rpc reply failed", zap.Error(err), zap.String("reply", replySubject))
		return
	}
	if err := s.nc.Publish(replySubject, body); err != nil {
		s.log.Warn("publish rpc reply failed", zap.Error(err), zap.String("reply", replySubject))
	}
}

func decodeEnvelope(raw []byte, subjectNodeID string) (envelope, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return envelope{}, err
	}
	if strings.TrimSpace(env.NodeID) == "" {
		return envelope{}, fmt.Errorf("missing node_id")
	}
	if strings.TrimSpace(env.ID) == "" {
		return envelope{}, fmt.Errorf("missing id")
	}
	if env.NodeID != subjectNodeID {
		return envelope{}, fmt.Errorf("node_id mismatch: subject=%s payload=%s", subjectNodeID, env.NodeID)
	}
	if len(env.Payload) == 0 {
		env.Payload = []byte("{}")
	}
	return env, nil
}

func decodePayload(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty payload")
	}
	return json.Unmarshal(raw, out)
}

func findTodo(task *model.TaskDetail, todoID string) *model.Todo {
	for i := range task.Todos {
		if task.Todos[i].ID == todoID {
			return &task.Todos[i]
		}
	}
	return nil
}
