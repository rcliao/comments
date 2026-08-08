package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/mcp"
)

// version is stamped at release time via -X main.version (see .goreleaser.yaml).
// A plain `go build` leaves it as "dev", which doctor reports as a source build.
var version = "dev"

// statusMark renders a check status for the terminal.
func statusMark(s comment.CheckStatus) string {
	switch s {
	case comment.StatusPass:
		return "✔"
	case comment.StatusWarn:
		return "⚠"
	default:
		return "✗"
	}
}

// doctorCommand reports install health: binary, MCP server, plugin state and
// sidecars. Exit 0 when nothing failed (warnings included), 1 otherwise.
func doctorCommand(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	skipMCP := fs.Bool("skip-mcp", false, "Skip the MCP handshake (avoids spawning a subprocess)")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	target := "."
	if rest := fs.Args(); len(rest) > 0 {
		target = rest[0]
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = ""
	}

	opts := comment.DoctorOptions{
		Target:  target,
		Version: version,
		ExePath: exePath,
		Probe:   mcp.Probe,
	}
	if *skipMCP {
		opts.Probe = nil
	}

	report := comment.RunDoctor(context.Background(), opts)

	if *jsonOut {
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
	} else {
		fmt.Printf("comments doctor — %s\n\n", target)
		for _, c := range report.Checks {
			fmt.Printf("  %s %-10s %s\n", statusMark(c.Status), c.Name, c.Detail)
		}
		fmt.Println()
		if report.OK() {
			fmt.Println("Install looks sound.")
		} else {
			fmt.Println("Install has problems. Fix the ✗ lines above.")
			fmt.Println("If the binary is missing or stale: go install github.com/rcliao/comments/cmd/comments@latest")
		}
	}

	if !report.OK() {
		return exitSilent(1)
	}
	return nil
}
