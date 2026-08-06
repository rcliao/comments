package tui

// Round tracking (Phase A of docs/design-markdown-render.md): signoff
// timestamps partition thread activity into review rounds so the TUI can
// answer "what changed since my last pass". Pure functions over the
// document's review history — no model state involved.

import (
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// lastSignoffTime returns the timestamp of the newest signoff in the review
// history. With no reviews it returns the zero time, so every thread counts
// as new activity (nothing has been signed off yet).
func lastSignoffTime(reviews []comment.ReviewRecord) time.Time {
	var latest time.Time
	for _, r := range reviews {
		if r.Timestamp.After(latest) {
			latest = r.Timestamp
		}
	}
	return latest
}

// threadHasNewActivity reports whether a thread has activity newer than the
// given signoff time: the root or any nested reply (walked recursively via
// LatestTimestamp) posted after `since`.
func threadHasNewActivity(t *comment.Comment, since time.Time) bool {
	return t.LatestTimestamp().After(since)
}

// anyNewActivity reports whether any of the threads has new activity
func anyNewActivity(threads []*comment.Comment, since time.Time) bool {
	for _, t := range threads {
		if threadHasNewActivity(t, since) {
			return true
		}
	}
	return false
}

// roundNumber returns the review round a timestamp belongs to:
// 1 + the number of signoffs strictly before it. Round 1 is everything
// before the first signoff; each signoff opens the next round.
func roundNumber(ts time.Time, reviews []comment.ReviewRecord) int {
	n := 1
	for _, r := range reviews {
		if r.Timestamp.Before(ts) {
			n++
		}
	}
	return n
}
