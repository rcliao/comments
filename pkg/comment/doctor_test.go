package comment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeCommentedDoc creates a markdown file plus a matching sidecar.
func writeCommentedDoc(t *testing.T, dir, name, content string) string {
	t.Helper()
	mdPath := filepath.Join(dir, name)
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	doc := &DocumentWithComments{
		Content: content,
		Threads: []*Comment{{
			ID:        "cdoc1",
			Author:    "claude",
			Line:      1,
			Text:      "[Q] why?",
			Timestamp: time.Now(),
			Replies:   []*Comment{},
		}},
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}
	return mdPath
}

// writePluginState writes a Claude Code plugin state file listing the plugins.
func writePluginState(t *testing.T, dir string, body string) string {
	t.Helper()
	path := filepath.Join(dir, "installed_plugins.json")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func findCheck(t *testing.T, r DoctorReport, name string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %q check in report %+v", name, r.Checks)
	return Check{}
}

func okProbe(context.Context, string) (MCPInfo, error) {
	return MCPInfo{ProtocolVersion: "2026-07-28", ToolCount: 20}, nil
}

func TestDoctorHealthyInstall(t *testing.T) {
	dir := t.TempDir()
	writeCommentedDoc(t, dir, "doc.md", "line one\nline two\n")
	state := writePluginState(t, dir, `{"plugins":{"comments@comments":[{"scope":"user"}]}}`)

	report := RunDoctor(context.Background(), DoctorOptions{
		Target:          dir,
		Version:         "2.2.0",
		ExePath:         "/usr/local/bin/comments",
		PluginStatePath: state,
		Probe:           okProbe,
	})

	if !report.OK() {
		t.Errorf("healthy install should pass, got %+v", report.Checks)
	}
	for _, c := range report.Checks {
		if c.Status != StatusPass {
			t.Errorf("check %q = %s (%s), want pass", c.Name, c.Status, c.Detail)
		}
	}
	if got := findCheck(t, report, "binary").Detail; got != "/usr/local/bin/comments (2.2.0)" {
		t.Errorf("binary detail = %q", got)
	}
	if got := findCheck(t, report, "mcp").Detail; got != "20 tools, protocol 2026-07-28" {
		t.Errorf("mcp detail = %q", got)
	}
}

func TestDoctorMissingBinary(t *testing.T) {
	dir := t.TempDir()
	report := RunDoctor(context.Background(), DoctorOptions{
		Target:          dir,
		ExePath:         "",
		PluginStatePath: filepath.Join(dir, "absent.json"),
		Probe:           okProbe,
	})

	if report.OK() {
		t.Error("a missing binary must fail the report")
	}
	if got := findCheck(t, report, "binary").Status; got != StatusFail {
		t.Errorf("binary check = %s, want fail", got)
	}
	// With no binary there is nothing to probe; that must fail too, not hang.
	if got := findCheck(t, report, "mcp").Status; got != StatusFail {
		t.Errorf("mcp check = %s, want fail", got)
	}
}

func TestDoctorStaleSidecar(t *testing.T) {
	dir := t.TempDir()
	mdPath := writeCommentedDoc(t, dir, "doc.md", "line one\nline two\n")
	// Edit the markdown behind the sidecar's back: the stored hash no longer
	// matches, which is exactly the staleness doctor should surface.
	if err := os.WriteFile(mdPath, []byte("line one\nline two\nline three\n"), 0644); err != nil {
		t.Fatal(err)
	}

	report := RunDoctor(context.Background(), DoctorOptions{
		Target:          dir,
		ExePath:         "/usr/local/bin/comments",
		PluginStatePath: filepath.Join(dir, "absent.json"),
		Probe:           okProbe,
	})

	sidecars := findCheck(t, report, "sidecars")
	if sidecars.Status != StatusWarn {
		t.Errorf("stale sidecar = %s (%s), want warn", sidecars.Status, sidecars.Detail)
	}
	// Staleness is recoverable, so it must not sink the exit code.
	if !report.OK() {
		t.Error("a stale sidecar is a warning, not a failure")
	}
}

func TestDoctorMCPHandshakeFailure(t *testing.T) {
	dir := t.TempDir()
	report := RunDoctor(context.Background(), DoctorOptions{
		Target:          dir,
		ExePath:         "/usr/local/bin/comments",
		PluginStatePath: filepath.Join(dir, "absent.json"),
		Probe: func(context.Context, string) (MCPInfo, error) {
			return MCPInfo{}, errors.New("connection refused")
		},
	})

	mcpCheck := findCheck(t, report, "mcp")
	if mcpCheck.Status != StatusFail {
		t.Errorf("mcp check = %s, want fail", mcpCheck.Status)
	}
	if report.OK() {
		t.Error("a failed handshake must sink the report")
	}
}

func TestDoctorPluginStateDegradesToWarning(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"malformed":    `{not json`,
		"unexpected":   `{"something_else": 1}`,
		"other-plugin": `{"plugins":{"zero@zero-plugins":[{"scope":"user"}]}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			state := writePluginState(t, t.TempDir(), body)
			report := RunDoctor(context.Background(), DoctorOptions{
				Target:          dir,
				ExePath:         "/usr/local/bin/comments",
				PluginStatePath: state,
				Probe:           okProbe,
			})
			// Another tool's undocumented file must never fail our install.
			if got := findCheck(t, report, "plugin").Status; got != StatusWarn {
				t.Errorf("plugin check = %s, want warn", got)
			}
			if !report.OK() {
				t.Error("plugin state problems must not sink the report")
			}
		})
	}
}

func TestDoctorSkipsMCPWhenNoProbe(t *testing.T) {
	dir := t.TempDir()
	report := RunDoctor(context.Background(), DoctorOptions{
		Target:          dir,
		ExePath:         "/usr/local/bin/comments",
		PluginStatePath: filepath.Join(dir, "absent.json"),
	})
	if got := findCheck(t, report, "mcp").Status; got != StatusWarn {
		t.Errorf("skipped mcp check = %s, want warn", got)
	}
	if !report.OK() {
		t.Error("skipping the mcp probe must not sink the report")
	}
}
