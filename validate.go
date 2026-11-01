package paysafe

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	MaxBodyBytes     = 65536
	MaxMetadataBytes = 8192
)

var (
	reID          = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	reHeaderName  = regexp.MustCompile(`^[A-Za-z0-9-]{1,100}$`)
	reRequestID   = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)
	rePhone       = regexp.MustCompile(`^\+255[0-9]{9}$`)
	reQueryKey    = regexp.MustCompile(`^[A-Za-z0-9_]{1,40}$`)
	reDateTime    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2}))?$`)
	reAPIKey      = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	protectedHdrs = map[string]bool{"authorization": true, "host": true, "content-length": true, "transfer-encoding": true, "connection": true}
	loopbackHosts = map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true, "[::1]": true, "host.docker.internal": true}
)

func invalid(param, format string, args ...any) error {
	return &InvalidRequestError{Reason: fmt.Sprintf(format, args...), Param: param}
}

func hasControl(s string, allowTab bool) bool {
	for _, r := range s {
		if r == 0x7F || (r < 0x20 && !(allowTab && r == '\t')) {
			return true
		}
	}
	return false
}

func validateIdempotencyKey(key string) error {
	if len(key) < 1 || len(key) > 255 {
		return invalid("", "Idempotency-Key must be 1 to 255 printable ASCII characters")
	}
	for _, r := range key {
		if r < 0x20 || r > 0x7E {
			return invalid("", "Idempotency-Key must be 1 to 255 printable ASCII characters")
		}
	}
	return nil
}

func validateRequestID(id string) error {
	if !reRequestID.MatchString(id) {
		return invalid("", "X-Request-ID must be 8 to 64 characters of A-Z a-z 0-9 . _ -")
	}
	return nil
}

// validateID guards values that go into URL paths.
func validateID(id, name string) error {
	if !reID.MatchString(id) {
		return invalid(name, "%s must be 1 to 64 characters of A-Z a-z 0-9 _ -", name)
	}
	return nil
}

func validateHeaders(h map[string]string) error {
	for name, value := range h {
		if !reHeaderName.MatchString(name) {
			return invalid("", "header names may only contain letters, digits and dashes")
		}
		if protectedHdrs[strings.ToLower(name)] {
			return invalid("", "the %s header cannot be overridden", name)
		}
		if len(value) > 4096 || hasControl(value, true) {
			return invalid("", "header %s has an invalid value", name)
		}
	}
	return nil
}

func validatePhone(phone, field string) error {
	if !rePhone.MatchString(phone) {
		return invalid(field, "%s must be an E.164 Tanzanian number (+255 followed by nine digits)", field)
	}
	return nil
}

func validateAmount(v, min, max int64, field string) error {
	if v < min || v > max {
		return invalid(field, "%s must be an integer between %d and %d", field, min, max)
	}
	return nil
}

func validateEnum(v, field string, allowed []string) error {
	if v == "" {
		return nil
	}
	for _, a := range allowed {
		if a == v {
			return nil
		}
	}
	return invalid(field, "%s must be one of %s", field, strings.Join(allowed, ", "))
}

func validateString(v *string, field string, min, max int) error {
	if v == nil {
		return nil
	}
	n := len([]rune(*v))
	if n < min || n > max || hasControl(*v, true) {
		return invalid(field, "%s must be a string of %d to %d characters", field, min, max)
	}
	return nil
}

func validateOptString(v, field string, min, max int) error {
	if v == "" {
		return nil
	}
	return validateString(&v, field, min, max)
}

func validateMetadata(m map[string]any, field string) error {
	if m == nil {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return invalid(field, "%s must be JSON-serialisable", field)
	}
	if len(b) > MaxMetadataBytes {
		return invalid(field, "%s must encode to at most %d bytes of JSON", field, MaxMetadataBytes)
	}
	return nil
}

func isLoopback(host string) bool { return loopbackHosts[strings.ToLower(host)] }

func validateURL(v, field string, maxLength int) error {
	if v == "" {
		return nil
	}
	if len(v) > maxLength || strings.ContainsAny(v, " \t\r\n") || hasControl(v, false) {
		return invalid(field, "%s must be a URL of at most %d characters", field, maxLength)
	}
	u, err := url.Parse(v)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return invalid(field, "%s must be an absolute URL", field)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname())) {
		return invalid(field, "%s must use https", field)
	}
	return nil
}

func validateLimit(limit int) error {
	if limit != 0 && (limit < 1 || limit > 100) {
		return invalid("limit", "limit must be an integer between 1 and 100")
	}
	return nil
}

func validateCursor(cursor string) error {
	if cursor == "" {
		return nil
	}
	if len(cursor) > 200 {
		return invalid("cursor", "cursor must be an opaque value returned by a previous list call")
	}
	for _, r := range cursor {
		if r < 0x21 || r > 0x7E {
			return invalid("cursor", "cursor must be an opaque value returned by a previous list call")
		}
	}
	return nil
}

func validateQueryValue(key, v string) error {
	if !reQueryKey.MatchString(key) {
		return invalid("", "query parameter names must be alphanumeric")
	}
	if len(v) > 500 || hasControl(v, false) {
		return invalid(key, "%s has an invalid value", key)
	}
	return nil
}

// Time formats a time for the API; the zero time is omitted.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func validateDateString(v, field string) error {
	if v == "" {
		return nil
	}
	if !reDateTime.MatchString(v) {
		return invalid(field, "%s must be an RFC 3339 timestamp or a YYYY-MM-DD date", field)
	}
	return nil
}
