package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/gmsas95/myrai-cli/internal/app"
	"github.com/gmsas95/myrai-cli/internal/cli"
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/onboarding"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "", "Path to config file")
	dataDir    = flag.String("data", "", "Path to data directory")
	cliMode    = flag.Bool("cli", false, "Run in CLI mode (one-shot or interactive)")
	message    = flag.String("m", "", "Message to send (CLI mode)")
	serverMode = flag.Bool("server", false, "Run in server mode")
	onboard    = flag.Bool("onboard", false, "Run onboarding wizard")
	version    = "dev"
)

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
			application := initApp()
			cli.HandleGatewayCommand(os.Args[2:], application)
			return
		case "status":
			cli.HandleStatusCommand()
			return
		case "doctor":
			cli.HandleDoctorCommand()
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

	application := initApp()

	if *cliMode || *message != "" {
		application.RunCLI(*message)
		return
	}

	application.RunServer()
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

func initApp() *app.App {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Myrai",
		zap.String("version", version),
		zap.String("mode", getMode()),
	)

	cfg, err := config.Load(*configPath, *dataDir)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

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

	skillsRegistry := skills.NewRegistry(st)
	app.RegisterSkills(cfg, st, skillsRegistry, logger)

	application := app.New(cfg, st, logger, pm, version)
	application.SetSkillsRegistry(skillsRegistry)

	return application
}

func getMode() string {
	if *cliMode || *message != "" {
		return "cli"
	}
	return "server"
}
