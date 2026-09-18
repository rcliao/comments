package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/mcp"
)

// cliOnly is every command that deliberately has no MCP tool, with the reason.
// A new command must land here or gain an MCP twin — there is no third option,
// which is what keeps the surface from growing a second path for one purpose.
var cliOnly = map[string]string{
	"view":      "human review surface; the verdict and suggestion decisions are made here and must not be agent-reachable",
	"serve":     "human review surface, in a browser",
	"template":  "for humans authoring templates; agents get the brief inside `context`",
	"gate":      "exit-code contract for scripts and CI; agents read the same decision from `inbox`",
	"doctor":    "install health",
	"bundle":    "index maintenance; `new` already refreshes indexes",
	"serve-mcp": "starts the MCP server",
	"help":      "usage",
}

// humanOnly decisions must never come back as commands or tools: a scripted
// signoff once let an agent record an approval under a human's name, and an
// unguarded accept let one rewrite a zone: human section through its own
// suggestion (docs/review-surface-e2e-2026-09-18.md).
var humanOnly = []string{"signoff", "accept", "reject", "batch-accept"}

func dispatchCommands(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	start := strings.Index(body, "switch command {")
	if start < 0 {
		t.Fatal("could not find the dispatch switch in main.go")
	}
	end := strings.Index(body[start:], "default:")
	if end < 0 {
		t.Fatal("could not find the end of the dispatch switch in main.go")
	}
	var names []string
	for _, m := range regexp.MustCompile(`"([a-z-]+)"`).FindAllStringSubmatch(
		strings.Join(regexp.MustCompile(`(?m)^\tcase [^\n]*`).FindAllString(body[start:start+end], -1), "\n"), -1) {
		if !strings.HasPrefix(m[1], "-") {
			names = append(names, m[1])
		}
	}
	sort.Strings(names)
	return names
}

// TestEveryMCPToolHasACLITwinAndViceVersa pins the rule the surface is built
// on: one command per purpose, the same on both surfaces. It reads the MCP
// catalog from registration and the CLI catalog from the dispatch switch, so
// neither side is a hand-maintained list.
func TestEveryMCPToolHasACLITwinAndViceVersa(t *testing.T) {
	commands := map[string]bool{}
	for _, name := range dispatchCommands(t) {
		commands[name] = true
	}

	twins := map[string]bool{}
	for _, tool := range mcp.NewServer().ToolNames() {
		twin := strings.ReplaceAll(strings.TrimPrefix(tool, "comments_"), "_", "-")
		twins[twin] = true
		if !commands[twin] {
			t.Errorf("MCP tool %s has no CLI command %q", tool, twin)
		}
		if reason, ok := cliOnly[twin]; ok {
			t.Errorf("%q is listed as CLI-only (%s) but has an MCP tool", twin, reason)
		}
	}
	for name := range commands {
		if !twins[name] && cliOnly[name] == "" {
			t.Errorf("CLI command %q has no MCP tool and no recorded reason in cliOnly", name)
		}
	}
	for name := range cliOnly {
		if !commands[name] {
			t.Errorf("cliOnly lists %q, which is not a command", name)
		}
	}
	for _, name := range humanOnly {
		if commands[name] || twins[name] {
			t.Errorf("%q is a human decision and must not exist as a command or tool", name)
		}
	}
}

// The help text is the one catalog still written by hand; every command must
// appear in it, and no removed command may linger.
func TestUsageTextNamesEveryCommand(t *testing.T) {
	for _, name := range dispatchCommands(t) {
		if !regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(name) + `\b`).MatchString(usageText) {
			t.Errorf("usage text has no entry for %q", name)
		}
	}
	for _, gone := range append([]string{"list", "resolve", "status", "batch-add", "batch-reply", "check-review"}, humanOnly...) {
		if regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(gone) + ` `).MatchString(usageText) {
			t.Errorf("usage text still documents removed command %q", gone)
		}
	}
}
