package main

import (
	"fmt"
	"github.com/glanceapp/glance/pkg/server"
	"log"
)

func main() {
	err := run()
	if err != nil {
		log.Fatalf("Failed to run: %v", err)
	}
}

func run() (err error) {
	app, err := server.NewApplication()
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	startServer, _ := app.Server()

	if err := startServer(); err != nil {
		log.Printf("Failed to start server: %v", err)
	}

	return nil
}
