package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rcliao/comments/pkg/webreview"
)

func serveCommand(target string, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:0", "loopback address to listen on")
	author := fs.String("author", defaultReviewAuthor(), "reviewer name used for comments and verdicts")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}
	if fs.NArg() != 0 {
		return failf("Usage: comments serve <file-or-dir> [--addr 127.0.0.1:0] [--author name]")
	}
	if err := validateLoopbackAddr(*addr); err != nil {
		return failf("Error: %v", err)
	}

	review, err := webreview.New(target, *author)
	if err != nil {
		return failf("Error starting review server: %v", err)
	}
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return failf("Error listening on %s: %v", *addr, err)
	}
	defer func() { _ = listener.Close() }()
	review.ConfigureListener(listener)

	httpServer := &http.Server{
		Handler:           review,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(listener) }()

	fmt.Printf("Comments review is ready for %s\n", target)
	fmt.Printf("Open: %s\n", review.BootstrapURL(listener))
	fmt.Println("Press Ctrl-C to stop. The server is available only on this computer.")

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			// SSE connections are intentionally long-lived and may outlast the
			// graceful window. Ctrl-C still means stop now, so close remaining
			// connections instead of turning a clean user exit into an error.
			_ = httpServer.Close()
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return failf("Review server stopped: %v", err)
	}
}

func validateLoopbackAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("--addr must be host:port: %w", err)
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("--addr must use a loopback host (127.0.0.1, ::1, or localhost)")
	}
	return nil
}

func defaultReviewAuthor() string {
	for _, name := range []string{os.Getenv("COMMENTS_AUTHOR"), os.Getenv("USER"), os.Getenv("USERNAME")} {
		if strings.TrimSpace(name) != "" {
			return name
		}
	}
	return "Reviewer"
}
