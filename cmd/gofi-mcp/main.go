package main

import (
	"context"
	"log"
	"os"

	"github.com/senoldak/gofi-mcp/internal/goficlient"
	"github.com/senoldak/gofi-mcp/internal/mcp"
)

const defaultURL = "http://localhost:8080"

func resolveURL(env string) string {
	if env == "" {
		return defaultURL
	}
	return env
}

func main() {
	baseURL := resolveURL(os.Getenv("GOFI_URL"))
	client := goficlient.New(baseURL)
	registry := mcp.NewRegistry(client)
	server := mcp.NewServer(registry, os.Stdin, os.Stdout)

	log.Printf("gofi-mcp %s ready, GOFI URL: %s", mcp.ServerVersion, baseURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		log.Fatalf("gofi-mcp: %v", err)
	}
}
