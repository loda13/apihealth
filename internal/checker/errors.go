package checker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
)

type ErrorType int

const (
	ErrorTypeAuth ErrorType = iota
	ErrorTypeRateLimit
	ErrorTypeTimeout
	ErrorTypeDNS
	ErrorTypeConnection
	ErrorTypeServer
	ErrorTypeUnknown
)

func (e ErrorType) String() string {
	switch e {
	case ErrorTypeAuth:
		return "Authentication"
	case ErrorTypeRateLimit:
		return "Rate Limit"
	case ErrorTypeTimeout:
		return "Timeout"
	case ErrorTypeDNS:
		return "DNS"
	case ErrorTypeConnection:
		return "Connection"
	case ErrorTypeServer:
		return "Server Error"
	default:
		return "Unknown"
	}
}

type CheckError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *CheckError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *CheckError) Unwrap() error {
	return e.Err
}

// ClassifyError categorizes errors for better diagnostics
func ClassifyError(err error, statusCode int) *CheckError {
	// Handle HTTP status code errors first
	if statusCode == 401 || statusCode == 403 {
		underlying := err
		if underlying == nil {
			underlying = fmt.Errorf("HTTP %d", statusCode)
		}
		return &CheckError{
			Type:    ErrorTypeAuth,
			Message: "Authentication failed",
			Err:     underlying,
		}
	}

	if statusCode == 429 {
		underlying := err
		if underlying == nil {
			underlying = fmt.Errorf("HTTP %d", statusCode)
		}
		return &CheckError{
			Type:    ErrorTypeRateLimit,
			Message: "Rate limit exceeded",
			Err:     underlying,
		}
	}

	if statusCode >= 500 {
		underlying := err
		if underlying == nil {
			underlying = fmt.Errorf("HTTP %d", statusCode)
		}
		return &CheckError{
			Type:    ErrorTypeServer,
			Message: fmt.Sprintf("Server error (HTTP %d)", statusCode),
			Err:     underlying,
		}
	}

	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for retry exhaustion and extract underlying error message
	var underlyingMsg string
	isRetryExhaustion := false
	if strings.Contains(errStr, "giving up after") {
		// Parse: "POST https://... giving up after N attempt(s): <underlying>"
		if idx := strings.LastIndex(errStr, ": "); idx != -1 {
			underlyingMsg = errStr[idx+2:]
			if underlyingMsg != "" && underlyingMsg != "<nil>" {
				isRetryExhaustion = true
			}
		}
	}

	// Timeout errors (check both original error and underlying message)
	if os.IsTimeout(err) || errors.Is(err, context.DeadlineExceeded) ||
		(isRetryExhaustion && (strings.Contains(underlyingMsg, "timeout") ||
			strings.Contains(underlyingMsg, "deadline exceeded"))) {
		return &CheckError{
			Type:    ErrorTypeTimeout,
			Message: "Request timeout",
			Err:     err,
		}
	}

	// DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) ||
		(isRetryExhaustion && (strings.Contains(underlyingMsg, "no such host") ||
			strings.Contains(underlyingMsg, "dns") ||
			strings.Contains(underlyingMsg, "name resolution"))) {
		return &CheckError{
			Type:    ErrorTypeDNS,
			Message: "DNS resolution failed",
			Err:     err,
		}
	}

	// Connection errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return &CheckError{
				Type:    ErrorTypeConnection,
				Message: "Connection failed",
				Err:     err,
			}
		}
	}

	// Connection refused
	if errors.Is(err, syscall.ECONNREFUSED) ||
		(isRetryExhaustion && strings.Contains(underlyingMsg, "connection refused")) {
		message := "Connection refused"
		if isRetryExhaustion {
			message = "Connection failed after retries"
		}
		return &CheckError{
			Type:    ErrorTypeConnection,
			Message: message,
			Err:     err,
		}
	}

	// Retry exhaustion with unclassified underlying error
	if strings.Contains(err.Error(), "giving up after") {
		return &CheckError{
			Type:    ErrorTypeConnection,
			Message: "Connection failed after retries",
			Err:     err,
		}
	}

	return &CheckError{
		Type:    ErrorTypeUnknown,
		Message: "Request failed",
		Err:     err,
	}
}
