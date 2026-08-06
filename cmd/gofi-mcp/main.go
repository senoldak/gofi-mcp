package main

import (
	"context"
	"log"
	"os"

	"github.com/senoldak/gofi-mcp/internal/coingecko"
	"github.com/senoldak/gofi-mcp/internal/fred"
	"github.com/senoldak/gofi-mcp/internal/fx"
	"github.com/senoldak/gofi-mcp/internal/goficlient"
	"github.com/senoldak/gofi-mcp/internal/mcp"
	"github.com/senoldak/gofi-mcp/internal/sec"
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

	if ua := os.Getenv("SEC_USER_AGENT"); ua != "" {
		for _, t := range mcp.SECTools(sec.New(ua)) {
			registry.Add(t)
		}
	}
	if key := os.Getenv("FRED_API_KEY"); key != "" {
		registry.Add(mcp.MacroIndicatorTool(fred.New(key)))
	}
	for _, t := range mcp.FXTools(fx.New()) {
		registry.Add(t)
	}
	for _, t := range mcp.CryptoTools(coingecko.New(os.Getenv("COINGECKO_API_KEY"))) {
		registry.Add(t)
	}

	server := mcp.NewServer(registry, os.Stdin, os.Stdout)

	log.Printf("gofi-mcp %s ready, GOFI URL: %s", mcp.ServerVersion, baseURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		log.Fatalf("gofi-mcp: %v", err)
	}
}
