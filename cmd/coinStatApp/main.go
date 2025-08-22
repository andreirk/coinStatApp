package main

import (
	"coinStatApp/config"
	"coinStatApp/internal/app"
	"coinStatApp/internal/app/dto"
	"coinStatApp/internal/handlers/http"
	"coinStatApp/internal/lib/logger/handlers/slogpretty"
	"coinStatApp/pkg/utils"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.LoadConfig()
	log := setupLogger(cfg.Env)
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Info("Shutting down...")
		cancel()
	}()

	// Initialize app
	log.Info("Initializing app...")

	app, err := app.NewApp(ctx, log, cfg)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to initialize app: %v", err))
	}

	// Start event processor
	log.Info("Starting event processor...")
	go app.EventProcessor.Run(ctx)

	// !!! For DEMO purposes, generate random swaps
	// This is not for production use!
	swapGenerator := utils.NewSwapGenerator()
	go func() {
		// Generate 1000 swaps every second
		log.Info("Starting swap generator...")
		for ctx.Err() == nil {
			// Generate a random swap and send it to the channel
			swaps := swapGenerator.GenerateRandomSwap(100)
			swapDtos := dto.FromModels(swaps)
			app.KafkaProducer.PublishSwapBatch(ctx, swapDtos)
			//app.EventProcessor.SwapCh <- swapDto
			time.Sleep(100 * time.Millisecond)
		}
		log.Info("Swap generator stopped")
	}()
	// !!! End of DEMO swap generation

	// Set up HTTP server
	httpAddr := fmt.Sprintf(":%s", app.Config.HTTPPort)
	httpServer := http.NewServer(httpAddr, app.StatsService, app.Broadcaster)

	// Start HTTP server in a goroutine
	go func() {
		log.Info(fmt.Sprintf("HTTP server listening on %s", httpAddr))
		if err := httpServer.Start(); err != nil {
			log.Info(fmt.Sprintf("HTTP server error: %v", err))
		}
	}()

	// Wait for termination signal
	<-ctx.Done()

	// Clean up app resources
	log.Info("Cleaning up app resources...")
	app.Cleanup(ctx)

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server with timeout
	log.Info("Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Info("HTTP server shutdown error: %v", err)
	}

	// Add a small delay to allow shutdown handlers to complete
	timer := time.NewTimer(500 * time.Millisecond)
	select {
	case <-timer.C:
	case <-shutdownCtx.Done():
	}

	log.Info("Service stopped.")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
