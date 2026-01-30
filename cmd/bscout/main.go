package main

import (
	"log"

	"github.com/better-monitoring/bscout/bootstrap"
	"github.com/better-monitoring/bscout/pkg/config"
)

func main() {
	cfg, err := config.LoadConfig()

	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	app := bootstrap.Bootstrap(cfg)
	log.Fatal(app.Listen(":" + cfg.PORT))
}
