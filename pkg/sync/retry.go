package sync

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// RetryConfig contains retry configuration
type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
	}
}

// RetryFunc is a function that can be retried
type RetryFunc func() error

// RetryWithBackoff executes a function with exponential backoff retry
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn RetryFunc) error {
	var lastErr error
	backoff := config.InitialInterval

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Check if we should retry
		if attempt >= config.MaxAttempts {
			break
		}

		// Log retry attempt (could be replaced with proper logging)
		fmt.Printf("[RETRY] Attempt %d/%d failed: %v, retrying in %v\n",
			attempt, config.MaxAttempts, err, backoff)

		// Wait with backoff
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}

		// Exponential backoff
		backoff *= 2
		if backoff > config.MaxInterval {
			backoff = config.MaxInterval
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network errors are retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Connection errors are retryable
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	// DNS errors are retryable
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.Temporary()
	}

	// Check error message for common retryable patterns
	errMsg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"temporary",
		"connection reset",
		"connection refused",
		"too many requests",
		"rate limit",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Check HTTP status codes
	if strings.Contains(errMsg, "status code") {
		// Extract status code and check if retryable
		if strings.Contains(errMsg, "429") || // Too Many Requests
			strings.Contains(errMsg, "500") || // Internal Server Error
			strings.Contains(errMsg, "502") || // Bad Gateway
			strings.Contains(errMsg, "503") || // Service Unavailable
			strings.Contains(errMsg, "504") { // Gateway Timeout
			return true
		}

		// 4xx errors (except 429) are not retryable
		if strings.Contains(errMsg, "400") ||
			strings.Contains(errMsg, "401") ||
			strings.Contains(errMsg, "403") ||
			strings.Contains(errMsg, "404") {
			return false
		}
	}

	// Default: don't retry
	return false
}

// IsHTTPError checks if an error is an HTTP error with the given status code
func IsHTTPError(err error, statusCode int) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	statusStr := fmt.Sprintf("%d", statusCode)
	return strings.Contains(errMsg, statusStr)
}

// RetryableHTTPClient wraps an HTTP client with retry logic
type RetryableHTTPClient struct {
	Client      *http.Client
	RetryConfig RetryConfig
}

// NewRetryableHTTPClient creates a new retryable HTTP client
func NewRetryableHTTPClient(client *http.Client, config RetryConfig) *RetryableHTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &RetryableHTTPClient{
		Client:      client,
		RetryConfig: config,
	}
}

// Do executes an HTTP request with retry logic
func (r *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	retryErr := RetryWithBackoff(req.Context(), r.RetryConfig, func() error {
		resp, err = r.Client.Do(req)
		if err != nil {
			return err
		}

		// Check if status code is retryable
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			resp.Body.Close()
			return fmt.Errorf("retryable status code: %d", resp.StatusCode)
		}

		return nil
	})

	if retryErr != nil {
		return nil, retryErr
	}

	return resp, nil
}

// WithRetry wraps a function with retry logic
func WithRetry(ctx context.Context, config RetryConfig, fn RetryFunc) error {
	return RetryWithBackoff(ctx, config, fn)
}

// RetryableError wraps an error to indicate it should be retried
type RetryableError struct {
	Err     error
	Attempt int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d): %v", e.Attempt, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, attempt int) *RetryableError {
	return &RetryableError{
		Err:     err,
		Attempt: attempt,
	}
}
