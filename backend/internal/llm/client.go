package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string  `json:"role"`
	Content string  `json:"content"`
	Name    *string `json:"name,omitempty"`
	// ToolCalls must be echoed back on the assistant message that requested them,
	// otherwise the "tool" messages that follow are rejected by OpenAI-compatible APIs.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID links a "tool" message to the assistant tool call it answers.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolDefinition describes a tool to the LLM.
type ToolDefinition struct {
	Type     string         `json:"type"`
	Function FunctionDef    `json:"function"`
}

type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall represents a tool call returned by the LLM.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatResponse is the LLM response.
type ChatResponse struct {
	Content   string     `json:"-"`
	ToolCalls []ToolCall `json:"-"`
}

// Client is the LLM client interface.
type Client interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResponse, error)
}

// OpenAIClient implements Client for OpenAI-compatible APIs (Codex, GPT, etc.).
type OpenAIClient struct {
	apiKey   string
	baseURL  string
	model    string
	client   *http.Client
}

func NewClient(apiKey, baseURL, model string) Client {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResponse, error) {
	body := map[string]any{
		"model":    c.model,
		"messages": messages,
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[LLM] API error %d: %s", resp.StatusCode, truncate(string(respBody), 500))
		return nil, fmt.Errorf("llm error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[LLM] parse error: %v, body: %s", err, truncate(string(respBody), 500))
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in llm response")
	}

	msg := result.Choices[0].Message
	finishReason := result.Choices[0].FinishReason

	if len(tools) > 0 {
		log.Printf("[LLM] response: finish_reason=%s, tool_calls=%d, content_len=%d",
			finishReason, len(msg.ToolCalls), len(msg.Content))
		if len(msg.ToolCalls) > 0 {
			log.Printf("[LLM] tool_call: %s(%s)", msg.ToolCalls[0].Function.Name,
				truncate(msg.ToolCalls[0].Function.Arguments, 100))
		}
		if msg.Content != "" {
			log.Printf("[LLM] content preview: %s", truncate(msg.Content, 120))
		}
	}

	return &ChatResponse{
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	}, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + fmt.Sprintf("...(%d chars)", len(s))
}
