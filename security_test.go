package paysafe_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	paysafe "github.com/AmDev255/paysafe-go"
)

func TestIDsThatCouldChangeThePathAreRejected(t *testing.T) {
	for _, id := range []string{"../balance", "pay_1/cancel", "pay_1?limit=1", "pay_1#x", "pay%2F1", "pay 1", "pay_1\n", "", strings.Repeat("a", 65)} {
		c, ft, _ := newClient(t, 4)
		_, err := c.Payments.Get(context.Background(), id, nil)
		var ire *paysafe.InvalidRequestError
		if !errors.As(err, &ire) {
			t.Fatalf("%q: expected InvalidRequestError, got %v", id, err)
		}
		if len(ft.Requests) != 0 {
			t.Fatalf("%q: a request was sent", id)
		}
	}
}

func TestIDsAndQueryValuesAreEncoded(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	_, body, _ := successOf(loadOperation(t, "getPayment"))
	ft.JSON(200, body, nil)
	if _, err := c.Payments.Get(context.Background(), "pay_01J7A9Q0K3M4N5P6R7S8T9V0W1", nil); err != nil {
		t.Fatal(err)
	}
	if got := ft.Last().URL.String(); got != "https://api.test.invalid/v1/payments/pay_01J7A9Q0K3M4N5P6R7S8T9V0W1" {
		t.Fatal(got)
	}
	_, list, _ := successOf(loadOperation(t, "listPayments"))
	ft.JSON(200, list, nil)
	if _, err := c.Payments.List(context.Background(), paysafe.ListPaymentsParams{Reference: "order&x=1 #2", Limit: 5}, nil); err != nil {
		t.Fatal(err)
	}
	if got := ft.Last().URL.RawQuery; got != "limit=5&reference=order%26x%3D1+%232" {
		t.Fatal(got)
	}
}

func TestHeaderInjectionAndProtectedHeadersAreRejected(t *testing.T) {
	c, _, _ := newClient(t, 4)
	for _, h := range []map[string]string{{"X-Custom": "abc\r\nX-Injected: 1"}, {"X-Custom": "abc\nX"}, {"X-Custom": "abc\x00"}, {"authorization": "Bearer other"}} {
		_, err := c.Balance.Get(context.Background(), &paysafe.RequestOptions{Headers: h})
		var ire *paysafe.InvalidRequestError
		if !errors.As(err, &ire) {
			t.Fatalf("%v: expected InvalidRequestError, got %v", h, err)
		}
	}
	for _, o := range []*paysafe.RequestOptions{{IdempotencyKey: "key\r\nX: 1"}, {RequestID: "bad id"}} {
		if _, err := c.Balance.Get(context.Background(), o); err == nil {
			t.Fatalf("%+v accepted", o)
		}
	}
}

func TestUnsafeBaseURLsAreRejected(t *testing.T) {
	hc := &http.Client{Transport: &fakeTransport{}}
	for _, u := range []string{"http://api.paysafe.co.tz", "https://user:pass@api.paysafe.co.tz", "https://api.paysafe.co.tz/?x=1", "https://api.paysafe.co.tz/#x", "api.paysafe.co.tz", "https://api.paysafe.co.tz\n", "ftp://api.paysafe.co.tz"} {
		var ce *paysafe.ConfigError
		if _, err := paysafe.NewClient(testKey, paysafe.WithBaseURL(u), paysafe.WithHTTPClient(hc)); !errors.As(err, &ce) {
			t.Fatalf("%q accepted: %v", u, err)
		}
	}
	if _, err := paysafe.NewClient(testKey, paysafe.WithBaseURL("http://localhost:8088"), paysafe.WithHTTPClient(hc)); err != nil {
		t.Fatal(err)
	}
	if _, err := paysafe.NewClient(testKey, paysafe.WithBaseURL("http://api.internal"), paysafe.WithAllowInsecureHTTP(), paysafe.WithHTTPClient(hc)); err != nil {
		t.Fatal(err)
	}
	if _, err := paysafe.NewClient(testKey+"\n", paysafe.WithHTTPClient(hc)); err == nil {
		t.Fatal("key with newline accepted")
	}
}

func TestRedirectsAreNeverFollowed(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	ft.Raw(302, "", map[string]string{"Location": "https://evil.invalid/steal"})
	_, err := c.Balance.Get(context.Background(), nil)
	var ae *paysafe.Error
	if !errors.As(err, &ae) || ae.Code != "unexpected_redirect" || len(ft.Requests) != 1 {
		t.Fatalf("got %v", err)
	}
}

func TestOversizedBodiesAndMetadataAreRejectedBeforeSending(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	base := paysafe.CreatePaymentParams{Amount: 10000, Customer: paysafe.CreateCustomerParams{Phone: "+255712340000"}}
	big := base
	big.Metadata = map[string]any{"big": strings.Repeat("b", 70000)}
	if _, err := c.Payments.Create(context.Background(), big, nil); err == nil {
		t.Fatal("oversized body accepted")
	}
	meta := base
	meta.Metadata = map[string]any{"x": strings.Repeat("y", 9000)}
	if _, err := c.Payments.Create(context.Background(), meta, nil); err == nil {
		t.Fatal("oversized metadata accepted")
	}
	if len(ft.Requests) != 0 {
		t.Fatal("request sent")
	}
}

func TestAmountsPhonesAndEnumsAreValidated(t *testing.T) {
	c, _, _ := newClient(t, 4)
	ctx := context.Background()
	for _, amt := range []int64{0, 999, 10_000_001, -5} {
		_, err := c.Payments.Create(ctx, paysafe.CreatePaymentParams{Amount: amt, Customer: paysafe.CreateCustomerParams{Phone: "+255712340000"}}, nil)
		var ire *paysafe.InvalidRequestError
		if !errors.As(err, &ire) || ire.Param != "amount" {
			t.Fatalf("amount %d: %v", amt, err)
		}
	}
	for _, phone := range []string{"0712340000", "+25571234000", "+255 712 340 000", "+1255712340000", "+255712340000\n"} {
		_, err := c.Payments.Create(ctx, paysafe.CreatePaymentParams{Amount: 10000, Customer: paysafe.CreateCustomerParams{Phone: phone}}, nil)
		var ire *paysafe.InvalidRequestError
		if !errors.As(err, &ire) || ire.Param != "customer.phone" {
			t.Fatalf("phone %q: %v", phone, err)
		}
	}
	if _, err := c.Payments.Create(ctx, paysafe.CreatePaymentParams{Amount: 10000, Customer: paysafe.CreateCustomerParams{Phone: "+255712340000"}, Channel: "paypal"}, nil); err == nil {
		t.Fatal("bad channel accepted")
	}
	if _, err := c.WebhookEndpoints.Create(ctx, paysafe.CreateWebhookEndpointParams{URL: "http://merchant.example/hook"}, nil); err == nil {
		t.Fatal("http webhook url accepted")
	}
	if _, err := c.Payments.List(ctx, paysafe.ListPaymentsParams{Cursor: strings.Repeat("c", 201)}, nil); err == nil {
		t.Fatal("long cursor accepted")
	}
}

func TestOversizedResponsesAreRefusedWithoutRetry(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	ft.Raw(200, "x", map[string]string{"Content-Length": "16777217"})
	_, err := c.Balance.Get(context.Background(), nil)
	var tl *paysafe.ResponseTooLargeError
	if !errors.As(err, &tl) || len(ft.Requests) != 1 {
		t.Fatalf("got %v after %d requests", err, len(ft.Requests))
	}
}

func TestNonJSONErrorBodiesAreTruncatedAndKeyNeverLeaks(t *testing.T) {
	c, ft, _ := newClient(t, 0)
	ft.Raw(400, strings.Repeat("<b>", 5000), map[string]string{"Content-Type": "text/html"})
	_, err := c.Balance.Get(context.Background(), nil)
	var ae *paysafe.Error
	if !errors.As(err, &ae) || ae.Code != "unknown" {
		t.Fatalf("got %v", err)
	}
	if raw, _ := ae.Details["raw"].(string); len(raw) > 4096 {
		t.Fatal("raw body not truncated")
	}
	if strings.Contains(ae.Error(), testKey) || strings.Contains(mustJSON(t, ae.Headers), testKey) {
		t.Fatal("key leaked")
	}
}

func TestWebhookInputSizesAreBounded(t *testing.T) {
	var v struct {
		Secret   string `json:"secret"`
		Body     string `json:"body"`
		Header   string `json:"header"`
		NowValid int64  `json:"nowValid"`
	}
	loadFixture(t, "webhook-vector", &v)
	opts := &paysafe.VerifyOptions{Now: v.NowValid}
	if paysafe.VerifySignature(v.Secret, []byte(strings.Repeat("x", 1048577)), v.Header, opts) {
		t.Fatal("oversized payload accepted")
	}
	if paysafe.VerifySignature(v.Secret, []byte(v.Body), strings.Repeat("t", 5000), opts) {
		t.Fatal("oversized header accepted")
	}
}
