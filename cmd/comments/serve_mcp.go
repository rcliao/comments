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
	log.Println("Available tools:")
	log.Println("  Read: comments_list, comments_get, comments_status")
	log.Println("  Write: comments_add, comments_reply, comments_resolve")
	log.Println("  Suggestions: comments_suggest, comments_accept, comments_reject")
	log.Println("  Batch: comments_batch_add, comments_batch_reply")
	log.Println("")

	if err := server.Serve(ctx); err != nil {
		return failf("MCP server error: %v", err)
	}
	return nil
}
