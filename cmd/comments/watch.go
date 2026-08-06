package main

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// watchCommand polls sidecars for a file or directory and emits review-state
// change events as NDJSON on stdout. The sidecar is the shared event bus:
// every writer (TUI, CLI, MCP, agents) persists actions there immediately,
// so one file watcher observes them all.
func watchCommand(target string, args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	interval := fs.Duration("interval", time.Second, "Poll interval")
	until := fs.String("until", "", "Exit 0 after emitting an event matching this comma-separated list of event types (e.g. signoff,gate_changed)")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	encoder := json.NewEncoder(os.Stdout)

	type watched struct {
		mtime time.Time
		snap  comment.WatchSnapshot
	}
	state := map[string]watched{}

	// emit writes events as NDJSON; it reports stop=true when the watcher
	// should exit cleanly (exit 0): the consumer closed stdout (EPIPE), or an
	// event matched --until.
	emit := func(events []comment.WatchEvent) (stop bool) {
		for _, e := range events {
			if err := encoder.Encode(e); err != nil {
				// stdout is gone (EPIPE: consumer exited). Exit cleanly instead
				// of lingering as an orphan process writing into a broken pipe.
				return true
			}
			if comment.MatchesUntil(e.Event, *until) {
				return true
			}
		}
		return false
	}

	for {
		files, err := comment.FindGateTargets(target)
		if err != nil {
			return failf("Error: %v", err)
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
			if seen && emit(comment.DiffSnapshots(file, prev.snap, snap)) {
				return nil
			}
			state[file] = watched{mtime: info.ModTime(), snap: snap}
		}
		time.Sleep(*interval)
	}
}
