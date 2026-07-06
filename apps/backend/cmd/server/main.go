package main

import (
	"log"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/router"

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

	r := router.New(cfg, gdb)
	log.Printf("Backend listening on http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
