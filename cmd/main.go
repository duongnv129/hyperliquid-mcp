// cmd/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/duongnv129/hyperliquid-mcp/internal/config"
	"github.com/duongnv129/hyperliquid-mcp/internal/hlclient"
	"github.com/duongnv129/hyperliquid-mcp/internal/tools"
)

func main() {
	transport := flag.String("transport", "stdio", "transport to serve on: stdio or http")
	addr := flag.String("addr", ":8080", "address to listen on when --transport=http")
	flag.Parse()

	if err := run(*transport, *addr); err != nil {
		fmt.Fprintln(os.Stderr, "hyperliquid-mcp:", err)
		os.Exit(1)
	}
}

func run(transport, addr string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	client, err := hlclient.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to hyperliquid: %w", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "hyperliquid-mcp", Version: "0.1.0"}, nil)
	tools.RegisterAccountTools(server, client)
	tools.RegisterMarketTools(server, client)
	tools.RegisterQueryTools(server, client)
	tools.RegisterOrderTools(server, client)
	tools.RegisterVaultTools(server, client)
	tools.RegisterMiscTools(server, client)

	switch transport {
	case "stdio":
		return server.Run(ctx, &mcp.StdioTransport{})
	case "http":
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
		log.Printf("hyperliquid-mcp listening on %s (http transport)", addr)
		return http.ListenAndServe(addr, handler)
	default:
		return fmt.Errorf("unknown transport %q (want stdio or http)", transport)
	}
}
