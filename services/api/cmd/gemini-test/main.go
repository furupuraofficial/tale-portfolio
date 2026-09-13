package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"tale-backend/internal/ar"
	"tale-backend/internal/lesson"
	"tale-backend/internal/realtime"

	"github.com/gordonklaus/portaudio"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	fmt.Println("=== Zashiki-warashi Voice Guide System (Gemini Live API) ===")
	fmt.Println()

	// Initialize PortAudio
	if err := portaudio.Initialize(); err != nil {
		log.Fatalf("Failed to initialize PortAudio: %v", err)
	}
	defer portaudio.Terminate()

	// Load rules database
	if err := realtime.LoadRules("configs/rules.json"); err != nil {
		log.Fatalf("Failed to load rules: %v", err)
	}

	// Connect to PostgreSQL (optional - only if DATABASE_URL is set)
	var lessonRepo *lesson.Repository
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
		} else {
			defer pool.Close()
			fmt.Println("Connected to PostgreSQL")

			// Setup lesson repository
			lessonRepo = lesson.NewRepository(pool)
			fmt.Println("Lesson repository ready")
		}
	} else {
		log.Println("DATABASE_URL not set, lesson API disabled")
	}
	_ = lessonRepo // TODO: integrate with Gemini client

	// Create AR Server
	arServer := ar.NewServer()

	// Create Gemini client
	client, err := realtime.NewGeminiClient()
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	// Start AR server in background
	go func() {
		if err := arServer.Start(8080); err != nil {
			log.Printf("AR Server error: %v", err)
		}
	}()

	// Connect Gemini client to AR server
	arServer.ConnectGeminiClient(client)

	// Connect to Gemini Live API
	fmt.Println("Connecting to Gemini Live API...")
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected!")
	fmt.Println()

	// Start audio stream
	fmt.Println("Starting audio stream...")
	if err := client.StartAudioStream(); err != nil {
		log.Fatalf("Failed to start audio stream: %v", err)
	}

	// Enable mic and set to free conversation mode
	client.SetMicEnabled(true)
	client.SetMode(realtime.ModeFree)
	client.SetCurrentStep(realtime.StepFreeQuestion)

	fmt.Println("Ready! Speak into the microphone...")
	fmt.Println("AR Server running on :8080")
	fmt.Println("WebSocket endpoint: ws://localhost:8080/ws")
	fmt.Println()
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	fmt.Println("\n\nExiting...")
	fmt.Println("Goodbye!")
}
