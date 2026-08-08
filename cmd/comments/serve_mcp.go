package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rcliao/comments/pkg/mcp"
)

// serveMCPCommand starts the MCP server over stdio
func serveMCPCommand() error {
	// Keep the advertised MCP version in step with the binary
	mcp.ServerVersion = version

	// Create context that listens for interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Create and start the MCP server
	server := mcp.NewServer()

	log.Println("Comments MCP server starting...")
	log.Println("Server name: comments-mcp-server")
	log.Println("Version: 1.0.0")
	log.Println("")
	log.Println("Available resources:")
	log.Println("  - comments://doc/{filepath}")
	log.Println("  - comments://thread/{filepath}/{thread_id}")
	log.Println("")
	// Printed from the registration list itself — a hand-maintained banner had
	// already drifted several tools behind reality.
	names := server.ToolNames()
	log.Printf("Available tools (%d):", len(names))
	for _, name := range names {
		log.Printf("  %s", name)
	}
	log.Println("")

	if err := server.Serve(ctx); err != nil {
		return failf("MCP server error: %v", err)
	}
	return nil
}
