package app

import (
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/skills/agentic"
	"github.com/gmsas95/myrai-cli/internal/skills/browser"
	"github.com/gmsas95/myrai-cli/internal/skills/documents"
	"github.com/gmsas95/myrai-cli/internal/skills/github"
	"github.com/gmsas95/myrai-cli/internal/skills/health"
	"github.com/gmsas95/myrai-cli/internal/skills/intelligence"
	"github.com/gmsas95/myrai-cli/internal/skills/notes"
	"github.com/gmsas95/myrai-cli/internal/skills/search"
	"github.com/gmsas95/myrai-cli/internal/skills/shopping"
	"github.com/gmsas95/myrai-cli/internal/skills/system"
	"github.com/gmsas95/myrai-cli/internal/skills/voice"
	"github.com/gmsas95/myrai-cli/internal/skills/weather"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

func RegisterSkills(cfg *config.Config, st *store.Store, registry *skills.Registry, logger *zap.Logger) {
	systemSkill := system.NewSystemSkill(cfg.Tools.AllowedCmds)
	registry.Register(systemSkill)

	githubSkill := github.NewGitHubSkill(cfg.Skills.GitHub.Token)
	registry.Register(githubSkill)

	notesSkill := notes.NewNotesSkill("")
	registry.Register(notesSkill)

	weatherSkill := weather.NewWeatherSkill()
	registry.Register(weatherSkill)

	browserSkill := browser.NewBrowserSkill(browser.Config{
		Enabled:  cfg.Skills.Browser.Enabled,
		Headless: cfg.Skills.Browser.Headless,
	})
	registry.Register(browserSkill)

	agenticSkill := agentic.NewAgenticSkill(cfg.Storage.DataDir)
	registry.Register(agenticSkill)

	voiceConfig := voice.DefaultConfig()
	voiceSkill := voice.NewVoiceSkill(voiceConfig)
	registry.Register(voiceSkill)

	docsConfig := documents.DefaultConfig()
	if googleProvider, ok := cfg.LLM.Providers["google"]; ok {
		docsConfig.APIKey = googleProvider.APIKey
	}
	docsSkill := documents.NewDocumentSkill(docsConfig)
	registry.Register(docsSkill)

	shoppingSkill, err := shopping.NewShoppingSkill(st.DB(), logger)
	if err != nil {
		logger.Error("Failed to create shopping skill", zap.Error(err))
	} else {
		registry.Register(shoppingSkill)
	}

	healthSkill, err := health.NewHealthSkill(st.DB(), logger)
	if err != nil {
		logger.Error("Failed to create health skill", zap.Error(err))
	} else {
		registry.Register(healthSkill)
	}

	intelSkill, err := intelligence.NewIntelligenceSkill(st.DB(), logger)
	if err != nil {
		logger.Error("Failed to create intelligence skill", zap.Error(err))
	} else {
		registry.Register(intelSkill)
	}

	searchSkill := search.NewSearchSkill(search.Config{
		Enabled:     cfg.Skills.Search.Enabled,
		Provider:    cfg.Skills.Search.Provider,
		APIKey:      cfg.Skills.Search.APIKey,
		MaxResults:  cfg.Skills.Search.MaxResults,
		TimeoutSecs: cfg.Skills.Search.TimeoutSecs,
	})
	if searchSkill.IsEnabled() {
		registry.Register(searchSkill)
		logger.Info("Search skill registered", zap.String("provider", cfg.Skills.Search.Provider))
	}
}
