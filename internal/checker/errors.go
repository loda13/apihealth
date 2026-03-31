package checker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
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
	if err == nil && (statusCode == 401 || statusCode == 403) {
		return &CheckError{
			Type:    ErrorTypeAuth,
			Message: "Authentication failed",
			Err:     fmt.Errorf("status code %d", statusCode),
		}
	}

	if statusCode == 429 {
		return &CheckError{
			Type:    ErrorTypeRateLimit,
			Message: "Rate limit exceeded",
			Err:     fmt.Errorf("status code %d", statusCode),
		}
	}

	if statusCode >= 500 {
		return &CheckError{
			Type:    ErrorTypeServer,
			Message: "Server error",
			Err:     fmt.Errorf("status code %d", statusCode),
		}
	}

	if err == nil {
		return nil
	}

	// Timeout errors
	if os.IsTimeout(err) || errors.Is(err, context.DeadlineExceeded) {
		return &CheckError{
			Type:    ErrorTypeTimeout,
			Message: "Request timeout",
			Err:     err,
		}
	}

	// DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
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
	if errors.Is(err, syscall.ECONNREFUSED) {
		return &CheckError{
			Type:    ErrorTypeConnection,
			Message: "Connection refused",
			Err:     err,
		}
	}

	return &CheckError{
		Type:    ErrorTypeUnknown,
		Message: "Unknown error",
		Err:     err,
	}
}
