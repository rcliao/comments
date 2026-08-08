package comment

import (
	"fmt"
	"os"
	"strings"
)

// Actor is who is driving a mutation. Template zones marked `zone: human`
// reserve decisions for the human, so the guard needs to know which one is
// calling — a question that used to be answered implicitly by the surface
// (MCP meant agent, CLI meant human). That assumption broke as soon as agents
// started driving the CLI, which SKILL.md explicitly blesses.
type Actor string

const (
	ActorHuman Actor = "human"
	ActorAgent Actor = "agent"
)

// ActorEnvVar forces the actor regardless of how the process was invoked.
const ActorEnvVar = "COMMENTS_ACTOR"

// ResolveActor decides who is calling. COMMENTS_ACTOR wins when set to a known
// value; otherwise an interactive terminal means a human and a piped or
// redirected stream means an agent, since agents run without a TTY.
func ResolveActor(isTTY bool) Actor {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(ActorEnvVar))) {
	case string(ActorHuman):
		return ActorHuman
	case string(ActorAgent):
		return ActorAgent
	}
	if isTTY {
		return ActorHuman
	}
	return ActorAgent
}

// StdoutIsTTY reports whether stdout is attached to a terminal.
func StdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

// GuardZoneResolve refuses an agent's attempt to resolve a thread that sits in
// a template section marked `zone: human`. Humans are always allowed, and a
// document with no template recorded has no zones to enforce.
//
// Both the CLI and the MCP server call this, so the guard cannot be sidestepped
// by choosing a different surface.
func GuardZoneResolve(doc *DocumentWithComments, absPath, threadID string, actor Actor) error {
	if actor == ActorHuman || doc.Template == "" {
		return nil
	}
	thread := doc.FindThreadByID(threadID)
	if thread == nil {
		return nil // let the caller's own not-found error report this
	}
	t, err := LoadTemplateForDoc(doc.Template, absPath)
	if err != nil {
		return nil // an unreadable template must not block legitimate work
	}
	if SectionZone(doc.Content, t, thread.Line) != ZoneHuman {
		return nil
	}
	return fmt.Errorf(
		"thread %s is in a human-decision zone (template %q); reply with your input instead — the human resolves it in the TUI, or via 'comments resolve' at a terminal (set %s=human to override)",
		threadID, doc.Template, ActorEnvVar)
}
