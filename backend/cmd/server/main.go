package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-native-sre-agent/backend/internal/config"
	"github.com/ai-native-sre-agent/backend/internal/server"
	"github.com/ai-native-sre-agent/backend/internal/tools"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	registry := tools.NewRegistry()
	// Diagnostic tools (read-only, always safe)
	registry.Register(tools.DockerStatsTool{})
	registry.Register(tools.DockerLogsTool{})
	registry.Register(tools.DockerPsTool{})
	registry.Register(tools.PrometheusQueryTool{})
	// Action tools (modify state, respect dry-run)
	registry.Register(tools.DockerRestartTool{})
	registry.Register(tools.DockerUpdateLimitTool{})
	registry.Register(tools.DockerPruneTool{})
	registry.Register(tools.DockerExecTool{})
	registry.Register(tools.ServiceRestartTool{})
	registry.Register(tools.ActionTool{})

	srv := server.New(cfg, registry)

	r := gin.Default()
	srv.Register(r)

	// Start alerter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartAlerter(ctx)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	addr := ":" + cfg.ServerPort
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
