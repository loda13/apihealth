package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/lodatang/apihealth/internal/config"
)

const anthropicVersion = "2023-06-01"

type AnthropicChecker struct {
	client      *retryablehttp.Client
	timeout     time.Duration
	maxRetries  int
}

type anthropicRequest struct {
	Model      string    `json:"model"`
	MaxTokens  int       `json:"max_tokens"`
	Messages   []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
}

type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewAnthropicChecker creates a new Anthropic API checker
func NewAnthropicChecker(timeout time.Duration, maxRetries int) *AnthropicChecker {
	client := retryablehttp.NewClient()
	client.RetryMax = maxRetries
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 5 * time.Second
	client.HTTPClient.Timeout = timeout

	// Custom retry policy: retry on 429 and 5xx, but not on auth errors
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if err != nil {
			return true, err
		}
		// Retry on rate limit and server errors
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			return true, nil
		}
		// Don't retry on auth errors
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return false, nil
		}
		return false, nil
	}

	return &AnthropicChecker{
		client:     client,
		timeout:    timeout,
		maxRetries: maxRetries,
	}
}

// CheckTarget performs a health check on an Anthropic API target
func (a *AnthropicChecker) CheckTarget(ctx context.Context, target config.Target) Result {
	start := time.Now()

	// Construct endpoint URL
	endpoint := fmt.Sprintf("%s/v1/messages", target.BaseURL)

	// Create minimal request payload
	reqBody := anthropicRequest{
		Model:     target.Model,
		MaxTokens: 1,
		Messages: []message{
			{Role: "user", Content: "Hi"},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Result{
			Name:     target.Name,
			Model:    target.Model,
			Success:  false,
			Duration: time.Since(start),
			Error:    ClassifyError(fmt.Errorf("marshaling request: %w", err), 0),
		}
	}

	// Create HTTP request
	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return Result{
			Name:     target.Name,
			Model:    target.Model,
			Success:  false,
			Duration: time.Since(start),
			Error:    ClassifyError(fmt.Errorf("creating request: %w", err), 0),
		}
	}

	// Set required headers
	req.Header.Set("x-api-key", target.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	// Execute request
	resp, err := a.client.Do(req)
	duration := time.Since(start)

	result := Result{
		Name:     target.Name,
		Model:    target.Model,
		Duration: duration,
	}

	if err != nil {
		result.Success = false
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
			result.StatusCode = statusCode
		}
		result.Error = ClassifyError(err, statusCode)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.Error = ClassifyError(fmt.Errorf("reading response: %w", err), resp.StatusCode)
		return result
	}

	// Check for success (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Parse response to verify it's valid
		var apiResp anthropicResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			result.Success = false
			result.Error = ClassifyError(fmt.Errorf("parsing response: %w", err), resp.StatusCode)
			return result
		}

		// Verify response has expected fields
		if apiResp.ID == "" || apiResp.Model == "" {
			result.Success = false
			result.Error = ClassifyError(fmt.Errorf("invalid response structure"), resp.StatusCode)
			return result
		}

		result.Success = true
		return result
	}

	// Handle error responses
	var apiErr anthropicError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, use the raw body
		result.Success = false
		result.Error = ClassifyError(fmt.Errorf("API error: %s", strings.TrimSpace(string(body))), resp.StatusCode)
		return result
	}

	// Build error message from API response
	var errMsg string
	if apiErr.Error.Type != "" && apiErr.Error.Message != "" {
		errMsg = fmt.Sprintf("%s: %s", apiErr.Error.Type, apiErr.Error.Message)
	} else if apiErr.Error.Message != "" {
		errMsg = apiErr.Error.Message
	} else if apiErr.Error.Type != "" {
		errMsg = apiErr.Error.Type
	} else {
		errMsg = fmt.Sprintf("HTTP %d error", resp.StatusCode)
	}

	result.Success = false
	result.Error = ClassifyError(fmt.Errorf("%s", errMsg), resp.StatusCode)
	return result
}
