package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// ClaudeProvider implements Provider for Anthropic's Claude API
type ClaudeProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(apiKey string) *ClaudeProvider {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return &ClaudeProvider{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		baseURL:    "https://api.anthropic.com/v1",
	}
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// Complete generates a completion
func (p *ClaudeProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Build the prompt
	prompt := p.buildPrompt(req)

	// Prepare API request
	apiReq := map[string]interface{}{
		"model": p.getModel(req.Model),
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": p.getMaxTokens(req.MaxTokens),
	}

	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make API call
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return &CompletionResponse{
		Text:         apiResp.Content[0].Text,
		Model:        apiResp.Model,
		TokensUsed:   apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		FinishReason: apiResp.StopReason,
		Type:         ResponseTypeComment,
	}, nil
}

// Stream generates a streaming completion
func (p *ClaudeProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	// TODO: Implement streaming support
	// For now, fall back to non-streaming
	chunks := make(chan StreamChunk, 1)
	go func() {
		defer close(chunks)
		resp, err := p.Complete(ctx, req)
		if err != nil {
			chunks <- StreamChunk{Error: err, Done: true}
			return
		}
		chunks <- StreamChunk{Delta: resp.Text, Done: true}
	}()
	return chunks, nil
}

// buildPrompt constructs the prompt for Claude
func (p *ClaudeProvider) buildPrompt(req CompletionRequest) string {
	var prompt strings.Builder

	prompt.WriteString("You are collaborating on a document using inline comments.\n\n")

	// Add document context
	if req.ContextStartLine > 0 && req.ContextEndLine > 0 {
		prompt.WriteString(fmt.Sprintf("Document context (lines %d-%d):\n", req.ContextStartLine, req.ContextEndLine))
		prompt.WriteString("```\n")
		prompt.WriteString(p.extractLineRange(req.DocumentContent, req.ContextStartLine, req.ContextEndLine))
		prompt.WriteString("\n```\n\n")
	} else {
		prompt.WriteString("Full document:\n```\n")
		prompt.WriteString(req.DocumentContent)
		prompt.WriteString("\n```\n\n")
	}

	// Add existing comments context
	if len(req.Comments) > 0 {
		prompt.WriteString("Existing comments:\n")
		for _, c := range req.Comments {
			prompt.WriteString(fmt.Sprintf("- Line %d (@%s): %s\n", c.Line, c.Author, c.Text))
		}
		prompt.WriteString("\n")
	}

	// Add user's prompt
	prompt.WriteString("User's request:\n")
	prompt.WriteString(req.Prompt)
	prompt.WriteString("\n\n")

	prompt.WriteString("Please provide your response as a comment that will be added to the document. ")
	prompt.WriteString("Be concise, helpful, and constructive.")

	return prompt.String()
}

// extractLineRange extracts a specific range of lines from content
func (p *ClaudeProvider) extractLineRange(content string, start, end int) string {
	lines := strings.Split(content, "\n")
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// getModel returns the model to use, with defaults
func (p *ClaudeProvider) getModel(model string) string {
	if model == "" {
		return "claude-3-5-sonnet-20241022"
	}
	return model
}

// getMaxTokens returns max tokens with default
func (p *ClaudeProvider) getMaxTokens(maxTokens int) int {
	if maxTokens == 0 {
		return 1024
	}
	return maxTokens
}
