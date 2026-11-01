package paysafe_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	paysafe "github.com/AmDev255/paysafe-go"
)

type step struct {
	Status       int               `json:"status"`
	Code         string            `json:"code"`
	Headers      map[string]string `json:"headers"`
	NetworkError bool              `json:"networkError"`
	Timeout      bool              `json:"timeout"`
	RawBody      *string           `json:"rawBody"`
}

type scenario struct {
	Name     string `json:"name"`
	Request  string `json:"request"`
	Sequence []step `json:"sequence"`
	Expect   struct {
		Attempts           int    `json:"attempts"`
		Result             string `json:"result"`
		Error              string `json:"error"`
		Code               string `json:"code"`
		SameIdempotencyKey bool   `json:"sameIdempotencyKey"`
		SleptAtLeastMs     int    `json:"sleptAtLeastMs"`
		IdempotentReplayed *bool  `json:"idempotentReplayed"`
	} `json:"expect"`
}

func callScenario(ctx context.Context, c *paysafe.Client, id string, f operationFixture) (any, error) {
	switch id {
	case "getBalance":
		return c.Balance.Get(ctx, nil)
	case "getPayment":
		return c.Payments.Get(ctx, "pay_01J7A9Q0K3M4N5P6R7S8T9V0W1", nil)
	case "listPayments":
		return c.Payments.List(ctx, paysafe.ListPaymentsParams{}, nil)
	case "createPayment":
		var p paysafe.CreatePaymentParams
		_ = json.Unmarshal(f.MinimalBody, &p)
		return c.Payments.Create(ctx, p, nil)
	case "createRefund":
		var p paysafe.CreateRefundParams
		_ = json.Unmarshal(f.MinimalBody, &p)
		return c.Refunds.Create(ctx, p, nil)
	case "createWebhookEndpoint":
		var p paysafe.CreateWebhookEndpointParams
		_ = json.Unmarshal(f.MinimalBody, &p)
		return c.WebhookEndpoints.Create(ctx, p, nil)
	}
	panic("unsupported operation " + id)
}

func matchesError(name string, err error) bool {
	var ae *paysafe.Error
	var ce *paysafe.ConnectionError
	switch name {
	case "ApiError":
		return errors.As(err, &ae)
	case "ApiConnectionError":
		return errors.As(err, &ce)
	case "IdempotencyConflictError":
		return paysafe.IsIdempotencyConflict(err)
	case "ValidationError":
		return paysafe.IsValidation(err)
	case "InternalError":
		return paysafe.IsInternal(err)
	case "ProviderError":
		return paysafe.IsProviderError(err)
	}
	return false
}

func TestRetryScenarios(t *testing.T) {
	var doc struct {
		Scenarios []scenario `json:"scenarios"`
	}
	loadFixture(t, "retry-scenarios", &doc)
	for _, sc := range doc.Scenarios {
		sc := sc
		t.Run(sc.Name, func(t *testing.T) {
			f := loadOperation(t, sc.Request)
			code, body, headers := successOf(f)
			opts := []paysafe.Option{}
			for _, s := range sc.Sequence {
				if s.Timeout {
					opts = append(opts, paysafe.WithTimeout(20*time.Millisecond))
				}
			}
			c, ft, sl := newClient(t, 4, opts...)
			for _, s := range sc.Sequence {
				switch {
				case s.NetworkError:
					ft.Err(&net.OpError{Op: "dial", Err: errors.New("connection refused")})
				case s.Timeout:
					ft.Hang()
				case s.RawBody != nil:
					ft.Raw(s.Status, *s.RawBody, map[string]string{"Content-Type": "text/html"})
				case s.Status < 300:
					h := map[string]string{}
					for k, v := range headers {
						h[k] = v
					}
					for k, v := range s.Headers {
						h[k] = v
					}
					ft.JSON(code, body, h)
				default:
					ec := s.Code
					if ec == "" {
						ec = "internal_error"
					}
					ft.JSON(s.Status, map[string]any{"error": map[string]any{"code": ec, "message": ec, "request_id": "req_x"}}, s.Headers)
				}
			}
			result, err := callScenario(context.Background(), c, sc.Request, f)
			if sc.Expect.Error != "" {
				if !matchesError(sc.Expect.Error, err) {
					t.Fatalf("expected %s, got %v", sc.Expect.Error, err)
				}
				if sc.Expect.Code != "" {
					var ae *paysafe.Error
					if !errors.As(err, &ae) || ae.Code != sc.Expect.Code {
						t.Fatalf("code %v", err)
					}
				}
			} else if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if len(ft.Requests) != sc.Expect.Attempts {
				t.Fatalf("attempts %d want %d", len(ft.Requests), sc.Expect.Attempts)
			}
			if ft.Pending() != 0 {
				t.Fatal("queued responses left")
			}
			if sc.Expect.SameIdempotencyKey {
				first := ft.Requests[0].Header.Get("Idempotency-Key")
				for _, r := range ft.Requests {
					if r.Header.Get("Idempotency-Key") != first || first == "" {
						t.Fatal("idempotency key changed between attempts")
					}
				}
			}
			if sc.Expect.SleptAtLeastMs > 0 && sl.Total() < time.Duration(sc.Expect.SleptAtLeastMs)*time.Millisecond {
				t.Fatalf("slept %v", sl.Total())
			}
			if sc.Expect.IdempotentReplayed != nil {
				if p, ok := result.(*paysafe.Payment); !ok || p.LastResponse.IdempotentReplayed != *sc.Expect.IdempotentReplayed {
					t.Fatalf("replayed %v", result)
				}
			}
		})
	}
}

func TestTimeoutIsDistinguished(t *testing.T) {
	c, ft, _ := newClient(t, 0, paysafe.WithTimeout(20*time.Millisecond))
	ft.Hang()
	_, err := c.Balance.Get(context.Background(), nil)
	var ce *paysafe.ConnectionError
	if !errors.As(err, &ce) || !ce.Timeout {
		t.Fatalf("got %v", err)
	}
}

func TestPerRequestMaxRetriesAndGeneratedKeyReuse(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	ft.JSON(503, map[string]any{"error": map[string]any{"code": "service_unavailable", "message": "x", "request_id": "r"}}, nil)
	zero := 0
	_, err := c.Payments.Create(context.Background(), paysafe.CreatePaymentParams{Amount: 10000, Customer: paysafe.CreateCustomerParams{Phone: "+255712340000"}}, &paysafe.RequestOptions{IdempotencyKey: "k-1", MaxRetries: &zero})
	if !paysafe.IsServiceUnavailable(err) || len(ft.Requests) != 1 {
		t.Fatalf("%v after %d", err, len(ft.Requests))
	}
	code, body, headers := successOf(loadOperation(t, "createPayment"))
	ft.JSON(500, map[string]any{"error": map[string]any{"code": "internal_error", "message": "x", "request_id": "r"}}, nil)
	ft.JSON(code, body, headers)
	p, err := c.Payments.Create(context.Background(), paysafe.CreatePaymentParams{Amount: 10000, Customer: paysafe.CreateCustomerParams{Phone: "+255712340000"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	key := p.LastResponse.IdempotencyKey
	if len(key) != 36 || ft.Requests[1].Header.Get("Idempotency-Key") != key || ft.Requests[2].Header.Get("Idempotency-Key") != key {
		t.Fatalf("key %q", key)
	}
}
