package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// ProbeTimeout bounds the doctor's MCP handshake so a wedged server reports a
// failed check instead of hanging the command.
const ProbeTimeout = 10 * time.Second

// Probe starts `<exePath> serve-mcp`, completes a handshake and counts the
// advertised tools. It uses the SDK client, which tries the stateless
// server/discover (SEP-2575) first and falls back to the legacy initialize
// handshake, so the probe follows whatever the server actually supports.
//
// It satisfies comment.MCPProbe and is injected into the doctor run: the check
// logic lives in pkg/comment and must not import this package.
func Probe(ctx context.Context, exePath string) (comment.MCPInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "comments-doctor",
		Version: ServerVersion,
	}, nil)

	transport := &mcp.CommandTransport{Command: exec.CommandContext(ctx, exePath, "serve-mcp")}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return comment.MCPInfo{}, fmt.Errorf("connect: %w", err)
	}
	// The probe only reads; a close failure tells us nothing about install health
	defer func() { _ = session.Close() }()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		return comment.MCPInfo{}, fmt.Errorf("tools/list: %w", err)
	}

	info := comment.MCPInfo{ToolCount: len(tools.Tools)}
	if res := session.InitializeResult(); res != nil {
		info.ProtocolVersion = res.ProtocolVersion
	}
	return info, nil
}
