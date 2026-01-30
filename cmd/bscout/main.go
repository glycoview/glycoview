package main

import (
	"log"

	"github.com/better-monitoring/bscout/internal/app"
	"github.com/better-monitoring/bscout/internal/config"
)

func main() {
	cfg, err := config.LoadConfig()

	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	app := app.NewServer(cfg)

	log.Fatal(app.Listen(":" + cfg.PORT))
}
