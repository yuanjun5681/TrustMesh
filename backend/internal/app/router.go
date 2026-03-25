package app

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/assistant"
	"trustmesh/backend/internal/auth"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/config"
	"trustmesh/backend/internal/embedding"
	"trustmesh/backend/internal/handler"
	"trustmesh/backend/internal/knowledge"
	"trustmesh/backend/internal/middleware"
	"trustmesh/backend/internal/store"
)

type App struct {
	Engine     *gin.Engine
	Store      *store.Store
	PeerSyncer *clawsynapse.PeerSyncer
}

func New(cfg config.Config, log *zap.Logger) (*App, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middleware.Recovery(log))
	engine.Use(middleware.Logging(log))
	engine.Use(middleware.CORS(cfg.AllowAllCORS))

	s, err := store.NewWithConfig(cfg, log)
	if err != nil {
		return nil, err
	}
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.TokenTTL)
	clawClient := clawsynapse.NewClient(cfg.ClawSynapseAPIURL, cfg.ClawSynapseTimeout)
	webhookHandler := clawsynapse.NewWebhookHandler(s, clawClient, cfg.ClawSynapseNodeID, log)
	peerSyncer := clawsynapse.NewPeerSyncer(clawClient, s, cfg.ClawSynapsePeerSync, log)
	if peerSyncer != nil {
		peerSyncer.Start()
	}

	authHandler := handler.NewAuthHandler(s, jwtManager)
	agentHandler := handler.NewAgentHandler(s, clawClient)
	projectHandler := handler.NewProjectHandler(s)
	conversationHandler := handler.NewConversationHandler(s, clawClient, log)
	taskHandler := handler.NewTaskHandler(s, clawClient, log)
	transferHandler := handler.NewTransferHandler(s, clawClient)
	dashboardHandler := handler.NewDashboardHandler(s)
	notificationHandler := handler.NewNotificationHandler(s)
	realtimeHandler := handler.NewRealtimeHandler(s)

	// Knowledge base components (optional - requires EMBEDDING_API_KEY)
	var knowledgeHandler *handler.KnowledgeHandler
	var qdrantClient *knowledge.QdrantClient
	var embeddingClient embedding.Client
	fileStorage := knowledge.NewLocalFileStorage(cfg.KnowledgeStorePath)

	if cfg.EmbeddingAPIKey != "" {
		var err error
		embeddingClient, err = embedding.NewClient(cfg)
		if err != nil {
			log.Warn("embedding client init failed, knowledge search disabled", zap.Error(err))
		}
		qdrantClient = knowledge.NewQdrantClient(cfg.QdrantURL, cfg.EmbeddingDimension)
		if err := qdrantClient.EnsureCollection(context.Background()); err != nil {
			log.Warn("qdrant collection init failed, knowledge search disabled", zap.Error(err))
			qdrantClient = nil
		}
	}

	var processor *knowledge.Processor
	if embeddingClient != nil && qdrantClient != nil {
		processor = knowledge.NewProcessor(fileStorage, embeddingClient, qdrantClient, s, log)
	}
	knowledgeHandler = handler.NewKnowledgeHandler(s, fileStorage, processor, embeddingClient, qdrantClient, log)

	// Inject knowledge components into webhook handler for knowledge.query support
	if embeddingClient != nil && qdrantClient != nil {
		webhookHandler.SetKnowledgeComponents(embeddingClient, qdrantClient)
	}

	engine.GET("/healthz", handler.Health)
	engine.POST("/webhook/clawsynapse", webhookHandler.HandleWebhook)

	v1 := engine.Group("/api/v1")
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)

	authed := v1.Group("")
	authed.Use(middleware.RequireAuth(jwtManager))

	authed.POST("/agents", agentHandler.Create)
	authed.GET("/agents", agentHandler.List)
	authed.GET("/agents/:id", agentHandler.Get)
	authed.PATCH("/agents/:id", agentHandler.Update)
	authed.DELETE("/agents/:id", agentHandler.Delete)
	authed.GET("/agents/:id/stats", agentHandler.Stats)
	authed.GET("/agents/:id/insights", agentHandler.Insights)

	authed.POST("/projects", projectHandler.Create)
	authed.GET("/projects", projectHandler.List)
	authed.GET("/projects/:projectId", projectHandler.Get)
	authed.PATCH("/projects/:projectId", projectHandler.Update)
	authed.DELETE("/projects/:projectId", projectHandler.Archive)

	authed.POST("/projects/:projectId/conversations", conversationHandler.Create)
	authed.GET("/projects/:projectId/conversations", conversationHandler.ListByProject)
	authed.GET("/conversations/:id", conversationHandler.Get)
	authed.POST("/conversations/:id/messages", conversationHandler.AppendMessage)

	authed.GET("/projects/:projectId/tasks", taskHandler.ListByProject)
	authed.GET("/tasks/:id", taskHandler.Get)
	authed.GET("/tasks/:id/events", taskHandler.ListEvents)
	authed.POST("/tasks/:id/todos/:todoId/dispatch", taskHandler.DispatchTodo)
	authed.GET("/tasks/:id/comments", taskHandler.ListComments)
	authed.POST("/tasks/:id/comments", taskHandler.AddComment)
	authed.GET("/tasks/:id/artifacts/:artifactId/transfer", transferHandler.GetTaskArtifactTransfer)
	authed.GET("/tasks/:id/artifacts/:artifactId/content", transferHandler.GetTaskArtifactContent)

	authed.GET("/dashboard/stats", dashboardHandler.Stats)
	authed.GET("/dashboard/events", dashboardHandler.RecentEvents)
	authed.GET("/dashboard/tasks", dashboardHandler.RecentTasks)
	authed.GET("/agents/:id/events", dashboardHandler.AgentEvents)

	authed.GET("/notifications", notificationHandler.List)
	authed.GET("/notifications/unread-count", notificationHandler.UnreadCount)
	authed.PATCH("/notifications/:id/read", notificationHandler.MarkRead)
	authed.POST("/notifications/mark-all-read", notificationHandler.MarkAllRead)
	authed.GET("/events/stream", realtimeHandler.Stream)

	kb := authed.Group("/knowledge")
	kb.POST("/documents", knowledgeHandler.Upload)
	kb.GET("/documents", knowledgeHandler.List)
	kb.GET("/documents/:id", knowledgeHandler.Get)
	kb.PATCH("/documents/:id", knowledgeHandler.Update)
	kb.DELETE("/documents/:id", knowledgeHandler.Delete)
	kb.GET("/documents/:id/chunks", knowledgeHandler.ListChunks)
	kb.POST("/documents/:id/reprocess", knowledgeHandler.Reprocess)
	kb.POST("/search", knowledgeHandler.Search)

	// Assistant (LLM-powered, optional)
	if cfg.AssistantAPIKey != "" {
		llmClient := assistant.NewLLMClient(cfg.AssistantAPIURL, cfg.AssistantAPIKey, cfg.AssistantModel)
		toolExecutor := assistant.NewToolExecutor(s, embeddingClient, qdrantClient)
		hasKnowledge := embeddingClient != nil && qdrantClient != nil
		assistantHandler := handler.NewAssistantHandler(llmClient, toolExecutor, hasKnowledge, log)
		authed.POST("/assistant/chat", assistantHandler.Chat)
		log.Info("assistant enabled", zap.String("model", cfg.AssistantModel))
	}

	return &App{Engine: engine, Store: s, PeerSyncer: peerSyncer}, nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	var errs []error
	if a.PeerSyncer != nil {
		a.PeerSyncer.Close()
	}
	if a.Store != nil {
		if err := a.Store.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
