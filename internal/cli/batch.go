package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gmsas95/myrai-cli/internal/agent"
	"github.com/gmsas95/myrai-cli/internal/app"
	"github.com/gmsas95/myrai-cli/internal/batch"
	"github.com/gmsas95/myrai-cli/internal/config"
	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/gmsas95/myrai-cli/internal/persona"
	"github.com/gmsas95/myrai-cli/internal/skills"
	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
)

func HandleBatchCommand(args []string) {
	if len(args) == 0 {
		PrintBatchHelp()
		return
	}

	inputFile := ""
	outputFile := ""
	concurrency := 3
	timeout := 60
	tier := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-i", "--input":
			if i+1 < len(args) {
				inputFile = args[i+1]
				i++
			}
		case "-o", "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		case "-c", "--concurrency":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &concurrency)
				i++
			}
		case "-t", "--timeout":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &timeout)
				i++
			}
		case "--tier":
			if i+1 < len(args) {
				tier = args[i+1]
				i++
			}
		case "-h", "--help":
			PrintBatchHelp()
			return
		}
	}

	if inputFile == "" {
		fmt.Println("Error: Input file is required")
		fmt.Println("Usage: myrai batch -i <input_file> [-o <output_file>]")
		os.Exit(1)
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Printf("Error: Input file not found: %s\n", inputFile)
		os.Exit(1)
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load("", "")
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

	workspacePath := cfg.Storage.DataDir
	pm, err := persona.NewPersonaManager(workspacePath, logger)
	if err != nil {
		logger.Warn("Failed to initialize persona manager", zap.Error(err))
	}

	provider, err := cfg.DefaultProvider()
	if err != nil {
		fmt.Printf("Error getting LLM provider: %v\n", err)
		os.Exit(1)
	}
	llmClient := llm.NewClient(provider)

	skillsRegistry := skills.NewRegistry(st)
	app.RegisterSkills(cfg, st, skillsRegistry, logger, llmClient)

	agentInstance := agent.New(llmClient, nil, st, logger, pm)
	agentInstance.SetSkillsRegistry(skillsRegistry)

	batchConfig := batch.Config{
		MaxConcurrency: concurrency,
		Timeout:        time.Duration(timeout) * time.Second,
		RetryCount:     2,
		RetryDelay:     1 * time.Second,
		SkipInvalid:    true,
		ValidateInput:  true,
	}

	baseProcessor := batch.NewProcessor(agentInstance, batchConfig, logger)

	var result *batch.Result

	ctx := context.Background()

	if tier != "" {
		var rlConfig batch.RateLimiterConfig
		switch tier {
		case "3":
			rlConfig = batch.Tier3Config()
			fmt.Println("⚡ Using Tier 3 rate limits: 200 concurrent, 5000 RPM, 3M TPM")
		case "4":
			rlConfig = batch.Tier4Config()
			fmt.Println("⚡ Using Tier 4 rate limits: 400 concurrent, 5000 RPM, 4M TPM")
		case "5":
			rlConfig = batch.Tier5Config()
			fmt.Println("⚡ Using Tier 5 rate limits: 1000 concurrent, 10000 RPM, 5M TPM")
		default:
			fmt.Printf("Unknown tier: %s. Using default limits.\n", tier)
			rlConfig = batch.RateLimiterConfig{MaxConcurrency: concurrency}
		}

		processor := batch.NewRateLimitedProcessor(baseProcessor, rlConfig)
		fmt.Printf("🤖 Processing batch file: %s\n", inputFile)
		fmt.Printf("   Concurrency: %d | Timeout: %ds | Tier: %s\n", rlConfig.MaxConcurrency, timeout, tier)
		fmt.Println()

		result, err = processor.ProcessFileWithRateLimit(ctx, inputFile, outputFile)
	} else {
		fmt.Printf("🤖 Processing batch file: %s\n", inputFile)
		fmt.Printf("   Concurrency: %d | Timeout: %ds\n", concurrency, timeout)
		fmt.Println()

		result, err = baseProcessor.ProcessFile(ctx, inputFile, outputFile)
	}
	if err != nil {
		fmt.Printf("Error processing batch: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result.Summary())

	if outputFile != "" {
		fmt.Printf("✓ Results saved to: %s\n", outputFile)
	}

	if result.Failed > 0 {
		fmt.Println("\nFailed items:")
		for _, item := range result.Items {
			if !item.Success && item.Error != "skipped" {
				fmt.Printf("  - %s: %s\n", item.ID, item.Error)
			}
		}
	}
}
