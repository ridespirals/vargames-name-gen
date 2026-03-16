package main

import (
	"fmt"
	"log"

	"vargames-name-gen/src/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v (set %s and %s, e.g. from .env)", err, config.EnvClientID, config.EnvClientSecret)
	}
	_ = cfg // use cfg.ClientID, cfg.ClientSecret, cfg.BaseURL for API calls
	fmt.Printf("Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n", cfg.BaseURL)
}
