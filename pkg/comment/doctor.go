package comment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckStatus is the outcome of a single doctor check. Only Fail affects the
// exit code: a warning is something worth knowing, not something broken.
type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
)

// Check is one line of the doctor report.
type Check struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail"`
}

// DoctorReport is the full install health report.
type DoctorReport struct {
	Checks []Check `json:"checks"`
}

// OK reports whether the install is sound. Warnings do not sink it, so CI can
// gate on the exit code without tripping over cosmetic notes.
func (r DoctorReport) OK() bool {
	for _, c := range r.Checks {
		if c.Status == StatusFail {
			return false
		}
	}
	return true
}

// MCPInfo is what a successful probe of `comments serve-mcp` learned.
type MCPInfo struct {
	ProtocolVersion string `json:"protocol_version,omitempty"`
	ToolCount       int    `json:"tool_count"`
}

// MCPProbe connects to the MCP server exposed by exePath and reports what it
// found. It is injected rather than implemented here because pkg/comment must
// not depend on the MCP adapter — see docs/ARCHITECTURE.md decision 8.
type MCPProbe func(ctx context.Context, exePath string) (MCPInfo, error)

// DoctorOptions configures a doctor run. Everything the checks need is passed
// in, so the whole report is reproducible in tests without touching the real
// environment.
type DoctorOptions struct {
	// Target is the file or directory scanned for sidecars.
	Target string
	// Version is the version string of the running binary.
	Version string
	// ExePath is the binary to report and to probe for MCP.
	ExePath string
	// PluginStatePath overrides the Claude Code plugin state file.
	PluginStatePath string
	// Probe runs the MCP handshake; nil skips the mcp check.
	Probe MCPProbe
}

// RunDoctor evaluates install health and returns one check per subsystem.
func RunDoctor(ctx context.Context, opts DoctorOptions) DoctorReport {
	return DoctorReport{Checks: []Check{
		checkBinary(opts),
		checkMCP(ctx, opts),
		checkPlugin(opts),
		checkSidecars(opts),
	}}
}

func checkBinary(opts DoctorOptions) Check {
	if opts.ExePath == "" {
		return Check{"binary", StatusFail, "could not resolve the running binary's path"}
	}
	version := opts.Version
	if version == "" || version == "dev" {
		// A source build is a sound install; it just cannot name a release.
		return Check{"binary", StatusPass, fmt.Sprintf("%s (dev build, no version stamped)", opts.ExePath)}
	}
	return Check{"binary", StatusPass, fmt.Sprintf("%s (%s)", opts.ExePath, version)}
}

func checkMCP(ctx context.Context, opts DoctorOptions) Check {
	if opts.Probe == nil {
		return Check{"mcp", StatusWarn, "skipped"}
	}
	if opts.ExePath == "" {
		return Check{"mcp", StatusFail, "no binary to probe"}
	}
	info, err := opts.Probe(ctx, opts.ExePath)
	if err != nil {
		return Check{"mcp", StatusFail, fmt.Sprintf("handshake failed: %v", err)}
	}
	detail := fmt.Sprintf("%d tools", info.ToolCount)
	if info.ProtocolVersion != "" {
		detail += fmt.Sprintf(", protocol %s", info.ProtocolVersion)
	}
	if info.ToolCount == 0 {
		return Check{"mcp", StatusFail, "server answered but advertised no tools"}
	}
	return Check{"mcp", StatusPass, detail}
}

// pluginStateFile is the Claude Code install record. It belongs to another
// tool and is undocumented, so every unexpected shape degrades to a warning
// rather than failing an otherwise healthy install.
func pluginStateFile(opts DoctorOptions) string {
	if opts.PluginStatePath != "" {
		return opts.PluginStatePath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
}

func checkPlugin(opts DoctorOptions) Check {
	path := pluginStateFile(opts)
	if path == "" {
		return Check{"plugin", StatusWarn, "unknown: could not resolve the home directory"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Check{"plugin", StatusWarn, "not installed as a Claude Code plugin (no plugin state file)"}
		}
		return Check{"plugin", StatusWarn, fmt.Sprintf("unknown: %v", err)}
	}

	var state struct {
		Plugins map[string]json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return Check{"plugin", StatusWarn, "unknown: plugin state file is not in the expected shape"}
	}
	for name := range state.Plugins {
		if name == "comments" || strings.HasPrefix(name, "comments@") {
			return Check{"plugin", StatusPass, fmt.Sprintf("installed as %s", name)}
		}
	}
	return Check{"plugin", StatusWarn, "not installed as a Claude Code plugin"}
}

func checkSidecars(opts DoctorOptions) Check {
	target := opts.Target
	if target == "" {
		target = "."
	}
	files, err := FindGateTargets(target)
	if err != nil {
		return Check{"sidecars", StatusFail, fmt.Sprintf("cannot scan %s: %v", target, err)}
	}
	if len(files) == 0 {
		return Check{"sidecars", StatusPass, fmt.Sprintf("no commented documents under %s", target)}
	}

	var stale, unreadable, orphans int
	for _, file := range files {
		doc, report, err := LoadFromSidecar(file)
		if err != nil {
			unreadable++
			continue
		}
		if report.Stale {
			stale++
		}
		for _, c := range doc.GetAllComments() {
			if c.IsOrphaned() {
				orphans++
			}
		}
	}

	detail := fmt.Sprintf("%d document(s)", len(files))
	if stale > 0 {
		detail += fmt.Sprintf(", %d stale", stale)
	}
	if orphans > 0 {
		detail += fmt.Sprintf(", %d orphaned comment(s)", orphans)
	}
	if unreadable > 0 {
		detail += fmt.Sprintf(", %d unreadable", unreadable)
		return Check{"sidecars", StatusFail, detail}
	}
	if stale > 0 || orphans > 0 {
		// Recoverable: re-anchoring on load handles staleness, and orphans are
		// a prompt to run `comments reanchor`, not a broken install.
		return Check{"sidecars", StatusWarn, detail}
	}
	return Check{"sidecars", StatusPass, detail}
}
