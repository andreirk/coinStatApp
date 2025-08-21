package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"coinStatApp/config"
	"coinStatApp/internal/domain/model"
)

func TestMain(m *testing.M) {
	log.Println("Running integration tests...")

	// Run all tests
	code := m.Run()

	log.Println("Integration tests completed.")

	// Return test status code
	if code != 0 {
		log.Println("Tests failed.")
	}
	os.Exit(code)
}

// TestHealthEndpoint tests the /health endpoint
func TestHealthEndpoint(t *testing.T) {
	// Skip if running in CI environment or with short flag
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up a test HTTP client
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make a request to the health endpoint
	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("Failed to make request to health endpoint: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Decode response
	var healthResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check response content
	if status, ok := healthResponse["status"]; !ok || status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", status)
	}
}

// TestStatsEndpoint tests the /stats endpoint
func TestStatsEndpoint(t *testing.T) {
	// Skip if running in CI environment or with short flag
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up a test HTTP client
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Give some time for stats to accumulate
	time.Sleep(1 * time.Second)

	// Make a request to the stats endpoint
	resp, err := client.Get("http://localhost:8080/stats")
	if err != nil {
		t.Fatalf("Failed to make request to stats endpoint: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Decode response (we only check if it's valid JSON)
	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode stats response: %v", err)
	}

	// Verify that we received some statistics
	if len(stats) == 0 {
		t.Errorf("Expected non-empty stats, got empty response")
	}
}

// TestGenerateSwaps verifies the swap generation function
func TestGenerateSwaps(t *testing.T) {
	swaps := generateSwaps(100)

	if len(swaps) != 100 {
		t.Errorf("Expected 100 swaps, got %d", len(swaps))
	}

	// Check that swaps have required fields
	for i, swap := range swaps {
		if swap.ID == "" {
			t.Errorf("Swap at index %d has empty ID", i)
		}
		if swap.Token == "" {
			t.Errorf("Swap at index %d has empty Token", i)
		}
		if swap.Side != "buy" && swap.Side != "sell" {
			t.Errorf("Swap at index %d has invalid Side: %s", i, swap.Side)
		}
		if swap.Timestamp.IsZero() {
			t.Errorf("Swap at index %d has zero Timestamp", i)
		}
	}
}

// TestAppInitialization ensures the app initializes correctly
func TestAppInitialization(t *testing.T) {
	// This test verifies that the app can initialize its components

	// Create a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Load configuration
	cfg := config.LoadConfig()

	if cfg == nil {
		t.Fatal("Failed to load configuration")
	}

	// Check that required config values are set
	if cfg.HTTPPort == "" {
		t.Error("HTTPPort not set in configuration")
	}

	if cfg.RedisAddr == "" {
		t.Error("RedisAddr not set in configuration")
	}
}
