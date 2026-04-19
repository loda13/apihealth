package checker

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"testing"
)

func TestClassifyError_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		wantType   ErrorType
		wantMsg    string
	}{
		{
			name:       "401 auth error",
			err:        fmt.Errorf("invalid_api_key"),
			statusCode: 401,
			wantType:   ErrorTypeAuth,
			wantMsg:    "Authentication failed",
		},
		{
			name:       "429 rate limit",
			err:        fmt.Errorf("giving up after 3 attempt(s)"),
			statusCode: 429,
			wantType:   ErrorTypeRateLimit,
			wantMsg:    "Rate limit exceeded",
		},
		{
			name:       "500 server error",
			err:        fmt.Errorf("internal server error"),
			statusCode: 500,
			wantType:   ErrorTypeServer,
			wantMsg:    "Server error (HTTP 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err, tt.statusCode)
			if result.Type != tt.wantType {
				t.Errorf("ClassifyError() type = %v, want %v", result.Type, tt.wantType)
			}
			if result.Message != tt.wantMsg {
				t.Errorf("ClassifyError() message = %v, want %v", result.Message, tt.wantMsg)
			}
		})
	}
}

func TestClassifyError_RetryExhaustion(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		wantType   ErrorType
		wantMsg    string
	}{
		{
			name:       "timeout with retry exhaustion",
			err:        fmt.Errorf("POST https://api.example.com/v1/messages giving up after 3 attempt(s): context deadline exceeded"),
			statusCode: 0,
			wantType:   ErrorTypeTimeout,
			wantMsg:    "Request timeout",
		},
		{
			name:       "DNS error with retry exhaustion",
			err:        fmt.Errorf("POST https://api.example.com/v1/messages giving up after 3 attempt(s): no such host"),
			statusCode: 0,
			wantType:   ErrorTypeDNS,
			wantMsg:    "DNS resolution failed",
		},
		{
			name:       "connection refused with retry exhaustion",
			err:        fmt.Errorf("POST https://api.example.com/v1/messages giving up after 3 attempt(s): connection refused"),
			statusCode: 0,
			wantType:   ErrorTypeConnection,
			wantMsg:    "Connection failed after retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err, tt.statusCode)
			if result.Type != tt.wantType {
				t.Errorf("ClassifyError() type = %v, want %v", result.Type, tt.wantType)
			}
			if result.Message != tt.wantMsg {
				t.Errorf("ClassifyError() message = %v, want %v", result.Message, tt.wantMsg)
			}
		})
	}
}

func TestClassifyError_DirectErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		wantType   ErrorType
	}{
		{
			name:       "context deadline exceeded",
			err:        context.DeadlineExceeded,
			statusCode: 0,
			wantType:   ErrorTypeTimeout,
		},
		{
			name:       "DNS error",
			err:        &net.DNSError{Err: "no such host", Name: "example.com"},
			statusCode: 0,
			wantType:   ErrorTypeDNS,
		},
		{
			name:       "connection refused",
			err:        syscall.ECONNREFUSED,
			statusCode: 0,
			wantType:   ErrorTypeConnection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err, tt.statusCode)
			if result.Type != tt.wantType {
				t.Errorf("ClassifyError() type = %v, want %v", result.Type, tt.wantType)
			}
		})
	}
}

func TestClassifyError_429WithRetryExhaustion(t *testing.T) {
	// This is the key test case: 429 with retry exhaustion should be classified as RateLimit
	err := fmt.Errorf("POST https://api.anthropic.com/v1/messages giving up after 3 attempt(s)")
	statusCode := 429

	result := ClassifyError(err, statusCode)

	if result.Type != ErrorTypeRateLimit {
		t.Errorf("ClassifyError() type = %v, want %v", result.Type, ErrorTypeRateLimit)
	}
	if result.Message != "Rate limit exceeded" {
		t.Errorf("ClassifyError() message = %v, want 'Rate limit exceeded'", result.Message)
	}
}
