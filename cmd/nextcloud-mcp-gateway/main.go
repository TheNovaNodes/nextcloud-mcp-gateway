package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/caldav"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/config"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/deck"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/hitl"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/ocs"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/server"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/webdav"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var (
	Version   = "1.0.0"
	CommitSHA = "dev"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v" || os.Args[1] == "version") {
		fmt.Printf("nextcloud-mcp-gateway version %s (commit %s)\n", Version, CommitSHA)
		return
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}

	if cfg.Username == "" || cfg.Password == "" {
		fmt.Fprintln(os.Stderr, "❌ CRITICAL: Nextcloud credentials (NC_USER, NC_APP_PASSWORD) missing in Vault/Env. Aborting.")
		os.Exit(1)
	}

	davCli := webdav.NewClient(cfg)
	caldavCli := caldav.NewClient(cfg)
	deckCli := deck.NewClient(cfg)
	ocsCli := ocs.NewClient(cfg)
	hitlMgr := hitl.NewManager(hitl.DefaultMaxPending, hitl.DefaultTTL)

	srv := server.NewServer(davCli, caldavCli, deckCli, ocsCli, hitlMgr)

	if err := mcpserver.ServeStdio(srv.MCPServer()); err != nil {
		log.Fatalf("MCP Server terminated with error: %v", err)
	}
}
