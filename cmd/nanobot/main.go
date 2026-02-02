package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/YOUR_USERNAME/nanobot/internal/api"
	"github.com/YOUR_USERNAME/nanobot/internal/config"
	"github.com/YOUR_USERNAME/nanobot/internal/store"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "", "Path to config file")
	dataDir    = flag.String("data", "", "Path to data directory")
	version    = "dev"
)

func main() {
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting nanobot",
		zap.String("version", version),
	)

	// Load configuration
	cfg, err := config.Load(*configPath, *dataDir)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize data store
	db, err := store.New(cfg.DataDir)
	if err != nil {
		logger.Fatal("Failed to initialize store", zap.Error(err))
	}
	defer db.Close()

	// Initialize and start API server
	server := api.New(cfg, db, logger)
	
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	logger.Info("Server started", 
		zap.String("address", cfg.Server.Address),
		zap.Int("port", cfg.Server.Port),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")
	
	if err := server.Shutdown(); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
}
