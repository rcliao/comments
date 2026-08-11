package comment

import (
	"encoding/json"
	"os"
	"testing"
)

// The plugin-version constant compiled into the binary must track the plugin
// manifest, or doctor's drift check would itself drift.
func TestPluginVersionMatchesManifest(t *testing.T) {
	data, err := os.ReadFile("../../.claude-plugin/plugin.json")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != PluginVersion {
		t.Fatalf("PluginVersion const %q != .claude-plugin/plugin.json %q — bump them together",
			PluginVersion, manifest.Version)
	}
}
