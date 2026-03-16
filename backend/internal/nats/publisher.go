package nats

import (
	"encoding/json"
	"fmt"

	natslib "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type Publisher struct {
	nc  *natslib.Conn
	log *zap.Logger
}

func NewPublisher(nc *natslib.Conn, log *zap.Logger) *Publisher {
	return &Publisher{nc: nc, log: log}
}

type ConversationMessagePayload struct {
	ConversationID string `json:"conversation_id"`
	ProjectID      string `json:"project_id"`
	Content        string `json:"content"`
}

type TaskCreatedPayload struct {
	TaskID         string `json:"task_id"`
	ProjectID      string `json:"project_id"`
	ConversationID string `json:"conversation_id"`
	Title          string `json:"title"`
}

type TaskUpdatedPayload struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type TodoAssignedPayload struct {
	TaskID      string `json:"task_id"`
	TodoID      string `json:"todo_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type TodoUpdatedPayload struct {
	TaskID  string `json:"task_id"`
	TodoID  string `json:"todo_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (p *Publisher) NotifyConversationMessage(nodeID string, payload ConversationMessagePayload) error {
	subject := fmt.Sprintf("notify.%s.conversation.message", nodeID)
	return p.publish(subject, payload)
}

func (p *Publisher) NotifyTaskCreated(nodeID string, payload TaskCreatedPayload) error {
	subject := fmt.Sprintf("notify.%s.task.created", nodeID)
	return p.publish(subject, payload)
}

func (p *Publisher) NotifyTaskUpdated(nodeID string, payload TaskUpdatedPayload) error {
	subject := fmt.Sprintf("notify.%s.task.updated", nodeID)
	return p.publish(subject, payload)
}

func (p *Publisher) NotifyTodoAssigned(nodeID string, payload TodoAssignedPayload) error {
	subject := fmt.Sprintf("notify.%s.todo.assigned", nodeID)
	return p.publish(subject, payload)
}

func (p *Publisher) NotifyTodoUpdated(nodeID string, payload TodoUpdatedPayload) error {
	subject := fmt.Sprintf("notify.%s.todo.updated", nodeID)
	return p.publish(subject, payload)
}

func (p *Publisher) publish(subject string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := p.nc.Publish(subject, body); err != nil {
		return err
	}
	p.log.Debug("nats publish", zap.String("subject", subject))
	return nil
}
