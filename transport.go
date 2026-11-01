package paysafe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	initialBackoff   = 500 * time.Millisecond
	maxBackoff       = 8 * time.Second
	inProgressSleep  = time.Second
	MaxResponseBytes = 16 << 20
)

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, opts *RequestOptions) (*Response, error) {
	if opts == nil {
		opts = &RequestOptions{}
	}
	if opts.IdempotencyKey != "" {
		if err := validateIdempotencyKey(opts.IdempotencyKey); err != nil {
			return nil, err
		}
	}
	if opts.RequestID != "" {
		if err := validateRequestID(opts.RequestID); err != nil {
			return nil, err
		}
	}
	if opts.MaxRetries != nil && *opts.MaxRetries < 0 {
		return nil, invalid("", "max retries must be zero or more")
	}
	if opts.Timeout < 0 {
		return nil, invalid("", "timeout must be positive")
	}
	if err := validateHeaders(opts.Headers); err != nil {
		return nil, err
	}
	for k, vs := range query {
		for _, v := range vs {
			if err := validateQueryValue(k, v); err != nil {
				return nil, err
			}
		}
	}

	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	var encoded []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, invalid("", "request body is not JSON-serialisable: %v", err)
		}
		if len(b) > MaxBodyBytes {
			return nil, invalid("", "request body exceeds the %d byte limit", MaxBodyBytes)
		}
		encoded = b
	}

	retryable := method == http.MethodGet || method == http.MethodDelete || method == http.MethodPatch || opts.IdempotencyKey != ""
	maxAttempts := 1 + c.maxRetries
	if opts.MaxRetries != nil {
		maxAttempts = 1 + *opts.MaxRetries
	}
	timeout := c.timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	for attempt := 1; ; attempt++ {
		started := time.Now()
		resp, err := c.send(ctx, method, target, encoded, opts, timeout)
		if err != nil {
			c.log(LogRecord{Method: method, Path: path, Attempt: attempt, Duration: time.Since(started), Err: err})
			var ce *ConnectionError
			if !errors.As(err, &ce) || !retryable || attempt >= maxAttempts || ctx.Err() != nil {
				return nil, err
			}
			if err := c.sleep(ctx, c.backoff(attempt)); err != nil {
				return nil, &ConnectionError{Err: err}
			}
			continue
		}
		c.log(LogRecord{Method: method, Path: path, Status: resp.Status, Attempt: attempt, Duration: time.Since(started), RequestID: resp.RequestID})
		if resp.Status < 300 {
			return resp, nil
		}
		if resp.Status < 400 {
			return nil, &Error{Status: resp.Status, Code: "unexpected_redirect", Message: "the API answered with a redirect, which this SDK never follows", RequestID: resp.RequestID, Headers: resp.Headers}
		}
		apiErr := errorFromResponse(resp)
		delay, ok := c.retryDelay(apiErr, retryable, attempt)
		if !ok || attempt >= maxAttempts {
			return nil, apiErr
		}
		if err := c.sleep(ctx, delay); err != nil {
			return nil, apiErr
		}
	}
}

func (c *Client) retryDelay(e *Error, retryable bool, attempt int) (time.Duration, bool) {
	switch {
	case e.Code == "idempotency_in_progress":
		return inProgressSleep, true
	case e.Status == 429 || e.Status == 503:
		if e.RetryAfter > 0 {
			return time.Duration(e.RetryAfter) * time.Second, true
		}
		return c.backoff(attempt), true
	case e.Status >= 500 && e.Code != "provider_error" && retryable:
		return c.backoff(attempt), true
	}
	return 0, false
}

func (c *Client) backoff(attempt int) time.Duration {
	ceiling := initialBackoff << (attempt - 1)
	if ceiling > maxBackoff || ceiling <= 0 {
		ceiling = maxBackoff
	}
	return time.Duration(c.rand() * float64(ceiling))
}

func (c *Client) send(ctx context.Context, method, target string, body []byte, opts *RequestOptions, timeout time.Duration) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, &ConnectionError{Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.apiVersion != "" {
		req.Header.Set("PaySafe-Version", c.apiVersion)
	}
	if opts.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", opts.IdempotencyKey)
	}
	if opts.RequestID != "" {
		req.Header.Set("X-Request-ID", opts.RequestID)
	}
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &ConnectionError{Err: err, Timeout: isTimeout(err)}
	}
	defer resp.Body.Close()
	if resp.ContentLength > MaxResponseBytes {
		return nil, &ResponseTooLargeError{}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, &ConnectionError{Err: err, Timeout: isTimeout(err)}
	}
	if len(data) > MaxResponseBytes {
		return nil, &ResponseTooLargeError{}
	}
	return buildResponse(resp.StatusCode, resp.Header, data, opts.IdempotencyKey), nil
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func (c *Client) log(r LogRecord) {
	if c.logger != nil {
		c.logger(r)
	}
}

func decode(resp *Response, v any) error {
	if len(resp.Body) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Body, v); err != nil {
		return &Error{Status: resp.Status, Code: "invalid_json", Message: "the API returned a body this SDK could not parse: " + err.Error(), RequestID: resp.RequestID, Headers: resp.Headers}
	}
	return nil
}
