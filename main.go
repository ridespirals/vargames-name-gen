package main

import (
	"flag"
	"fmt"
	"log"

	"vargames-name-gen/src/config"
	"vargames-name-gen/src/igdb"
)

func main() {
	verbose := flag.Bool("verbose", false, "enable progress logging for IGDB client and fetchers")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v (set %s and %s, e.g. from .env)", err, config.EnvClientID, config.EnvClientSecret)
	}
	enableLog := *verbose || cfg.Verbose
	logger := igdb.LoggerFromVerbose(enableLog)
	_ = igdb.NewClient(cfg, igdb.WithLogger(logger))
	if enableLog {
		log.Println("verbose logging enabled (IGDB client/fetchers)")
	}
	fmt.Printf("Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n", cfg.BaseURL)
}

