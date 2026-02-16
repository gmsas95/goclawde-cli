package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gmsas95/myrai-cli/internal/agent"
	"github.com/gmsas95/myrai-cli/internal/api"
	"github.com/gmsas95/myrai-cli/internal/channels/discord"
	"github.com/gmsas95/myrai-cli/internal/channels/telegram"
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/cron"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/mcp"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/vector"
	"github.com/gmsas95/myrai-cli/pkg/tools"
	"go.uber.org/zap"
)

type App struct {
	Config         *config.Config
	Store          *store.Store
	Logger         *zap.Logger
	SkillsRegistry *skills.Registry
	TelegramBot    *telegram.Bot
	DiscordBot     *discord.Bot
	CronRunner     *cron.Runner
	PersonaManager *persona.PersonaManager
	Version        string
}

func New(cfg *config.Config, st *store.Store, logger *zap.Logger, pm *persona.PersonaManager, version string) *App {
	return &App{
		Config:         cfg,
		Store:          st,
		Logger:         logger,
		PersonaManager: pm,
		Version:        version,
	}
}

func (app *App) SetSkillsRegistry(registry *skills.Registry) {
	app.SkillsRegistry = registry
}

func (app *App) RunServer() {
	provider, err := app.Config.DefaultProvider()
	if err != nil {
		app.Logger.Fatal("Failed to get LLM provider", zap.Error(err))
	}
	llmClient := llm.NewClient(provider)

	agentInstance := agent.New(llmClient, nil, app.Store, app.Logger, app.PersonaManager)
	agentInstance.SetSkillsRegistry(app.SkillsRegistry)

	agentLoop := agent.NewAgentLoop(agentInstance, app.Logger)
	agentInstance.SetAgentLoop(agentLoop)

	var contextManager *agent.ContextManager
	if app.Config.Vector.Enabled {
		vectorSearcher, err := vector.NewSearcher(&app.Config.Vector, app.Store, app.Logger)
		if err != nil {
			app.Logger.Warn("Failed to create vector searcher", zap.Error(err))
		} else {
			contextManager = agent.NewContextManager(app.Store, vectorSearcher, llmClient, app.Logger)
			agentInstance.SetContextManager(contextManager)
			app.Logger.Info("Context manager initialized with vector search")
		}
	} else {
		contextManager = agent.NewContextManager(app.Store, nil, llmClient, app.Logger)
		agentInstance.SetContextManager(contextManager)
		app.Logger.Info("Context manager initialized (without vector search)")
	}

	if app.Config.Channels.Telegram.Enabled {
		telegramCfg := telegram.Config{
			Token:     app.Config.Channels.Telegram.BotToken,
			Enabled:   true,
			AllowList: app.Config.Channels.Telegram.AllowList,
		}

		go func() {
			bot, err := telegram.NewBot(telegramCfg, agentInstance, app.Store, app.Logger)
			if err != nil {
				app.Logger.Error("Failed to create Telegram bot", zap.Error(err))
				return
			}
			if err := bot.Start(); err != nil {
				app.Logger.Error("Failed to start Telegram bot", zap.Error(err))
				return
			}
			app.TelegramBot = bot
			app.Logger.Info("Telegram bot started")
		}()
	}

	if app.Config.Channels.Discord.Enabled && app.Config.Channels.Discord.Token != "" {
		discordCfg := discord.Config{
			Token:   app.Config.Channels.Discord.Token,
			Enabled: true,
			AllowDM: true,
		}

		go func() {
			db, err := discord.NewBot(discordCfg, agentInstance, app.Store, app.Logger)
			if err != nil {
				app.Logger.Error("Failed to create Discord bot", zap.Error(err))
				return
			}
			if err := db.Start(); err != nil {
				app.Logger.Error("Failed to start Discord bot", zap.Error(err))
				return
			}
			app.DiscordBot = db
			app.Logger.Info("Discord bot started")
		}()
	}

	if app.Config.MCP.Enabled {
		toolRegistry := tools.NewRegistry(app.Config.Tools.AllowedCmds)
		mcpServer := mcp.NewServer(app.Config, toolRegistry)
		go func() {
			addr := fmt.Sprintf("%s:%d", app.Config.MCP.Host, app.Config.MCP.Port)
			app.Logger.Info("Starting MCP server", zap.String("addr", addr))
			if err := mcpServer.Start(addr); err != nil {
				app.Logger.Error("MCP server error", zap.Error(err))
			}
		}()
	}

	if app.Config.Cron.Enabled {
		cronConfig := cron.Config{
			CheckInterval: app.Config.Cron.IntervalMinutes,
			MaxConcurrent: app.Config.Cron.MaxConcurrent,
		}
		app.CronRunner = cron.NewRunner(cronConfig, agentInstance, app.Store, app.Logger)
		if err := app.CronRunner.Start(); err != nil {
			app.Logger.Error("Failed to start cron runner", zap.Error(err))
		} else {
			app.Logger.Info("Cron runner started")
		}
	}

	server := api.New(app.Config, app.Store, app.Logger)
	server.SetSkillsRegistry(app.SkillsRegistry)

	go func() {
		if err := server.Start(); err != nil {
			app.Logger.Fatal("Server error", zap.Error(err))
		}
	}()

	app.Logger.Info("Server started",
		zap.String("address", app.Config.Server.Address),
		zap.Int("port", app.Config.Server.Port),
		zap.String("url", fmt.Sprintf("http://localhost:%d", app.Config.Server.Port)),
	)

	skillsList := app.SkillsRegistry.ListSkills()
	app.Logger.Info("Loaded skills", zap.Int("count", len(skillsList)))
	for _, skill := range skillsList {
		app.Logger.Info("Skill",
			zap.String("name", skill.Name()),
			zap.String("version", skill.Version()),
			zap.Int("tools", len(skill.Tools())),
		)
	}

	if app.PersonaManager != nil {
		identity := app.PersonaManager.GetIdentity()
		app.Logger.Info("Persona loaded", zap.String("name", identity.Name))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Logger.Info("Shutting down...")

	if app.TelegramBot != nil {
		app.TelegramBot.Stop()
	}

	if app.DiscordBot != nil {
		app.DiscordBot.Stop()
	}

	if app.CronRunner != nil {
		app.CronRunner.Stop()
	}

	if err := server.Shutdown(); err != nil {
		app.Logger.Error("Server shutdown error", zap.Error(err))
	}
}

func (app *App) CreateAgent() (*agent.Agent, error) {
	provider, err := app.Config.DefaultProvider()
	if err != nil {
		return nil, err
	}
	llmClient := llm.NewClient(provider)

	agentInstance := agent.New(llmClient, nil, app.Store, app.Logger, app.PersonaManager)
	agentInstance.SetSkillsRegistry(app.SkillsRegistry)

	return agentInstance, nil
}

func (app *App) RunCLI(message string) {
	agentInstance, err := app.CreateAgent()
	if err != nil {
		app.Logger.Fatal("Failed to create agent", zap.Error(err))
	}

	if message != "" {
		OneShot(agentInstance, message)
		return
	}

	Interactive(agentInstance)
}

func OneShot(agentInstance *agent.Agent, msg string) {
	fmt.Println("🤖 Myrai is thinking...")
	fmt.Println()

	ctx := context.Background()
	resp, err := agentInstance.Chat(ctx, agent.ChatRequest{
		Message: msg,
		Stream:  false,
	})

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(resp.Content)
	fmt.Printf("\n⏱️  Response time: %v | Tokens: %d\n", resp.ResponseTime, resp.TokensUsed)
}

func Interactive(agentInstance *agent.Agent) {
	fmt.Println("🤖 Myrai - Interactive Mode")
	fmt.Println("Type 'exit' or 'quit' to exit, 'help' for commands")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("👤 You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("👋 Goodbye!")
			return
		case "help", "h":
			PrintInteractiveHelp()
			continue
		case "new", "n":
			fmt.Println("🆕 New conversation started")
			continue
		case "clear", "cls":
			fmt.Print("\033[H\033[2J")
			continue
		}

		fmt.Println()
		fmt.Print("🤖 Myrai: ")

		var fullResponse strings.Builder
		start := time.Now()

		resp, err := agentInstance.Chat(ctx, agent.ChatRequest{
			Message: input,
			Stream:  true,
			OnStream: func(chunk string) {
				fmt.Print(chunk)
				fullResponse.WriteString(chunk)
			},
		})

		if err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
			continue
		}

		fmt.Println()
		fmt.Printf("\n⏱️  Response time: %v | Tokens: %d\n", time.Since(start), resp.TokensUsed)
		fmt.Println()
	}
}

func PrintInteractiveHelp() {
	fmt.Println()
	fmt.Println("Interactive Commands:")
	fmt.Println("  help, h     - Show this help")
	fmt.Println("  new, n      - Start new conversation")
	fmt.Println("  clear, cls  - Clear screen")
	fmt.Println("  exit, quit  - Exit the program")
	fmt.Println()
}
