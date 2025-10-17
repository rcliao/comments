package llm

import (
	"context"

	"github.com/rcliao/comments/pkg/comment"
)

// Provider represents an LLM service provider
type Provider interface {
	// Complete generates a completion based on the context
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// Stream generates a streaming completion
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)

	// Name returns the provider name
	Name() string
}

// CompletionRequest represents a request for LLM completion
type CompletionRequest struct {
	// Document content for context
	DocumentContent string

	// Existing comments for context
	Comments []*comment.Comment

	// User's prompt/question
	Prompt string

	// Optional: specific line range for context
	ContextStartLine int
	ContextEndLine   int

	// Optional: specific comment to respond to
	ParentCommentID string

	// Model to use (provider-specific)
	Model string

	// Temperature for randomness (0.0 - 1.0)
	Temperature float64

	// Max tokens to generate
	MaxTokens int
}

// CompletionResponse represents the LLM's response
type CompletionResponse struct {
	// Generated text
	Text string

	// Suggested comment placement (line number)
	SuggestedLine int

	// Whether this should be a comment or an edit
	Type ResponseType

	// Metadata
	Model         string
	TokensUsed    int
	FinishReason  string
}

// StreamChunk represents a piece of streamed response
type StreamChunk struct {
	// Delta text (incremental)
	Delta string

	// Whether this is the final chunk
	Done bool

	// Error if any
	Error error
}

// ResponseType indicates how the LLM response should be used
type ResponseType string

const (
	ResponseTypeComment    ResponseType = "comment"     // Add as comment
	ResponseTypeEdit       ResponseType = "edit"        // Suggest as edit
	ResponseTypeSuggestion ResponseType = "suggestion"  // General suggestion
)

// ContextBuilder builds context for LLM requests
type ContextBuilder struct{}

// BuildContext creates a context string for the LLM
func (cb *ContextBuilder) BuildContext(doc string, comments []*comment.Comment, startLine, endLine int) string {
	// TODO: Implement smart context building
	// - Include relevant document sections
	// - Include related comments
	// - Format for optimal LLM understanding
	return doc
}
