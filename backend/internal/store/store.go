package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

type Store struct {
	mu sync.RWMutex

	streamMu sync.RWMutex

	users       map[string]*model.User
	usersByMail map[string]string

	agents      map[string]*model.Agent
	agentByNode map[string]string

	projects map[string]*model.Project

	conversations        map[string]*model.Conversation
	projectConversations map[string][]string

	tasks             map[string]*model.TaskDetail
	projectTasks      map[string][]string
	conversationTasks map[string]string
	taskEvents        map[string][]model.Event
	userEvents        map[string][]*model.Event
	agentEvents       map[string][]*model.Event
	processedMessages map[string]processedMessage

	taskComments map[string][]model.Comment

	notifications     map[string]*model.Notification
	userNotifications map[string][]string

	mongoEnabled           bool
	mongoClient            *mongo.Client
	mongoUsers             *mongo.Collection
	mongoAgents            *mongo.Collection
	mongoProjects          *mongo.Collection
	mongoConversations     *mongo.Collection
	mongoTasks             *mongo.Collection
	mongoEvents            *mongo.Collection
	mongoComments          *mongo.Collection
	mongoProcessedMessages *mongo.Collection
	mongoNotifications     *mongo.Collection
	mongoTimeout           time.Duration
	log                    *zap.Logger

	taskSubscribers         map[string]map[chan model.TaskStreamSnapshot]struct{}
	conversationSubscribers map[string]map[chan model.ConversationStreamSnapshot]struct{}
}

type processedMessage struct {
	Action     string `bson:"action" json:"action"`
	ResourceID string `bson:"resource_id" json:"resource_id"`
}

type AgentPresence struct {
	NodeID     string
	LastSeenAt time.Time
}

func New() *Store {
	return &Store{
		users:                   make(map[string]*model.User),
		usersByMail:             make(map[string]string),
		agents:                  make(map[string]*model.Agent),
		agentByNode:             make(map[string]string),
		projects:                make(map[string]*model.Project),
		conversations:           make(map[string]*model.Conversation),
		projectConversations:    make(map[string][]string),
		tasks:                   make(map[string]*model.TaskDetail),
		projectTasks:            make(map[string][]string),
		conversationTasks:       make(map[string]string),
		taskEvents:              make(map[string][]model.Event),
		userEvents:              make(map[string][]*model.Event),
		agentEvents:             make(map[string][]*model.Event),
		processedMessages:       make(map[string]processedMessage),
		taskComments:            make(map[string][]model.Comment),
		notifications:           make(map[string]*model.Notification),
		userNotifications:       make(map[string][]string),
		taskSubscribers:         make(map[string]map[chan model.TaskStreamSnapshot]struct{}),
		conversationSubscribers: make(map[string]map[chan model.ConversationStreamSnapshot]struct{}),
	}
}

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		panic(errors.New("failed to generate id"))
	}
	return hex.EncodeToString(buf)
}

func copyUser(u *model.User) *model.User {
	clone := *u
	return &clone
}

func copyAgent(a *model.Agent) *model.Agent {
	clone := *a
	if a.Capabilities != nil {
		clone.Capabilities = append([]string(nil), a.Capabilities...)
	}
	if a.LastSeenAt != nil {
		t := *a.LastSeenAt
		clone.LastSeenAt = &t
	}
	return &clone
}

func copyProject(p *model.Project) *model.Project {
	clone := *p
	return &clone
}

func copyTask(t *model.TaskDetail) *model.TaskDetail {
	clone := *t
	clone.Todos = append([]model.Todo{}, t.Todos...)
	clone.Artifacts = append([]model.TaskArtifact{}, t.Artifacts...)
	clone.Result = model.TaskResult{
		Summary:     t.Result.Summary,
		FinalOutput: t.Result.FinalOutput,
		Metadata:    copyMap(t.Result.Metadata),
	}
	for i := range clone.Todos {
		clone.Todos[i].Result.Metadata = copyMap(clone.Todos[i].Result.Metadata)
		clone.Todos[i].Result.ArtifactRefs = append([]model.TodoResultArtifactRef{}, clone.Todos[i].Result.ArtifactRefs...)
	}
	for i := range clone.Artifacts {
		clone.Artifacts[i].Metadata = copyMap(clone.Artifacts[i].Metadata)
	}
	return &clone
}

func copyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mongoWriteError(err error) *transport.AppError {
	return &transport.AppError{
		Status:  500,
		Code:    "INTERNAL_ERROR",
		Message: "failed to persist state",
		Details: map[string]any{"cause": err.Error()},
	}
}
