package paysafe_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	paysafe "github.com/AmDev255/paysafe-go"
)

// Runs against the real sandbox when PAYSAFE_SECRET_KEY holds a paysafe_test_ key. Never uses a live key.
func sandbox(t *testing.T) *paysafe.Client {
	t.Helper()
	key := os.Getenv("PAYSAFE_SECRET_KEY")
	if !strings.HasPrefix(key, "paysafe_test_") {
		t.Skip("PAYSAFE_SECRET_KEY (test key) not set")
	}
	c, err := paysafe.NewClient("")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func token() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func TestIntegrationBalanceAndTransactions(t *testing.T) {
	c := sandbox(t)
	ctx := context.Background()
	b, err := c.Balance.Get(ctx, nil)
	if err != nil || b.Livemode || b.LastResponse.RequestID == "" {
		t.Fatalf("%v %v", b, err)
	}
	page, err := c.Balance.Transactions(ctx, paysafe.ListParams{Limit: 5}, nil)
	if err != nil || len(page.Data) > 5 {
		t.Fatalf("%v %v", page, err)
	}
}

func TestIntegrationPaymentLifecycle(t *testing.T) {
	c := sandbox(t)
	ctx := context.Background()
	p, err := c.Payments.Create(ctx, paysafe.CreatePaymentParams{Amount: 10000, Customer: paysafe.CreateCustomerParams{Phone: "+255712340000"}, Reference: "sdk-go-" + token()}, nil)
	if err != nil || p.LastResponse.Status != 202 {
		t.Fatalf("%v %v", p, err)
	}
	final, err := c.Payments.WaitForTerminal(ctx, p.ID, &paysafe.WaitOptions{Timeout: 60 * time.Second})
	if err != nil || final.Status != "succeeded" {
		t.Fatalf("%v %v", final, err)
	}
	declined, err := c.Payments.Create(ctx, paysafe.CreatePaymentParams{Amount: 10000, Customer: paysafe.CreateCustomerParams{Phone: "+255712341111"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := c.Payments.WaitForTerminal(ctx, declined.ID, &paysafe.WaitOptions{Timeout: 60 * time.Second})
	if err != nil || failed.Status != "failed" || failed.Failure == nil || failed.Failure.Code == nil || *failed.Failure.Code != "customer_declined" {
		t.Fatalf("%v %v", failed, err)
	}
	if page, err := c.Payments.List(ctx, paysafe.ListPaymentsParams{Limit: 3}, nil); err != nil || len(page.Data) == 0 {
		t.Fatalf("%v %v", page, err)
	}
	if got, err := c.Payments.Get(ctx, p.ID, nil); err != nil || got.ID != p.ID {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := c.Payments.Cancel(ctx, p.ID, nil, nil); !paysafe.IsInvalidState(err) {
		t.Fatalf("expected invalid_state, got %v", err)
	}
}

func TestIntegrationValidationAndNotFound(t *testing.T) {
	c := sandbox(t)
	ctx := context.Background()
	_, err := c.Request(ctx, "POST", "/v1/payments", nil, map[string]any{"amount": 5}, &paysafe.RequestOptions{IdempotencyKey: "sdk-go-bad-" + token()})
	if !paysafe.IsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if _, err := c.Payments.Get(ctx, "pay_00000000000000000000000000", nil); !paysafe.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestIntegrationWebhookEndpointRoundTrip(t *testing.T) {
	c := sandbox(t)
	ctx := context.Background()
	created, err := c.WebhookEndpoints.Create(ctx, paysafe.CreateWebhookEndpointParams{URL: "https://example.com/paysafe/" + token(), Events: []string{"payment.succeeded"}}, nil)
	if err != nil || !strings.HasPrefix(created.Secret, "paysafe_webhook_") {
		t.Fatalf("%v %v", created, err)
	}
	if upd, err := c.WebhookEndpoints.Update(ctx, created.ID, paysafe.UpdateWebhookEndpointParams{Status: "DISABLED"}, nil); err != nil || upd.Status != "DISABLED" {
		t.Fatalf("%v %v", upd, err)
	}
	list, err := c.WebhookEndpoints.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range list {
		found = found || e.ID == created.ID
	}
	if !found {
		t.Fatal("created endpoint not listed")
	}
	if err := c.WebhookEndpoints.Delete(ctx, created.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.WebhookEndpoints.Get(ctx, created.ID, nil); !paysafe.IsNotFound(err) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestIntegrationEventsAndCustomers(t *testing.T) {
	c := sandbox(t)
	ctx := context.Background()
	events, err := c.Events.List(ctx, paysafe.ListEventsParams{Limit: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events.Data {
		if got, err := c.Events.Get(ctx, e.ID, nil); err != nil || got.ID != e.ID {
			t.Fatalf("%v %v", got, err)
		}
	}
	customers, err := c.Customers.List(ctx, paysafe.ListCustomersParams{Limit: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, cu := range customers.Data {
		if got, err := c.Customers.Get(ctx, cu.ID, nil); err != nil || got.ID != cu.ID {
			t.Fatalf("%v %v", got, err)
		}
	}
}
