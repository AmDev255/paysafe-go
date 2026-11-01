package paysafe_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	paysafe "github.com/AmDev255/paysafe-go"
)

func TestEveryDocumentedCodeMapsToItsHelper(t *testing.T) {
	cases := []struct {
		status    int
		code      string
		is        func(error) bool
		retryable bool
	}{
		{400, "validation_error", paysafe.IsValidation, false},
		{401, "authentication_error", paysafe.IsAuthentication, false},
		{403, "permission_error", paysafe.IsPermission, false},
		{403, "feature_disabled", paysafe.IsFeatureDisabled, false},
		{404, "not_found", paysafe.IsNotFound, false},
		{402, "insufficient_funds", paysafe.IsInsufficientFunds, false},
		{409, "conflict", paysafe.IsConflict, false},
		{409, "idempotency_conflict", paysafe.IsIdempotencyConflict, false},
		{409, "idempotency_in_progress", paysafe.IsIdempotencyInProgress, true},
		{409, "invalid_state", paysafe.IsInvalidState, false},
		{429, "rate_limited", paysafe.IsRateLimited, true},
		{502, "provider_error", paysafe.IsProviderError, false},
		{503, "provider_unavailable", paysafe.IsProviderUnavailable, true},
		{503, "service_unavailable", paysafe.IsServiceUnavailable, true},
		{500, "internal_error", paysafe.IsInternal, true},
	}
	for _, tc := range cases {
		c, ft, _ := newClient(t, 0)
		body := map[string]any{"error": map[string]any{"code": tc.code, "message": "m", "request_id": "req_1", "param": "amount", "details": map[string]any{"scope": "api:key"}, "doc_url": "https://paysafe.co.tz/docs/errors#" + tc.code}}
		ft.JSON(tc.status, body, map[string]string{"Retry-After": "3"})
		_, err := c.Balance.Get(context.Background(), nil)
		var ae *paysafe.Error
		if !errors.As(err, &ae) {
			t.Fatalf("%s: %v", tc.code, err)
		}
		if ae.Status != tc.status || ae.Code != tc.code || ae.RequestID != "req_1" || ae.Param != "amount" || ae.RetryAfter != 3 || ae.DocURL == "" {
			t.Fatalf("%s: %+v", tc.code, ae)
		}
		if !tc.is(err) {
			t.Fatalf("%s: helper returned false", tc.code)
		}
		if ae.Retryable() != tc.retryable {
			t.Fatalf("%s: retryable %v", tc.code, ae.Retryable())
		}
		if !strings.Contains(ae.Error(), tc.code) {
			t.Fatalf("%s: %s", tc.code, ae.Error())
		}
	}
}

func TestUnknownCodeFallsBackToStatus(t *testing.T) {
	c, ft, _ := newClient(t, 0)
	ft.JSON(404, map[string]any{"error": map[string]any{"code": "brand_new", "message": "x", "request_id": "r"}}, nil)
	_, err := c.Balance.Get(context.Background(), nil)
	if !paysafe.IsNotFound(err) {
		t.Fatalf("expected not found by status: %v", err)
	}
	ft.Raw(500, "oops", map[string]string{"X-Request-ID": "req_h"})
	_, err = c.Balance.Get(context.Background(), nil)
	var ae *paysafe.Error
	if !errors.As(err, &ae) || ae.Code != "unknown" || ae.RequestID != "req_h" || !paysafe.IsInternal(err) {
		t.Fatalf("got %v", err)
	}
}

func TestErrorHelpers(t *testing.T) {
	c, ft, _ := newClient(t, 0)
	ft.JSON(403, map[string]any{"error": map[string]any{"code": "permission_error", "message": "x", "request_id": "r", "details": map[string]any{"code": "ip_not_allowed", "ip": "2a02::1"}}}, nil)
	_, err := c.Balance.Get(context.Background(), nil)
	var ae *paysafe.Error
	if !errors.As(err, &ae) || !ae.IPNotAllowed() || ae.ObservedIP() != "2a02::1" {
		t.Fatalf("got %v", err)
	}
	ft.JSON(402, map[string]any{"error": map[string]any{"code": "insufficient_funds", "message": "x", "request_id": "r", "details": map[string]any{"available": 10, "requested": 20}}}, nil)
	_, err = c.Balance.Get(context.Background(), nil)
	if !errors.As(err, &ae) {
		t.Fatal(err)
	}
	if a, ok := ae.Available(); !ok || a != 10 {
		t.Fatalf("available %d %v", a, ok)
	}
	if r, ok := ae.Requested(); !ok || r != 20 {
		t.Fatalf("requested %d %v", r, ok)
	}
}
