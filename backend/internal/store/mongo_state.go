package store

import (
	"context"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"trustmesh/backend/internal/config"
	"trustmesh/backend/internal/model"
)

type processedMessageRecord struct {
	ID         string `bson:"_id"`
	Action     string `bson:"action"`
	ResourceID string `bson:"resource_id"`
}

func (s *Store) enableMongo(cfg config.Config, log *zap.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoTimeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return err
	}

	db := client.Database(cfg.MongoDatabase)
	s.mongoEnabled = true
	s.mongoClient = client
	s.mongoUsers = db.Collection("users")
	s.mongoAgents = db.Collection("agents")
	s.mongoProjects = db.Collection("projects")
	s.mongoConversations = db.Collection("conversations")
	s.mongoTasks = db.Collection("tasks")
	s.mongoEvents = db.Collection("events")
	s.mongoComments = db.Collection("comments")
	s.mongoProcessedMessages = db.Collection("processed_messages")
	s.mongoNotifications = db.Collection("notifications")
	s.mongoTimeout = cfg.MongoTimeout
	if log != nil {
		s.log = log
	}

	if err := s.ensureMongoIndexes(); err != nil {
		_ = client.Disconnect(context.Background())
		s.clearMongoCollections()
		return err
	}
	if err := s.loadMongoState(); err != nil {
		_ = client.Disconnect(context.Background())
		s.clearMongoCollections()
		return err
	}

	if s.log != nil {
		s.log.Info("mongo repository store enabled", zap.String("database", cfg.MongoDatabase))
	}
	return nil
}

func (s *Store) Close() error {
	if !s.mongoEnabled || s.mongoClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.mongoTimeout)
	defer cancel()
	return s.mongoClient.Disconnect(ctx)
}

func (s *Store) clearMongoCollections() {
	s.mongoEnabled = false
	s.mongoClient = nil
	s.mongoUsers = nil
	s.mongoAgents = nil
	s.mongoProjects = nil
	s.mongoConversations = nil
	s.mongoTasks = nil
	s.mongoEvents = nil
	s.mongoComments = nil
	s.mongoProcessedMessages = nil
	s.mongoNotifications = nil
}

func (s *Store) mongoContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.mongoTimeout)
}

func (s *Store) ensureMongoIndexes() error {
	indexes := map[*mongo.Collection][]mongo.IndexModel{
		s.mongoUsers: {
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		s.mongoAgents: {
			{Keys: bson.D{{Key: "node_id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}}},
			{Keys: bson.D{{Key: "status", Value: 1}}},
			{Keys: bson.D{{Key: "role", Value: 1}}},
			{Keys: bson.D{{Key: "capabilities", Value: 1}}},
		},
		s.mongoProjects: {
			{Keys: bson.D{{Key: "user_id", Value: 1}}},
			{Keys: bson.D{{Key: "pm_agent_id", Value: 1}}},
		},
		s.mongoTasks: {
			{Keys: bson.D{{Key: "project_id", Value: 1}, {Key: "status", Value: 1}}},
			{Keys: bson.D{{Key: "conversation_id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "pm_agent_id", Value: 1}}},
			{Keys: bson.D{{Key: "todos.assignee.agent_id", Value: 1}, {Key: "todos.status", Value: 1}}},
			{Keys: bson.D{{Key: "status", Value: 1}}},
		},
		s.mongoEvents: {
			{Keys: bson.D{{Key: "task_id", Value: 1}, {Key: "created_at", Value: 1}}},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: 1}}},
			{Keys: bson.D{{Key: "actor_id", Value: 1}, {Key: "created_at", Value: 1}}},
		},
		s.mongoComments: {
			{Keys: bson.D{{Key: "task_id", Value: 1}, {Key: "created_at", Value: 1}}},
		},
		s.mongoNotifications: {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "is_read", Value: 1}, {Key: "created_at", Value: -1}}},
		},
		s.mongoConversations: {
			{Keys: bson.D{{Key: "project_id", Value: 1}, {Key: "user_id", Value: 1}}},
		},
	}

	for collection, models := range indexes {
		if collection == nil || len(models) == 0 {
			continue
		}
		ctx, cancel := s.mongoContext()
		_, err := collection.Indexes().CreateMany(ctx, models)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadMongoState() error {
	users, err := s.loadUsers()
	if err != nil {
		return err
	}
	agents, err := s.loadAgents()
	if err != nil {
		return err
	}
	projects, err := s.loadProjects()
	if err != nil {
		return err
	}
	conversations, projectConversations, err := s.loadConversations()
	if err != nil {
		return err
	}
	tasks, projectTasks, conversationTasks, err := s.loadTasks()
	if err != nil {
		return err
	}
	taskEvents, userEvents, agentEvents, err := s.loadEvents()
	if err != nil {
		return err
	}
	taskComments, err := s.loadComments()
	if err != nil {
		return err
	}
	processedMessages, err := s.loadProcessedMessages()
	if err != nil {
		return err
	}
	notifications, userNotifications, err := s.loadNotifications()
	if err != nil {
		return err
	}

	usersByMail := make(map[string]string, len(users))
	for id, user := range users {
		usersByMail[user.Email] = id
	}
	agentByNode := make(map[string]string, len(agents))
	for id, agent := range agents {
		agentByNode[agent.NodeID] = id
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = users
	s.usersByMail = usersByMail
	s.agents = agents
	s.agentByNode = agentByNode
	s.projects = projects
	s.conversations = conversations
	s.projectConversations = projectConversations
	s.tasks = tasks
	s.projectTasks = projectTasks
	s.conversationTasks = conversationTasks
	s.taskEvents = taskEvents
	s.userEvents = userEvents
	s.agentEvents = agentEvents
	s.taskComments = taskComments
	s.processedMessages = processedMessages
	s.notifications = notifications
	s.userNotifications = userNotifications
	return nil
}

func (s *Store) loadUsers() (map[string]*model.User, error) {
	items := make(map[string]*model.User)
	if s.mongoUsers == nil {
		return items, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoUsers.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []model.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	for i := range users {
		user := users[i]
		items[user.ID] = copyUser(&user)
	}
	return items, nil
}

func (s *Store) loadAgents() (map[string]*model.Agent, error) {
	items := make(map[string]*model.Agent)
	if s.mongoAgents == nil {
		return items, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoAgents.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var agents []model.Agent
	if err := cursor.All(ctx, &agents); err != nil {
		return nil, err
	}
	for i := range agents {
		agent := agents[i]
		items[agent.ID] = copyAgent(&agent)
	}
	return items, nil
}

func (s *Store) loadProjects() (map[string]*model.Project, error) {
	items := make(map[string]*model.Project)
	if s.mongoProjects == nil {
		return items, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoProjects.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []model.Project
	if err := cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	for i := range projects {
		project := projects[i]
		items[project.ID] = copyProject(&project)
	}
	return items, nil
}

func (s *Store) loadConversations() (map[string]*model.Conversation, map[string][]string, error) {
	items := make(map[string]*model.Conversation)
	projectConversations := make(map[string][]string)
	if s.mongoConversations == nil {
		return items, projectConversations, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoConversations.Find(ctx, bson.D{})
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	var conversations []model.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, nil, err
	}
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].CreatedAt.Before(conversations[j].CreatedAt)
	})
	for i := range conversations {
		conversation := conversations[i]
		convCopy := conversation
		items[conversation.ID] = &convCopy
		projectConversations[conversation.ProjectID] = append(projectConversations[conversation.ProjectID], conversation.ID)
	}
	return items, projectConversations, nil
}

func (s *Store) loadTasks() (map[string]*model.TaskDetail, map[string][]string, map[string]string, error) {
	items := make(map[string]*model.TaskDetail)
	projectTasks := make(map[string][]string)
	conversationTasks := make(map[string]string)
	if s.mongoTasks == nil {
		return items, projectTasks, conversationTasks, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoTasks.Find(ctx, bson.D{})
	if err != nil {
		return nil, nil, nil, err
	}
	defer cursor.Close(ctx)

	var tasks []model.TaskDetail
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, nil, nil, err
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	for i := range tasks {
		task := tasks[i]
		items[task.ID] = copyTask(&task)
		projectTasks[task.ProjectID] = append(projectTasks[task.ProjectID], task.ID)
		conversationTasks[task.ConversationID] = task.ID
	}
	return items, projectTasks, conversationTasks, nil
}

func (s *Store) loadEvents() (map[string][]model.Event, map[string][]*model.Event, map[string][]*model.Event, error) {
	taskEvents := make(map[string][]model.Event)
	userEvents := make(map[string][]*model.Event)
	agentEvents := make(map[string][]*model.Event)
	if s.mongoEvents == nil {
		return taskEvents, userEvents, agentEvents, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoEvents.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, nil, nil, err
	}
	defer cursor.Close(ctx)

	var events []model.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, nil, nil, err
	}
	for i := range events {
		event := events[i]
		if event.TaskID != "" {
			taskEvents[event.TaskID] = append(taskEvents[event.TaskID], event)
		}
		if event.UserID != "" {
			userEvents[event.UserID] = append(userEvents[event.UserID], &events[i])
		}
		if event.ActorType == "agent" && event.ActorID != "" {
			agentEvents[event.ActorID] = append(agentEvents[event.ActorID], &events[i])
		}
	}
	return taskEvents, userEvents, agentEvents, nil
}

func (s *Store) loadNotifications() (map[string]*model.Notification, map[string][]string, error) {
	items := make(map[string]*model.Notification)
	userNotifications := make(map[string][]string)
	if s.mongoNotifications == nil {
		return items, userNotifications, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoNotifications.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	var notifications []model.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, nil, err
	}
	for i := range notifications {
		n := notifications[i]
		items[n.ID] = &n
		userNotifications[n.UserID] = append(userNotifications[n.UserID], n.ID)
	}
	return items, userNotifications, nil
}

func (s *Store) loadProcessedMessages() (map[string]processedMessage, error) {
	items := make(map[string]processedMessage)
	if s.mongoProcessedMessages == nil {
		return items, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoProcessedMessages.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []processedMessageRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	for _, record := range records {
		items[record.ID] = processedMessage{
			Action:     record.Action,
			ResourceID: record.ResourceID,
		}
	}
	return items, nil
}

func (s *Store) persistUserUnsafe(user *model.User) error {
	if !s.mongoEnabled || s.mongoUsers == nil || user == nil {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	_, err := s.mongoUsers.ReplaceOne(ctx, bson.M{"_id": user.ID}, copyUser(user), options.Replace().SetUpsert(true))
	return err
}

func (s *Store) persistAgentUnsafe(agent *model.Agent) error {
	if !s.mongoEnabled || s.mongoAgents == nil || agent == nil {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	_, err := s.mongoAgents.ReplaceOne(ctx, bson.M{"_id": agent.ID}, copyAgent(agent), options.Replace().SetUpsert(true))
	return err
}

func (s *Store) deleteAgentUnsafe(agentID string) error {
	if !s.mongoEnabled || s.mongoAgents == nil || strings.TrimSpace(agentID) == "" {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	_, err := s.mongoAgents.DeleteOne(ctx, bson.M{"_id": agentID})
	return err
}

func (s *Store) persistProjectUnsafe(project *model.Project) error {
	if !s.mongoEnabled || s.mongoProjects == nil || project == nil {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	_, err := s.mongoProjects.ReplaceOne(ctx, bson.M{"_id": project.ID}, copyProject(project), options.Replace().SetUpsert(true))
	return err
}

func (s *Store) persistConversationUnsafe(conversation *model.Conversation) error {
	if !s.mongoEnabled || s.mongoConversations == nil || conversation == nil {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	clone := *conversation
	clone.Messages = append([]model.ConversationMessage(nil), conversation.Messages...)
	_, err := s.mongoConversations.ReplaceOne(ctx, bson.M{"_id": conversation.ID}, clone, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) persistTaskUnsafe(task *model.TaskDetail) error {
	if !s.mongoEnabled || s.mongoTasks == nil || task == nil {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	_, err := s.mongoTasks.ReplaceOne(ctx, bson.M{"_id": task.ID}, copyTask(task), options.Replace().SetUpsert(true))
	return err
}

func (s *Store) persistTaskEventsUnsafe(taskID string) error {
	if !s.mongoEnabled || s.mongoEvents == nil || strings.TrimSpace(taskID) == "" {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	if _, err := s.mongoEvents.DeleteMany(ctx, bson.M{"task_id": taskID}); err != nil {
		return err
	}
	events := s.taskEvents[taskID]
	if len(events) == 0 {
		return nil
	}
	docs := make([]any, 0, len(events))
	for _, event := range events {
		docs = append(docs, event)
	}
	_, err := s.mongoEvents.InsertMany(ctx, docs)
	return err
}

func (s *Store) persistNotificationUnsafe(n *model.Notification) {
	if !s.mongoEnabled || s.mongoNotifications == nil || n == nil {
		return
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	clone := *n
	if _, err := s.mongoNotifications.ReplaceOne(ctx, bson.M{"_id": n.ID}, clone, options.Replace().SetUpsert(true)); err != nil {
		if s.log != nil {
			s.log.Warn("failed to persist notification", zap.String("id", n.ID), zap.Error(err))
		}
	}
}

func (s *Store) persistProcessedMessageUnsafe(key string) error {
	if !s.mongoEnabled || s.mongoProcessedMessages == nil || key == "" {
		return nil
	}
	record, ok := s.processedMessages[key]
	if !ok {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	_, err := s.mongoProcessedMessages.ReplaceOne(ctx, bson.M{"_id": key}, processedMessageRecord{
		ID:         key,
		Action:     record.Action,
		ResourceID: record.ResourceID,
	}, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) persistAgentGraphUnsafe(agentID string) error {
	agent, ok := s.agents[agentID]
	if !ok {
		return nil
	}
	if err := s.persistAgentUnsafe(agent); err != nil {
		return err
	}
	for _, project := range s.projects {
		if project.PMAgentID == agentID {
			if err := s.persistProjectUnsafe(project); err != nil {
				return err
			}
		}
	}
	for _, task := range s.tasks {
		needsPersist := task.PMAgentID == agentID
		if !needsPersist {
			for _, todo := range task.Todos {
				if todo.Assignee.AgentID == agentID {
					needsPersist = true
					break
				}
			}
		}
		if needsPersist {
			if err := s.persistTaskUnsafe(task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) persistTaskBundleUnsafe(taskID string) error {
	task, ok := s.tasks[taskID]
	if !ok {
		return nil
	}
	if err := s.persistTaskUnsafe(task); err != nil {
		return err
	}
	return s.persistTaskEventsUnsafe(taskID)
}

func (s *Store) loadComments() (map[string][]model.Comment, error) {
	taskComments := make(map[string][]model.Comment)
	if s.mongoComments == nil {
		return taskComments, nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	cursor, err := s.mongoComments.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []model.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	for _, c := range comments {
		taskComments[c.TaskID] = append(taskComments[c.TaskID], c)
	}
	return taskComments, nil
}

func (s *Store) persistCommentUnsafe(c *model.Comment) error {
	if !s.mongoEnabled || s.mongoComments == nil || c == nil {
		return nil
	}
	ctx, cancel := s.mongoContext()
	defer cancel()
	_, err := s.mongoComments.ReplaceOne(ctx, bson.M{"_id": c.ID}, c, options.Replace().SetUpsert(true))
	return err
}
