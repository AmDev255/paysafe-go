package paysafe

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimit mirrors the X-RateLimit-* headers.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     int64
}

// Response is the raw outcome of a request; every model carries one in LastResponse.
type Response struct {
	Status             int
	Headers            map[string]string // lower-cased names
	Body               []byte
	RequestID          string
	RateLimit          *RateLimit
	IdempotentReplayed bool
	IdempotencyKey     string
	RetryAfter         int // seconds, 0 when absent
}

func rateLimitFromHeaders(h map[string]string) *RateLimit {
	l, lok := h["x-ratelimit-limit"]
	r, rok := h["x-ratelimit-remaining"]
	s, sok := h["x-ratelimit-reset"]
	if !lok && !rok && !sok {
		return nil
	}
	limit, _ := strconv.Atoi(l)
	remaining, _ := strconv.Atoi(r)
	reset, _ := strconv.ParseInt(s, 10, 64)
	return &RateLimit{Limit: limit, Remaining: remaining, Reset: reset}
}

func retryAfterFromHeaders(h map[string]string, now time.Time) int {
	v, ok := h["retry-after"]
	if !ok {
		return 0
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return n
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			return 0
		}
		return int((d + time.Second - 1) / time.Second)
	}
	return 0
}

func buildResponse(status int, header http.Header, body []byte, idempotencyKey string) *Response {
	h := make(map[string]string, len(header))
	for k, v := range header {
		h[strings.ToLower(k)] = strings.Join(v, ", ")
	}
	return &Response{
		Status:             status,
		Headers:            h,
		Body:               body,
		RequestID:          h["x-request-id"],
		RateLimit:          rateLimitFromHeaders(h),
		IdempotentReplayed: h["idempotent-replayed"] == "true",
		IdempotencyKey:     idempotencyKey,
		RetryAfter:         retryAfterFromHeaders(h, time.Now()),
	}
}

type errorEnvelope struct {
	Error struct {
		Code      string         `json:"code"`
		Message   string         `json:"message"`
		RequestID string         `json:"request_id"`
		Param     string         `json:"param"`
		Details   map[string]any `json:"details"`
		DocURL    string         `json:"doc_url"`
	} `json:"error"`
}

func errorFromResponse(r *Response) *Error {
	e := &Error{Status: r.Status, Code: "unknown", Message: "HTTP " + strconv.Itoa(r.Status) + " from the PaySafe API", RequestID: r.RequestID, Headers: r.Headers, RetryAfter: r.RetryAfter, RateLimit: r.RateLimit}
	var env errorEnvelope
	if len(r.Body) > 0 && json.Unmarshal(r.Body, &env) == nil && (env.Error.Code != "" || env.Error.Message != "") {
		if env.Error.Code != "" {
			e.Code = env.Error.Code
		}
		if env.Error.Message != "" {
			e.Message = env.Error.Message
		}
		if env.Error.RequestID != "" {
			e.RequestID = env.Error.RequestID
		}
		e.Param = env.Error.Param
		e.Details = env.Error.Details
		e.DocURL = env.Error.DocURL
		return e
	}
	if len(r.Body) > 0 {
		raw := r.Body
		if len(raw) > 4096 {
			raw = raw[:4096]
		}
		e.Details = map[string]any{"raw": string(raw)}
	}
	return e
}
