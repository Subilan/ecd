package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Subilan/ecd/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
)

// CallAI sends a request to the OpenAI-compatible API with retry logic.
// It runs the API call synchronously and returns the raw response content.
func CallAI(ctx context.Context, systemPrompt, userMessage string, cfg config.AIConfig) (string, error) {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = cfg.BaseURL
	client := openai.NewClientWithConfig(clientConfig)

	req := openai.ChatCompletionRequest{
		Model: cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userMessage,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := client.CreateChatCompletion(ctx, req)
		if err != nil {
			lastErr = err
			if !shouldRetry(err) {
				return "", classifyError(err)
			}
			continue
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("AI returned no choices")
		}
		return resp.Choices[0].Message.Content, nil
	}

	return "", classifyError(lastErr)
}

// CallAIWithCache checks the cache, calls AI if needed, and caches the result.
// Returns the raw JSON response content and whether it was served from cache.
func CallAIWithCache(ctx context.Context, input, systemPrompt, userMessage string, cfg config.AIConfig, bypass bool) (string, bool, error) {
	if !bypass && cfg.CacheEnabled {
		if cached, ok := CacheGet(input); ok {
			return string(cached), true, nil
		}
	}

	response, err := CallAI(ctx, systemPrompt, userMessage, cfg)
	if err != nil {
		return "", false, err
	}

	if cfg.CacheEnabled {
		if err := CacheSet(input, []byte(response)); err != nil {
			// Cache write failure is non-fatal
			_ = err
		}
	}

	return response, false, nil
}

// ParseJSONResponse unmarshals the AI JSON response into a generic structure.
func ParseJSONResponse(response string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("unexpected AI response: %w", err)
	}
	return result, nil
}

// shouldRetry returns true if the error is transient and worth retrying.
func shouldRetry(err error) bool {
	switch e := err.(type) {
	case *openai.APIError:
		switch e.HTTPStatusCode {
		case 429, 500, 502, 503, 504:
			return true
		}
	case *openai.RequestError:
		return true
	}
	return false
}

// classifyError wraps API errors with user-friendly messages.
func classifyError(err error) error {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case *openai.APIError:
		switch e.HTTPStatusCode {
		case 401, 403:
			return fmt.Errorf("invalid API key — run /init to reconfigure")
		case 429:
			return fmt.Errorf("rate limited — try again later")
		default:
			return fmt.Errorf("API error: %s", e.Message)
		}
	case *openai.RequestError:
		return fmt.Errorf("connection failed — check network and base URL")
	default:
		return err
	}
}
