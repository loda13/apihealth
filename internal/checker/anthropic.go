package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/lodatang/apihealth/internal/config"
)

const anthropicVersion = "2023-06-01"

type AnthropicChecker struct {
	client     *retryablehttp.Client
	timeout    time.Duration
	maxRetries int
	maxTokens  int
	message    string
}

type anthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
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
func NewAnthropicChecker(timeout time.Duration, maxRetries, maxTokens int, message string) *AnthropicChecker {
	client := retryablehttp.NewClient()
	client.RetryMax = maxRetries
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 5 * time.Second
	client.HTTPClient.Timeout = timeout
	client.Logger = nil // suppress default retryablehttp logs

	// Custom retry policy: retry on 429 and 5xx, but not on auth errors
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if err != nil {
			return true, err
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			return true, nil
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return false, nil
		}
		return false, nil
	}

	return &AnthropicChecker{
		client:     client,
		timeout:    timeout,
		maxRetries: maxRetries,
		maxTokens:  maxTokens,
		message:    message,
	}
}

// CheckTarget performs a health check on an Anthropic API target
func (a *AnthropicChecker) CheckTarget(ctx context.Context, target config.Target) Result {
	start := time.Now()
	retryCount := 0

	// Show retry progress on stderr
	a.client.RequestLogHook = func(_ retryablehttp.Logger, _ *http.Request, attempt int) {
		if attempt > 0 {
			retryCount = attempt
			fmt.Fprintf(os.Stderr, "\r[%s] 正在重试 %d/%d...", target.Name, attempt, a.maxRetries)
		}
	}

	endpoint := fmt.Sprintf("%s/v1/messages", target.BaseURL)

	reqBody := anthropicRequest{
		Model:     target.Model,
		MaxTokens: a.maxTokens,
		Messages: []message{
			{Role: "user", Content: a.message},
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

	req.Header.Set("x-api-key", target.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := a.client.Do(req)
	duration := time.Since(start)

	// Clear retry progress line
	if retryCount > 0 {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 60))
	}

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.Error = ClassifyError(fmt.Errorf("reading response: %w", err), resp.StatusCode)
		return result
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var apiResp anthropicResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			result.Success = false
			result.Error = ClassifyError(fmt.Errorf("parsing response: %w", err), resp.StatusCode)
			return result
		}
		if apiResp.ID == "" || apiResp.Model == "" {
			result.Success = false
			result.Error = ClassifyError(fmt.Errorf("invalid response structure"), resp.StatusCode)
			return result
		}
		result.Success = true
		return result
	}

	// Parse API error response and save type/message directly
	var apiErr anthropicError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		result.Success = false
		result.Error = ClassifyError(fmt.Errorf("API error: %s", strings.TrimSpace(string(body))), resp.StatusCode)
		return result
	}

	result.Success = false
	result.APIErrorType = apiErr.Error.Type
	result.APIErrorMsg = apiErr.Error.Message
	result.Error = ClassifyError(fmt.Errorf("%s: %s", apiErr.Error.Type, apiErr.Error.Message), resp.StatusCode)
	return result
}
