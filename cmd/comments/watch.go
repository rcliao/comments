package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// watchCommand polls sidecars for a file or directory and emits review-state
// change events as NDJSON on stdout. The sidecar is the shared event bus:
// every writer (TUI, CLI, MCP, agents) persists actions there immediately,
// so one file watcher observes them all.
func watchCommand(target string, args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	interval := fs.Duration("interval", time.Second, "Poll interval")
	until := fs.String("until", "", "Exit 0 after emitting an event matching this comma-separated list of event types (e.g. signoff,gate_changed)")
	fs.Parse(args)

	encoder := json.NewEncoder(os.Stdout)

	type watched struct {
		mtime time.Time
		snap  comment.WatchSnapshot
	}
	state := map[string]watched{}

	emit := func(events []comment.WatchEvent) {
		for _, e := range events {
			if err := encoder.Encode(e); err != nil {
				// stdout is gone (EPIPE: consumer exited). Exit cleanly instead
				// of lingering as an orphan process writing into a broken pipe.
				os.Exit(0)
			}
			if comment.MatchesUntil(e.Event, *until) {
				os.Exit(0)
			}
		}
	}

	for {
		files, err := comment.FindGateTargets(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, file := range files {
			info, err := os.Stat(comment.GetSidecarPath(file))
			if err != nil {
				continue
			}
			prev, seen := state[file]
			if seen && !info.ModTime().After(prev.mtime) {
				continue
			}
			snap := comment.TakeSnapshot(file)
			if seen {
				emit(comment.DiffSnapshots(file, prev.snap, snap))
			}
			state[file] = watched{mtime: info.ModTime(), snap: snap}
		}
		time.Sleep(*interval)
	}
}
