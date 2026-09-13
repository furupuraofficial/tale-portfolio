package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"tale-backend/internal/api"
	"tale-backend/internal/ar"
	"tale-backend/internal/lesson"
	"tale-backend/internal/realtime"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	fmt.Println("=== TALE Voice Guide (OpenAI Realtime API) ===")

	if err := realtime.LoadRules("configs/rules.json"); err != nil {
		log.Fatalf("Failed to load rules: %v", err)
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
		} else {
			defer pool.Close()
			lessonRepo := lesson.NewRepository(pool)
			api.RegisterLessonAPI(lessonRepo)
			fmt.Println("Lesson API registered at /lessons")
		}
	} else {
		log.Println("DATABASE_URL not set, lesson API disabled")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}

	arServer := ar.NewServer()
	client, err := realtime.NewClient(apiKey)
	if err != nil {
		log.Fatalf("Failed to create Realtime client: %v", err)
	}
	defer client.Close()

	api.SetRealtimeClient(client)
	api.RegisterSwiftAPI(arServer, client)

	go func() {
		if err := arServer.Start(8080); err != nil {
			log.Printf("AR Server error: %v", err)
		}
	}()

	arServer.ConnectRealtimeClient(client)
	client.LoadHistory()

	fmt.Println("Connecting to Realtime API...")
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected. Waiting for /conversation/start...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	if err := client.SaveHistory(); err != nil {
		log.Printf("Warning: failed to save history: %v", err)
	}

	msgCount, turnCount := client.GetHistoryStats()
	fmt.Printf("Conversation stats: %d messages (%d turns)\n", msgCount, turnCount)
}
