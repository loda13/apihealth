package checker

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
	"github.com/lodatang/apihealth/internal/config"
)

// Result represents the outcome of a health check
type Result struct {
	Name       string
	Model      string
	Success    bool
	StatusCode int
	Duration   time.Duration
	Error      *CheckError
}

// Checker performs concurrent health checks
type Checker struct {
	anthropic *AnthropicChecker
	workers   int
}

// NewChecker creates a new checker with the specified configuration
func NewChecker(timeout time.Duration, workers int, maxRetries int) *Checker {
	return &Checker{
		anthropic: NewAnthropicChecker(timeout, maxRetries),
		workers:   workers,
	}
}

// CheckAll performs health checks on all targets concurrently
func (c *Checker) CheckAll(ctx context.Context, targets []config.Target) []Result {
	results := make([]Result, len(targets))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.workers)

	for i, target := range targets {
		i, target := i, target // Capture loop variables
		g.Go(func() error {
			results[i] = c.anthropic.CheckTarget(ctx, target)
			return nil // Don't fail fast on individual errors
		})
	}

	// Wait for all checks to complete
	_ = g.Wait() // Ignore error since we don't fail fast

	return results
}
