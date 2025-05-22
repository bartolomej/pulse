package main

import (
	"fmt"
	"github.com/glanceapp/glance/pkg/server"
	"log"
)

func main() {
	configPath := "./config/root.yml"
	err := run(configPath)
	if err != nil {
		log.Fatalf("Failed to run: %v", err)
	}
}

func run(configPath string) (err error) {
	configContents, _, err := server.ParseYAMLIncludes(configPath)
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	cfg, err := server.NewConfigFromYAML(configContents)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	app, err := server.NewApplication(cfg)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	startServer, _ := app.Server()

	if err := startServer(); err != nil {
		log.Printf("Failed to start server: %v", err)
	}

	return nil
}
