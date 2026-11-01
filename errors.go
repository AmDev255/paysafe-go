package paysafe

import (
	"errors"
	"fmt"
)

// Error is any non-2xx answer from the API. Inspect Code (stable, lower_snake_case) or use the Is* helpers.
type Error struct {
	Status     int
	Code       string
	Message    string
	RequestID  string
	Param      string
	Details    map[string]any
	DocURL     string
	Headers    map[string]string
	RetryAfter int // seconds, 0 when absent
	RateLimit  *RateLimit
}

func (e *Error) Error() string {
	id := e.RequestID
	if id == "" {
		id = "no request id"
	}
	return fmt.Sprintf("%s (%d) %s [%s]", e.Code, e.Status, e.Message, id)
}

// Retryable reports whether the documented retry policy allows retrying with the same idempotency key.
func (e *Error) Retryable() bool {
	if e.Status == 429 || e.Status == 503 || e.Code == "idempotency_in_progress" {
		return true
	}
	return e.Status >= 500 && e.Code != "provider_error"
}

// IPNotAllowed is true for permission errors caused by the project's IP allowlist.
func (e *Error) IPNotAllowed() bool {
	c, _ := e.Details["code"].(string)
	return c == "ip_not_allowed" || c == "ip_allowlist_empty"
}

// ObservedIP returns the address the API saw on an IP-allowlist rejection.
func (e *Error) ObservedIP() string {
	ip, _ := e.Details["ip"].(string)
	return ip
}

func (e *Error) detailInt(key string) (int64, bool) {
	switch v := e.Details[key].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	}
	return 0, false
}

// Available and Requested carry the figures of an insufficient_funds error.
func (e *Error) Available() (int64, bool) { return e.detailInt("available") }
func (e *Error) Requested() (int64, bool) { return e.detailInt("requested") }

// Scope names the rate limit that was hit on a rate_limited error.
func (e *Error) Scope() string {
	s, _ := e.Details["scope"].(string)
	return s
}

func codeIs(err error, codes ...string) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	for _, c := range codes {
		if e.Code == c {
			return true
		}
	}
	return false
}

func statusIs(err error, status int) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == status
}

// Error classification helpers mirror the documented error codes.
func IsValidation(err error) bool { return codeIs(err, "validation_error") || statusIs(err, 400) }
func IsAuthentication(err error) bool {
	return codeIs(err, "authentication_error") || statusIs(err, 401)
}
func IsPermission(err error) bool { return codeIs(err, "permission_error") }
func IsFeatureDisabled(err error) bool {
	return codeIs(err, "feature_disabled")
}
func IsNotFound(err error) bool          { return codeIs(err, "not_found") || statusIs(err, 404) }
func IsInsufficientFunds(err error) bool { return codeIs(err, "insufficient_funds") }
func IsConflict(err error) bool {
	return codeIs(err, "conflict", "idempotency_conflict", "idempotency_in_progress", "invalid_state") || statusIs(err, 409)
}
func IsIdempotencyConflict(err error) bool   { return codeIs(err, "idempotency_conflict") }
func IsIdempotencyInProgress(err error) bool { return codeIs(err, "idempotency_in_progress") }
func IsInvalidState(err error) bool          { return codeIs(err, "invalid_state") }
func IsRateLimited(err error) bool           { return codeIs(err, "rate_limited") || statusIs(err, 429) }
func IsProviderError(err error) bool         { return codeIs(err, "provider_error") }
func IsProviderUnavailable(err error) bool   { return codeIs(err, "provider_unavailable") }
func IsServiceUnavailable(err error) bool    { return codeIs(err, "service_unavailable") }
func IsInternal(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status >= 500 && !IsProviderError(err) && !IsProviderUnavailable(err) && !IsServiceUnavailable(err)
}

// ConnectionError means no HTTP response was received.
type ConnectionError struct {
	Err     error
	Timeout bool
}

func (e *ConnectionError) Error() string {
	if e.Timeout {
		return "paysafe: request timed out: " + e.Err.Error()
	}
	return "paysafe: could not reach the API: " + e.Err.Error()
}
func (e *ConnectionError) Unwrap() error { return e.Err }

// ResponseTooLargeError is returned when a body exceeds MaxResponseBytes. It is never retried.
type ResponseTooLargeError struct{}

func (*ResponseTooLargeError) Error() string { return "paysafe: response body exceeds the 16 MB limit" }

// ConfigError is a bad client option.
type ConfigError struct{ Reason string }

func (e *ConfigError) Error() string { return "paysafe: " + e.Reason }

// InvalidRequestError is raised locally before any request is sent.
type InvalidRequestError struct {
	Reason string
	Param  string
}

func (e *InvalidRequestError) Error() string { return "paysafe: " + e.Reason }

// PollingTimeoutError is returned by the Wait* helpers; LastSeen is the last object fetched.
type PollingTimeoutError struct {
	Reason   string
	LastSeen any
}

func (e *PollingTimeoutError) Error() string { return "paysafe: " + e.Reason }

// SignatureError explains why a webhook delivery was rejected. Never includes the secret.
type SignatureError struct{ Reason string }

func (e *SignatureError) Error() string { return "paysafe: webhook signature " + e.Reason }

const (
	SignatureMissingHeader             = "missing_header"
	SignatureMalformedHeader           = "malformed_header"
	SignatureTimestampOutsideTolerance = "timestamp_outside_tolerance"
	SignatureMismatch                  = "signature_mismatch"
	SignatureInvalidPayload            = "invalid_payload"
)
