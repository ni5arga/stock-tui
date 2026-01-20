package data

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

var defaultClient = &http.Client{
	Timeout: 10 * time.Second,
}

type httpError struct {
	StatusCode int
	Status     string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("http error: %s", e.Status)
}

func (e *httpError) IsRateLimit() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

func (e *httpError) IsRetryable() bool {
	return e.StatusCode >= 500
}

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %s", e.RetryAfter)
}

type fetchOptions struct {
	MaxRetries int
	BaseDelay  time.Duration
}

func defaultFetchOptions() fetchOptions {
	return fetchOptions{
		MaxRetries: 3,
		BaseDelay:  500 * time.Millisecond,
	}
}

func fetch(ctx context.Context, url string, opts *fetchOptions) ([]byte, error) {
	if opts == nil {
		o := defaultFetchOptions()
		opts = &o
	}

	var lastErr error
	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := opts.BaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "stock-tui/1.0")
		req.Header.Set("Accept", "application/json")

		resp, err := defaultClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Read body first to close properly
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			// Do not retry 429 inside the library, let the app handle it
			retryStr := resp.Header.Get("Retry-After")
			retryAfter := 60 * time.Second // default
			if retryStr != "" {
				if d, err := time.ParseDuration(retryStr + "s"); err == nil {
					retryAfter = d
				}
			}
			return nil, &RateLimitError{RetryAfter: retryAfter}
		}

		if resp.StatusCode != http.StatusOK {
			herr := &httpError{StatusCode: resp.StatusCode, Status: resp.Status}
			if herr.IsRetryable() {
				lastErr = herr
				continue
			}
			return nil, herr
		}

		return body, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("after %d retries: %w", opts.MaxRetries, lastErr)
	}
	return nil, fmt.Errorf("fetch failed")
}
