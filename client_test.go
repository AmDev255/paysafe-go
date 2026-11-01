package paysafe_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	paysafe "github.com/AmDev255/paysafe-go"
)

func TestEnvironmentFromKeyPrefix(t *testing.T) {
	hc := &http.Client{Transport: &fakeTransport{}}
	c, err := paysafe.NewClient(testKey, paysafe.WithHTTPClient(hc))
	if err != nil || c.Environment != paysafe.EnvironmentTest {
		t.Fatalf("test key: %v %v", c, err)
	}
	c, err = paysafe.NewClient("paysafe_live_9f2Kx7Qm3Rt8Vw2Yz5Ab1Cd4Ef6Gh9Jk", paysafe.WithHTTPClient(hc))
	if err != nil || c.Environment != paysafe.EnvironmentLive {
		t.Fatalf("live key: %v", err)
	}
	var ce *paysafe.ConfigError
	if _, err := paysafe.NewClient("sk_test_abc", paysafe.WithHTTPClient(hc)); !errors.As(err, &ce) {
		t.Fatalf("expected ConfigError, got %v", err)
	}
}

func TestMissingKey(t *testing.T) {
	t.Setenv("PAYSAFE_SECRET_KEY", "")
	if _, err := paysafe.NewClient(""); err == nil {
		t.Fatal("expected an error")
	}
}

func TestEnvKeyAndBaseURL(t *testing.T) {
	t.Setenv("PAYSAFE_SECRET_KEY", testKey)
	t.Setenv("PAYSAFE_BASE_URL", "http://127.0.0.1:8088/v1")
	ft := (&fakeTransport{}).JSON(200, balanceBody(t), nil)
	c, err := paysafe.NewClient("", paysafe.WithHTTPClient(&http.Client{Transport: ft}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Balance.Get(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got := ft.Last().URL.String(); got != "http://127.0.0.1:8088/v1/balance" {
		t.Fatalf("url %s", got)
	}
}

func balanceBody(t testing.TB) json.RawMessage {
	_, body, _ := successOf(loadOperation(t, "getBalance"))
	return body
}

func TestStandardHeaders(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	ft.JSON(200, balanceBody(t), nil)
	if _, err := c.Balance.Get(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	h := ft.Last().Header
	if h.Get("Authorization") != "Bearer "+testKey || h.Get("Accept") != "application/json" || !strings.HasPrefix(h.Get("User-Agent"), "paysafe-go/"+paysafe.Version) || h.Get("Content-Type") != "" {
		t.Fatalf("headers %v", h)
	}
}

func TestAPIVersionHeader(t *testing.T) {
	c, ft, _ := newClient(t, 4, paysafe.WithAPIVersion("2026-09-07"))
	ft.JSON(200, balanceBody(t), nil)
	if _, err := c.Balance.Get(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if ft.Last().Header.Get("PaySafe-Version") != "2026-09-07" {
		t.Fatal("missing PaySafe-Version")
	}
}

func TestResponseMetadata(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	ft.JSON(200, balanceBody(t), map[string]string{"X-RateLimit-Limit": "50", "X-RateLimit-Remaining": "49", "X-RateLimit-Reset": "1788775205"})
	b, err := c.Balance.Get(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	r := b.LastResponse
	if r.Status != 200 || r.RequestID != requestID || r.RateLimit == nil || r.RateLimit.Remaining != 49 || r.IdempotentReplayed {
		t.Fatalf("metadata %+v", r)
	}
	if strings.Contains(mustJSON(t, b), "LastResponse") {
		t.Fatal("LastResponse must not be serialised")
	}
}

func TestLowLevelRequest(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	ft.JSON(200, balanceBody(t), nil)
	resp, err := c.Request(context.Background(), "get", "v1/balance", nil, nil, nil)
	if err != nil || resp.Status != 200 || ft.Last().URL.String() != "https://api.test.invalid/v1/balance" {
		t.Fatalf("%v %v", resp, err)
	}
}

func TestLoggerWithoutSecrets(t *testing.T) {
	var records []paysafe.LogRecord
	c, ft, _ := newClient(t, 4, paysafe.WithLogger(func(r paysafe.LogRecord) { records = append(records, r) }))
	ft.JSON(200, balanceBody(t), nil)
	if _, err := c.Balance.Get(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Method != "GET" || records[0].Path != "/v1/balance" || records[0].Status != 200 || records[0].Attempt != 1 {
		t.Fatalf("records %+v", records)
	}
	if strings.Contains(mustJSON(t, records), testKey) {
		t.Fatal("key leaked into log record")
	}
}
