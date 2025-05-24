package server

import (
	"fmt"
	"github.com/glanceapp/glance/pkg/widgets"
	"os"
)

type config struct {
	AppName     string `yaml:"app-name"`
	Host        string `yaml:"host"`
	Port        uint16 `yaml:"port"`
	Proxied     bool   `yaml:"proxied"`
	AssetsPath  string `yaml:"assets-path"`
	BaseURL     string `yaml:"base-url"`
	FaviconURL  string `yaml:"favicon-url"`
	FaviconType string `yaml:"favicon-type"`
}

func newConfig() *config {
	return &config{
		AppName:    "Pulse",
		Host:       "localhost",
		Port:       8080,
		Proxied:    false,
		AssetsPath: "./assets",
		BaseURL:    "/",
	}
}

func (c *config) Validate() error {
	if c.AssetsPath != "" {
		if _, err := os.Stat(c.AssetsPath); os.IsNotExist(err) {
			return fmt.Errorf("assets directory does not exist: %s", c.AssetsPath)
		}
	}
}
