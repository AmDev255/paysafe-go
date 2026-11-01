package paysafe

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const (
	minPollInterval  = 500 * time.Millisecond
	slowPollAfter    = 30 * time.Second
	slowPollInterval = 5 * time.Second
)

func apiPath(segments ...string) string {
	p := "/v1"
	for _, s := range segments {
		p += "/" + url.PathEscape(s)
	}
	return p
}

func uuid4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("paysafe: crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func withKey(opts *RequestOptions) *RequestOptions {
	if opts == nil {
		return &RequestOptions{IdempotencyKey: uuid4()}
	}
	if opts.IdempotencyKey != "" {
		return opts
	}
	copied := *opts
	copied.IdempotencyKey = uuid4()
	return &copied
}

func setIf(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func listQuery(limit int, cursor string) (url.Values, error) {
	if err := validateLimit(limit); err != nil {
		return nil, err
	}
	if err := validateCursor(cursor); err != nil {
		return nil, err
	}
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	setIf(q, "cursor", cursor)
	return q, nil
}

// WaitOptions tune the Wait* helpers.
type WaitOptions struct {
	Interval time.Duration       // minimum 500 ms
	Timeout  time.Duration       // give up after this much waiting
	OnUpdate func(status string) // called on every status change
}

func poll[T any](ctx context.Context, c *Client, fetch func(context.Context) (T, error), status func(T) string, terminal func(T) bool, defaults WaitOptions, opts *WaitOptions, what string) (T, error) {
	interval, timeout := defaults.Interval, defaults.Timeout
	var onUpdate func(string)
	if opts != nil {
		if opts.Interval > 0 {
			interval = opts.Interval
		}
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
		onUpdate = opts.OnUpdate
	}
	if interval < minPollInterval {
		interval = minPollInterval
	}
	var waited time.Duration
	last := "\x00"
	var zero T
	for {
		obj, err := fetch(ctx)
		if err != nil {
			return zero, err
		}
		if s := status(obj); onUpdate != nil && s != last {
			onUpdate(s)
			last = s
		}
		if terminal(obj) {
			return obj, nil
		}
		if waited >= timeout {
			return zero, &PollingTimeoutError{Reason: fmt.Sprintf("%s did not reach a terminal state within %s", what, timeout), LastSeen: obj}
		}
		delay := interval
		if waited >= slowPollAfter && delay < slowPollInterval {
			delay = slowPollInterval
		}
		if remaining := timeout - waited; delay > remaining {
			delay = remaining
		}
		if err := c.sleep(ctx, delay); err != nil {
			return zero, &PollingTimeoutError{Reason: fmt.Sprintf("polling for %s was cancelled: %v", what, err), LastSeen: obj}
		}
		waited += delay
	}
}
