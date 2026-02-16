package api

import (
	"time"

	"github.com/gmsas95/myrai-cli/internal/agent"
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/vector"
	"github.com/gmsas95/myrai-cli/pkg/tools"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Server struct {
	app            *fiber.App
	config         *config.Config
	store          *store.Store
	agent          *agent.Agent
	llmClient      *llm.Client
	tools          *tools.Registry
	skillsRegistry *skills.Registry
	logger         *zap.Logger
	personaManager *persona.PersonaManager
	contextManager *agent.ContextManager
}

func New(cfg *config.Config, store *store.Store, logger *zap.Logger) *Server {
	provider, err := cfg.DefaultProvider()
	if err != nil {
		logger.Fatal("Failed to get LLM provider", zap.Error(err))
	}
	llmClient := llm.NewClient(provider)

	toolRegistry := tools.NewRegistry(cfg.Tools.AllowedCmds)

	personaManager, err := persona.NewPersonaManager(cfg.Storage.DataDir, logger)
	if err != nil {
		logger.Warn("Failed to initialize persona manager", zap.Error(err))
		personaManager = nil
	}

	agentInstance := agent.New(llmClient, toolRegistry, store, logger, personaManager)

	var contextManager *agent.ContextManager
	if cfg.Vector.Enabled {
		vectorSearcher, err := vector.NewSearcher(&cfg.Vector, store, logger)
		if err != nil {
			logger.Warn("Failed to create vector searcher", zap.Error(err))
		} else {
			contextManager = agent.NewContextManager(store, vectorSearcher, llmClient, logger)
			agentInstance.SetContextManager(contextManager)
			logger.Info("Context manager initialized with vector search")
		}
	} else {
		contextManager = agent.NewContextManager(store, nil, llmClient, logger)
		agentInstance.SetContextManager(contextManager)
		logger.Info("Context manager initialized (without vector search)")
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	s := &Server{
		app:            app,
		config:         cfg,
		store:          store,
		agent:          agentInstance,
		llmClient:      llmClient,
		tools:          toolRegistry,
		logger:         logger,
		personaManager: personaManager,
	}

	if s.skillsRegistry != nil {
		s.agent.SetSkillsRegistry(s.skillsRegistry)
	}

	s.setupRoutes()
	return s
}

func (s *Server) SetSkillsRegistry(registry *skills.Registry) {
	s.skillsRegistry = registry
	if s.agent != nil {
		s.agent.SetSkillsRegistry(registry)
	}
}
