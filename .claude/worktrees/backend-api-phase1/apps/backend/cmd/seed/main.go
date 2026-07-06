package main

import (
	"log"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/seed"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	gdb, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(gdb); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	if err := seed.Run(gdb); err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}
	log.Println("Seed complete")
}
