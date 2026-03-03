package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/gmsas95/myrai-cli/internal/app"
	"github.com/gmsas95/myrai-cli/internal/cli"
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/jobs"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/onboarding"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"github.com/gmsas95/myrai-cli/internal/tui"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "", "Path to config file")
	dataDir    = flag.String("data", "", "Path to data directory")
	cliMode    = flag.Bool("cli", false, "Run in CLI mode (one-shot or interactive)")
	tuiMode    = flag.Bool("tui", false, "Run in beautiful TUI mode")
	message    = flag.String("m", "", "Message to send (CLI mode)")
	serverMode = flag.Bool("server", false, "Run in server mode")
	onboard    = flag.Bool("onboard", false, "Run onboarding wizard")
	version    = "dev"
)

// AppContext holds the application context for graceful shutdown
type AppContext struct {
	App         *app.App
	JobRegistry *jobs.Registry
	Logger      *zap.Logger
	Store       *store.Store
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "onboard":
			runOnboarding()
			return
		case "project":
			cli.HandleProjectCommand(os.Args[2:])
			return
		case "persona":
			cli.HandlePersonaCommand(os.Args[2:])
			return
		case "user":
			cli.HandleUserCommand(os.Args[2:])
			return
		case "batch":
			cli.HandleBatchCommand(os.Args[2:])
			return
		case "config":
			cli.HandleConfigCommand(os.Args[2:])
			return
		case "skills":
			cli.HandleSkillsCommand(os.Args[2:])
			return
		case "channels":
			cli.HandleChannelsCommand(os.Args[2:])
			return
		case "gateway":
			if len(os.Args) > 2 && (os.Args[2] == "-h" || os.Args[2] == "--help" || os.Args[2] == "help") {
				cli.PrintGatewayHelp()
				return
			}
			appCtx := initAppWithGracefulShutdown()
			cli.HandleGatewayCommand(os.Args[2:], appCtx.App)
			return
		case "status":
			cli.HandleStatusCommand()
			return
		case "doctor":
			cli.HandleDoctorCommand()
			return
		case "memory":
			cli.HandleNeuralCommand(os.Args[2:])
			return
		case "chain":
			cli.HandleChainCommand(os.Args[2:])
			return
		case "tools":
			cli.HandleToolsInventoryCommand(os.Args[2:])
			return
		case "intent":
			cli.HandleIntentCommand(os.Args[2:])
			return
		case "marketplace":
			cli.HandleMarketplaceCommand(os.Args[2:])
			return
		case "job":
			handleJobCommand(os.Args[2:])
			return
		case "help", "--help", "-h":
			cli.PrintExtendedHelp()
			return
		case "version", "--version", "-v":
			fmt.Printf("Myrai version %s\n", version)
			return
		}
	}

	flag.Parse()

	if onboarding.CheckFirstRun() && !*onboard && term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("🤖 Welcome to Myrai!")
		fmt.Println()
		fmt.Println("It looks like this is your first time running Myrai.")
		fmt.Println("Let's set up your personal AI assistant.")
		fmt.Println()
		fmt.Print("Run onboarding wizard? (Y/n): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			runOnboarding()
			return
		}
	}

	if *onboard {
		runOnboarding()
		return
	}

	appCtx := initAppWithGracefulShutdown()

	if *tuiMode {
		// Run beautiful TUI mode
		agentInstance, err := appCtx.App.CreateAgent()
		if err != nil {
			appCtx.Logger.Fatal("Failed to create agent", zap.Error(err))
		}

		if err := tui.Run(agentInstance); err != nil {
			appCtx.Logger.Fatal("TUI error", zap.Error(err))
		}
		shutdown(appCtx)
		return
	}

	if *cliMode || *message != "" {
		appCtx.App.RunCLI(*message)
		shutdown(appCtx)
		return
	}

	// Server mode with graceful shutdown
	runServerWithGracefulShutdown(appCtx)
}

func runOnboarding() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	wizard := onboarding.NewWizard(logger)
	if err := wizard.Run(); err != nil {
		fmt.Printf("\n❌ Onboarding failed: %v\n", err)
		os.Exit(1)
	}
}

func initAppWithGracefulShutdown() *AppContext {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Info("Starting Myrai",
		zap.String("version", version),
		zap.String("mode", getMode()),
	)

	cfg, err := config.Load(*configPath, *dataDir)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Run security checks and warnings
	runSecurityChecks(cfg, logger)

	st, err := store.New(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize store", zap.Error(err))
	}

	workspacePath := cfg.Storage.DataDir
	pm, err := persona.NewPersonaManager(workspacePath, logger)
	if err != nil {
		logger.Warn("Failed to initialize persona manager", zap.Error(err))
		pm = nil
	}

	// Create LLM client for vision skill
	var llmClient *llm.Client
	provider, err := cfg.DefaultProvider()
	if err != nil {
		logger.Warn("Failed to get LLM provider", zap.Error(err))
	} else {
		llmClient = llm.NewClient(provider)
	}

	skillsRegistry := skills.NewRegistry(st)
	app.RegisterSkills(cfg, st, skillsRegistry, logger, llmClient)

	application := app.New(cfg, st, logger, pm, version)
	application.SetSkillsRegistry(skillsRegistry)

	// Initialize job registry
	var jobRegistry *jobs.Registry
	if cfg.Cron.Enabled {
		jobRegistry, err = jobs.NewRegistry(st, cfg, logger)
		if err != nil {
			logger.Warn("Failed to initialize job registry", zap.Error(err))
		} else {
			if err := jobRegistry.Initialize(); err != nil {
				logger.Warn("Failed to initialize jobs", zap.Error(err))
			}
		}
	}

	return &AppContext{
		App:         application,
		JobRegistry: jobRegistry,
		Logger:      logger,
		Store:       st,
	}
}

func runServerWithGracefulShutdown(appCtx *AppContext) {
	// Start job scheduler if available
	if appCtx.JobRegistry != nil {
		appCtx.JobRegistry.Start()
		appCtx.Logger.Info("Background job scheduler started")
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Run server in goroutine
	serverDoneCh := make(chan struct{})
	go func() {
		appCtx.App.RunServer()
		close(serverDoneCh)
	}()

	// Wait for shutdown signal or server to exit
	select {
	case sig := <-sigCh:
		appCtx.Logger.Info("Received shutdown signal",
			zap.String("signal", sig.String()),
		)
	case <-serverDoneCh:
		appCtx.Logger.Info("Server exited")
	}

	// Graceful shutdown
	shutdown(appCtx)
}

func shutdown(appCtx *AppContext) {
	appCtx.Logger.Info("Shutting down gracefully...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop job scheduler
	if appCtx.JobRegistry != nil {
		appCtx.Logger.Info("Stopping job scheduler...")
		appCtx.JobRegistry.Stop()

		// Wait for jobs to complete
		if err := appCtx.JobRegistry.Wait(shutdownCtx); err != nil {
			appCtx.Logger.Warn("Timeout waiting for jobs to complete", zap.Error(err))
		}
	}

	// Close store
	if appCtx.Store != nil {
		appCtx.Logger.Info("Closing store...")
		appCtx.Store.Close()
	}

	// Sync logger
	appCtx.Logger.Sync()

	appCtx.Logger.Info("Shutdown complete")
}

func handleJobCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: myrai job <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  list              List all scheduled jobs")
		fmt.Println("  run <job-id>      Run a job immediately")
		fmt.Println("  enable <job-id>   Enable a job")
		fmt.Println("  disable <job-id>  Disable a job")
		return
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load(*configPath, *dataDir)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	st, err := store.New(cfg)
	if err != nil {
		fmt.Printf("Error initializing store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	jobRegistry, err := jobs.NewRegistry(st, cfg, logger)
	if err != nil {
		fmt.Printf("Error initializing job registry: %v\n", err)
		os.Exit(1)
	}

	if err := jobRegistry.Initialize(); err != nil {
		fmt.Printf("Error initializing jobs: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		jobList := jobRegistry.ListJobs()
		if len(jobList) == 0 {
			fmt.Println("No jobs registered")
			return
		}
		fmt.Printf("%-30s %-20s %-10s %s\n", "ID", "Name", "Status", "Next Run")
		fmt.Println(strings.Repeat("-", 80))
		for _, job := range jobList {
			status := "disabled"
			if job.Enabled {
				status = "enabled"
			}
			nextRun := "N/A"
			if job.NextRun != nil {
				nextRun = job.NextRun.Format("2006-01-02 15:04")
			}
			fmt.Printf("%-30s %-20s %-10s %s\n", job.ID, job.Name, status, nextRun)
		}

	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: myrai job run <job-id>")
			os.Exit(1)
		}
		jobID := args[1]
		fmt.Printf("Triggering job: %s\n", jobID)
		if err := jobRegistry.ManualJobTrigger(jobID); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Job triggered successfully")

	case "enable":
		if len(args) < 2 {
			fmt.Println("Usage: myrai job enable <job-id>")
			os.Exit(1)
		}
		jobID := args[1]
		if err := jobRegistry.GetScheduler().EnableJob(jobID); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Job %s enabled\n", jobID)

	case "disable":
		if len(args) < 2 {
			fmt.Println("Usage: myrai job disable <job-id>")
			os.Exit(1)
		}
		jobID := args[1]
		if err := jobRegistry.GetScheduler().DisableJob(jobID); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Job %s disabled\n", jobID)

	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		os.Exit(1)
	}
}

func getMode() string {
	if *cliMode || *message != "" {
		return "cli"
	}
	return "server"
}

func runSecurityChecks(cfg *config.Config, logger *zap.Logger) {
	// Check if admin password is configured
	if cfg.Security.AdminPassword == "" {
		logger.Info("Security: No admin password configured")
		logger.Info("  - Dashboard is open (relying on network-level security)")
		logger.Info("  - To enable password auth: Set GOCLAWDE_ADMIN_PASSWORD")
		logger.Info("  - Recommended: Use reverse proxy with HTTPS or VPN")
	} else {
		logger.Info("Security: Admin password configured")
	}

	// Check CORS configuration
	if len(cfg.Security.AllowOrigins) == 1 && cfg.Security.AllowOrigins[0] == "*" {
		logger.Warn("Security: CORS is set to allow all origins (*)")
		logger.Warn("  - This is suitable for local development only")
		logger.Warn("  - For production, set GOCLAWDE_SECURITY_ALLOW_ORIGINS to your domain")
	}

	// Check server binding
	if cfg.Server.Address == "0.0.0.0" {
		logger.Warn("Security: Server is bound to all interfaces (0.0.0.0)")
		logger.Warn("  - Accessible from any network interface")
		logger.Warn("  - Recommended: Use reverse proxy with HTTPS")
	} else if cfg.Server.Address == "127.0.0.1" || cfg.Server.Address == "localhost" {
		logger.Info("Security: Server is bound to localhost only")
		logger.Info("  - Only accessible from this machine")
	}

	// Check JWT secret
	if cfg.Security.JWTSecret == "" {
		logger.Warn("Security: JWT secret not configured, using random value")
		logger.Warn("  - Sessions won't persist across restarts")
		logger.Warn("  - Set GOCLAWDE_JWT_SECRET for persistent sessions")
	}

	// Check if running as root
	if os.Getuid() == 0 {
		logger.Warn("Security: Running as root is not recommended")
		logger.Warn("  - Consider running as a non-root user")
	}
}
