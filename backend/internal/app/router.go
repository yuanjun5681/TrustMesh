package app

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/auth"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/config"
	"trustmesh/backend/internal/handler"
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
	agentHandler := handler.NewAgentHandler(s)
	projectHandler := handler.NewProjectHandler(s)
	conversationHandler := handler.NewConversationHandler(s, clawClient, log)
	taskHandler := handler.NewTaskHandler(s, clawClient, log)

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
