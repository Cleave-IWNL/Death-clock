package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	tgClient "death-clock/clients/telegram"
	event_consumer "death-clock/consumer/event-consumer"
	"death-clock/events/telegram"
	"death-clock/storage/sqlite"

	_ "github.com/mattn/go-sqlite3"
)

const (
	tgBotHost      = "api.telegram.org"
	storageSqlPath = "data/sqlite/storage.db"
	batchSize      = 100
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  .env file not found, reading environment variables directly")
	}

	token := mustToken()

	s, err := sqlite.New(storageSqlPath)
	if err != nil {
		log.Fatalf("can't connect to the storage: %s", err)
	}

	eventsProcessor := telegram.New(
		tgClient.New(tgBotHost, token),
		s,
	)

	if err := s.InitSchema(context.Background()); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}

	log.Print("service started")

	consumer := event_consumer.New(eventsProcessor, eventsProcessor, batchSize)

	if err := consumer.Start(); err != nil {
		log.Fatal("service is stopped", err)
	}
}

func mustToken() string {
	token := os.Getenv("TG_BOT_TOKEN")
	if token == "" {
		log.Fatal("TG_BOT_TOKEN is not set in environment")
	}
	return token
}
