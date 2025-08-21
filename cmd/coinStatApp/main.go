package main

import (
	"coinStatApp/config"
	"coinStatApp/internal/app"
	"coinStatApp/internal/app/dto"
	"coinStatApp/pkg/utils"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coinStatApp/internal/handlers/http"
)

func main() {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("Shutting down...")
		cancel()
	}()

	// Initialize app
	log.Println("Initializing app...")
	cfg := config.LoadConfig()
	app, err := app.NewApp(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Start event processor
	log.Println("Starting event processor...")
	go app.EventProcessor.Run(ctx)

	// !!! For demonstration purposes, generate random swaps
	// This is not for production use!
	swapGenerator := utils.NewSwapGenerator()
	go func() {
		// Generate 1000 swaps every second
		log.Println("Starting swap generator...")
		for ctx.Err() == nil {
			// Generate a random swap and send it to the channel
			swaps := swapGenerator.GenerateRandomSwap(100)
			swapDtos := dto.FromModels(swaps)
			app.KafkaProducer.PublishSwapBatch(ctx, swapDtos)
			//app.EventProcessor.SwapCh <- swapDto
			time.Sleep(100 * time.Millisecond)
		}
		log.Println("Swap generator stopped")
	}()
	// !!! End of swap generation

	// Set up HTTP server
	httpAddr := fmt.Sprintf(":%s", app.Config.HTTPPort)
	httpServer := http.NewServer(httpAddr, app.StatsService, app.Broadcaster)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("HTTP server listening on %s", httpAddr)
		if err := httpServer.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for termination signal
	<-ctx.Done()

	// Clean up app resources
	log.Println("Cleaning up app resources...")
	app.Cleanup(ctx)

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server with timeout
	log.Println("Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Add a small delay to allow shutdown handlers to complete
	timer := time.NewTimer(500 * time.Millisecond)
	select {
	case <-timer.C:
	case <-shutdownCtx.Done():
	}

	log.Println("Service stopped.")
}
