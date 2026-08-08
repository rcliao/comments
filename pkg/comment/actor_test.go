package comment

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/term"
)

func TestResolveActor(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		isTTY bool
		want  Actor
	}{
		{"tty means human", "", true, ActorHuman},
		{"no tty means agent", "", false, ActorAgent},
		{"env forces agent at a terminal", "agent", true, ActorAgent},
		{"env forces human without a tty", "human", false, ActorHuman},
		{"env is case insensitive", "HUMAN", false, ActorHuman},
		{"unknown env falls back to tty", "robot", true, ActorHuman},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(ActorEnvVar, tc.env)
			if got := ResolveActor(tc.isTTY); got != tc.want {
				t.Errorf("ResolveActor(%v) with %s=%q = %q, want %q",
					tc.isTTY, ActorEnvVar, tc.env, got, tc.want)
			}
		})
	}
}

// Regression: /dev/null is a character device, so an os.ModeCharDevice check
// reported `cmd >/dev/null` as an interactive terminal — which let an agent
// resolve human-zone threads simply by discarding output. Neither a discarded
// stream nor a redirected file is a terminal.
func TestStdoutIsTTYRejectsNonTerminals(t *testing.T) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = devNull.Close() }()

	regular, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = regular.Close() }()

	for name, f := range map[string]*os.File{"devnull": devNull, "regular file": regular} {
		if term.IsTerminal(f.Fd()) {
			t.Errorf("%s must not be reported as a terminal", name)
		}
	}
}

// humanZoneDoc builds a design-doc-shaped document with one thread in the
// human-zone "Problem" section and one in the agent-writable "Proposed Design".
func humanZoneDoc() (*DocumentWithComments, string, string) {
	content := strings.Join([]string{
		"# Title",
		"",
		"## Problem",
		"",
		"Something is broken.",
		"",
		"## Proposed Design",
		"",
		"Fix it.",
		"",
	}, "\n")

	inProblem := &Comment{ID: "chuman", Line: 5, Author: "template", Text: "[Q] why?", Timestamp: time.Now()}
	inDesign := &Comment{ID: "cagent", Line: 9, Author: "template", Text: "[Q] how?", Timestamp: time.Now()}

	return &DocumentWithComments{
		Content:  content,
		Template: "design-doc",
		Threads:  []*Comment{inProblem, inDesign},
	}, inProblem.ID, inDesign.ID
}

func TestGuardZoneResolve(t *testing.T) {
	doc, humanThread, agentThread := humanZoneDoc()
	path := t.TempDir() + "/doc.md"

	t.Run("agent refused in human zone", func(t *testing.T) {
		err := GuardZoneResolve(doc, path, humanThread, ActorAgent)
		if err == nil {
			t.Fatal("expected refusal for agent resolving a human-zone thread")
		}
		if !strings.Contains(err.Error(), "human-decision zone") {
			t.Errorf("unexpected error text: %v", err)
		}
	})

	t.Run("human allowed in human zone", func(t *testing.T) {
		if err := GuardZoneResolve(doc, path, humanThread, ActorHuman); err != nil {
			t.Errorf("human must never be blocked, got: %v", err)
		}
	})

	t.Run("agent allowed outside human zone", func(t *testing.T) {
		if err := GuardZoneResolve(doc, path, agentThread, ActorAgent); err != nil {
			t.Errorf("agent must be allowed in a writable zone, got: %v", err)
		}
	})

	t.Run("no template means no zones", func(t *testing.T) {
		untemplated := &DocumentWithComments{Content: doc.Content, Threads: doc.Threads}
		if err := GuardZoneResolve(untemplated, path, humanThread, ActorAgent); err != nil {
			t.Errorf("a doc without a template has no zones, got: %v", err)
		}
	})

	t.Run("unknown thread defers to caller", func(t *testing.T) {
		if err := GuardZoneResolve(doc, path, "cnope", ActorAgent); err != nil {
			t.Errorf("missing thread must not be reported as a zone violation, got: %v", err)
		}
	})
}
