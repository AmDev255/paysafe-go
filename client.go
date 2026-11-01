// Package paysafe is the official Go SDK for the PaySafe payments API (https://paysafe.co.tz/docs).
package paysafe

import (
	"context"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	Version           = "0.1.0"
	DefaultBaseURL    = "https://api.paysafe.co.tz"
	DefaultTimeout    = 30 * time.Second
	DefaultMaxRetries = 4
)

// Environment is derived from the key prefix.
type Environment string

const (
	EnvironmentTest Environment = "test"
	EnvironmentLive Environment = "live"
)

// LogRecord describes one HTTP attempt. It never contains the key or any body.
type LogRecord struct {
	Method    string
	Path      string
	Status    int // 0 when no response was received
	Attempt   int
	Duration  time.Duration
	RequestID string
	Err       error
}

// Client talks to one PaySafe project in one environment.
type Client struct {
	Environment Environment

	Payments         *Payments
	PaymentLinks     *PaymentLinks
	ControlNumbers   *ControlNumbers
	Refunds          *Refunds
	Balance          *Balance
	Customers        *Customers
	Transfers        *Transfers
	WebhookEndpoints *WebhookEndpoints
	Events           *Events

	apiKey     string
	baseURL    string
	timeout    time.Duration
	maxRetries int
	apiVersion string
	userAgent  string
	http       *http.Client
	logger     func(LogRecord)
	sleep      func(context.Context, time.Duration) error
	rand       func() float64
	allowHTTP  bool
	uaSuffix   string
}

// Option configures a Client.
type Option func(*Client)

func WithBaseURL(u string) Option          { return func(c *Client) { c.baseURL = u } }
func WithTimeout(d time.Duration) Option   { return func(c *Client) { c.timeout = d } }
func WithMaxRetries(n int) Option          { return func(c *Client) { c.maxRetries = n } }
func WithAPIVersion(v string) Option       { return func(c *Client) { c.apiVersion = v } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }
func WithLogger(f func(LogRecord)) Option  { return func(c *Client) { c.logger = f } }
func WithUserAgentSuffix(s string) Option  { return func(c *Client) { c.uaSuffix = s } }
func WithAllowInsecureHTTP() Option        { return func(c *Client) { c.allowHTTP = true } }
func withSleep(f func(context.Context, time.Duration) error) Option {
	return func(c *Client) { c.sleep = f }
}
func withRand(f func() float64) Option { return func(c *Client) { c.rand = f } }

func environmentFromKey(key string) (Environment, error) {
	switch {
	case strings.HasPrefix(key, "paysafe_test_"):
		return EnvironmentTest, nil
	case strings.HasPrefix(key, "paysafe_live_"):
		return EnvironmentLive, nil
	}
	return "", &ConfigError{Reason: "API key must start with paysafe_test_ or paysafe_live_"}
}

// NewClient builds a client. An empty apiKey falls back to PAYSAFE_SECRET_KEY; the base URL falls back
// to PAYSAFE_BASE_URL and then to DefaultBaseURL.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	c := &Client{
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
		sleep:      defaultSleep,
		rand:       rand.Float64,
	}
	for _, o := range opts {
		o(c)
	}
	if apiKey == "" {
		apiKey = os.Getenv("PAYSAFE_SECRET_KEY")
	}
	if apiKey == "" {
		return nil, &ConfigError{Reason: "no API key given and PAYSAFE_SECRET_KEY is not set"}
	}
	if !reAPIKey.MatchString(apiKey) {
		return nil, &ConfigError{Reason: "API key contains invalid characters"}
	}
	env, err := environmentFromKey(apiKey)
	if err != nil {
		return nil, err
	}
	c.apiKey = apiKey
	c.Environment = env

	if c.baseURL == "" {
		c.baseURL = os.Getenv("PAYSAFE_BASE_URL")
	}
	if c.baseURL == "" {
		c.baseURL = DefaultBaseURL
	}
	if strings.ContainsAny(c.baseURL, " \t\r\n") || hasControl(c.baseURL, false) {
		return nil, &ConfigError{Reason: "base URL contains whitespace or control characters"}
	}
	c.baseURL = strings.TrimRight(c.baseURL, "/")
	c.baseURL = strings.TrimSuffix(c.baseURL, "/v1")
	u, err := url.Parse(c.baseURL)
	if err != nil || u.Scheme == "" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, &ConfigError{Reason: "base URL must be an absolute URL without credentials, query or fragment"}
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (c.allowHTTP || isLoopback(u.Hostname()))) {
		return nil, &ConfigError{Reason: "base URL must use https (plain http is only allowed for localhost or with WithAllowInsecureHTTP)"}
	}
	if c.timeout <= 0 {
		return nil, &ConfigError{Reason: "timeout must be positive"}
	}
	if c.maxRetries < 0 {
		return nil, &ConfigError{Reason: "max retries must be zero or more"}
	}
	if hasControl(c.uaSuffix, false) {
		return nil, &ConfigError{Reason: "user agent suffix contains control characters"}
	}
	c.userAgent = "paysafe-go/" + Version + " (Go " + runtime.Version() + ")"
	if c.uaSuffix != "" {
		c.userAgent += " " + c.uaSuffix
	}
	if c.http == nil {
		c.http = &http.Client{}
	}
	if c.http.CheckRedirect == nil {
		c.http.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}

	c.Payments = &Payments{c: c}
	c.PaymentLinks = &PaymentLinks{c: c}
	c.ControlNumbers = &ControlNumbers{c: c}
	c.Refunds = &Refunds{c: c}
	c.Balance = &Balance{c: c}
	c.Customers = &Customers{c: c}
	c.Transfers = &Transfers{c: c}
	c.WebhookEndpoints = &WebhookEndpoints{c: c}
	c.Events = &Events{c: c}
	return c, nil
}

func defaultSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// RequestOptions tune one call.
type RequestOptions struct {
	// Required for payments, refunds and transfers; generated when empty.
	IdempotencyKey string
	// Your own correlation id, 8 to 64 characters of A-Z a-z 0-9 . _ -
	RequestID string
	// Per-attempt timeout for this call.
	Timeout time.Duration
	// Retries after the first attempt for this call; nil keeps the client default.
	MaxRetries *int
	// Extra headers. Authorization, Host and framing headers cannot be set.
	Headers map[string]string
}

// Request is a low-level escape hatch: any method and path under /v1 with the decoded response.
func (c *Client) Request(ctx context.Context, method, path string, query url.Values, body any, opts *RequestOptions) (*Response, error) {
	return c.do(ctx, strings.ToUpper(method), "/"+strings.TrimLeft(path, "/"), query, body, opts)
}
