package paysafe_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	paysafe "github.com/AmDev255/paysafe-go"
)

const (
	contractID  = "pay_01J7A9Q0K3M4N5P6R7S8T9V0W1"
	contractKey = "order-8812-attempt-1"
)

var keyed = &paysafe.RequestOptions{IdempotencyKey: contractKey}

func unmarshalInto[T any](t testing.TB, raw json.RawMessage) T {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func fixtureTime(t testing.TB, f operationFixture, key string) time.Time {
	s, _ := f.Query[key].(string)
	if s == "" {
		return time.Time{}
	}
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func str(f operationFixture, key string) string { s, _ := f.Query[key].(string); return s }
func num(f operationFixture, key string) int    { n, _ := f.Query[key].(float64); return int(n) }

// calls maps every operationId to the SDK method that implements it.
func calls(t testing.TB) map[string]func(context.Context, *paysafe.Client, operationFixture) (any, error) {
	return map[string]func(context.Context, *paysafe.Client, operationFixture) (any, error){
		"createPayment": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Payments.Create(ctx, unmarshalInto[paysafe.CreatePaymentParams](t, f.Body), keyed)
		},
		"listPayments": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Payments.List(ctx, paysafe.ListPaymentsParams{Limit: num(f, "limit"), Cursor: str(f, "cursor"), Status: str(f, "status"), Reference: str(f, "reference"), CustomerPhone: str(f, "customer_phone"), CreatedGte: fixtureTime(t, f, "created_gte"), CreatedLte: fixtureTime(t, f, "created_lte")}, nil)
		},
		"getPayment": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Payments.Get(ctx, contractID, nil)
		},
		"cancelPayment": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			p := unmarshalInto[paysafe.CancelPaymentParams](t, f.Body)
			return c.Payments.Cancel(ctx, contractID, &p, keyed)
		},
		"createPaymentLink": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			p := unmarshalInto[paysafe.CreatePaymentLinkParams](t, f.Body)
			var raw map[string]any
			_ = json.Unmarshal(f.Body, &raw)
			if s, ok := raw["expires_at"].(string); ok {
				p.ExpiresAt, _ = time.Parse(time.RFC3339, s)
			}
			return c.PaymentLinks.Create(ctx, p, nil)
		},
		"listPaymentLinks": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.PaymentLinks.List(ctx, paysafe.ListPaymentLinksParams{Limit: num(f, "limit"), Cursor: str(f, "cursor"), Status: str(f, "status")}, nil)
		},
		"getPaymentLink": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.PaymentLinks.Get(ctx, contractID, nil)
		},
		"updatePaymentLink": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.PaymentLinks.Update(ctx, contractID, unmarshalInto[paysafe.UpdatePaymentLinkParams](t, f.Body), nil)
		},
		"createControlNumber": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.ControlNumbers.Create(ctx, unmarshalInto[paysafe.CreateControlNumberParams](t, f.Body), nil)
		},
		"listControlNumbers": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.ControlNumbers.List(ctx, paysafe.ListControlNumbersParams{Limit: num(f, "limit"), Cursor: str(f, "cursor"), Status: str(f, "status"), State: str(f, "state"), Batch: str(f, "batch"), Customer: str(f, "customer"), Q: str(f, "q")}, nil)
		},
		"getControlNumber": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.ControlNumbers.Get(ctx, contractID, nil)
		},
		"cancelControlNumber": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			var b struct {
				Reason string `json:"reason"`
			}
			_ = json.Unmarshal(f.Body, &b)
			return c.ControlNumbers.Cancel(ctx, contractID, b.Reason, nil)
		},
		"createRefund": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Refunds.Create(ctx, unmarshalInto[paysafe.CreateRefundParams](t, f.Body), keyed)
		},
		"listRefunds": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Refunds.List(ctx, paysafe.ListRefundsParams{Limit: num(f, "limit"), Cursor: str(f, "cursor"), Payment: str(f, "payment"), Status: str(f, "status")}, nil)
		},
		"getRefund": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Refunds.Get(ctx, contractID, nil)
		},
		"getBalance": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Balance.Get(ctx, nil)
		},
		"listTransactions": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Balance.Transactions(ctx, paysafe.ListParams{Limit: num(f, "limit"), Cursor: str(f, "cursor")}, nil)
		},
		"listCustomers": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Customers.List(ctx, paysafe.ListCustomersParams{Limit: num(f, "limit"), Cursor: str(f, "cursor"), Search: str(f, "search")}, nil)
		},
		"getCustomer": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Customers.Get(ctx, contractID, nil)
		},
		"createTransfer": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Transfers.Create(ctx, unmarshalInto[paysafe.CreateTransferParams](t, f.Body), keyed)
		},
		"listTransfers": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Transfers.List(ctx, paysafe.ListParams{Limit: num(f, "limit"), Cursor: str(f, "cursor")}, nil)
		},
		"getTransfer": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Transfers.Get(ctx, contractID, nil)
		},
		"listWebhookEndpoints": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.WebhookEndpoints.List(ctx, nil)
		},
		"createWebhookEndpoint": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.WebhookEndpoints.Create(ctx, unmarshalInto[paysafe.CreateWebhookEndpointParams](t, f.Body), nil)
		},
		"getWebhookEndpoint": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.WebhookEndpoints.Get(ctx, contractID, nil)
		},
		"updateWebhookEndpoint": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.WebhookEndpoints.Update(ctx, contractID, unmarshalInto[paysafe.UpdateWebhookEndpointParams](t, f.Body), nil)
		},
		"deleteWebhookEndpoint": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return nil, c.WebhookEndpoints.Delete(ctx, contractID, nil)
		},
		"listEvents": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Events.List(ctx, paysafe.ListEventsParams{Limit: num(f, "limit"), Cursor: str(f, "cursor"), Type: str(f, "type"), From: fixtureTime(t, f, "from"), To: fixtureTime(t, f, "to"), ResourceID: str(f, "resourceId")}, nil)
		},
		"getEvent": func(ctx context.Context, c *paysafe.Client, f operationFixture) (any, error) {
			return c.Events.Get(ctx, contractID, nil)
		},
	}
}

func operations(t testing.TB) []string {
	var index struct {
		Operations []struct {
			OperationID string `json:"operationId"`
		} `json:"operations"`
	}
	loadFixture(t, "index", &index)
	var ids []string
	for _, o := range index.Operations {
		ids = append(ids, o.OperationID)
	}
	return ids
}

func TestEveryOperationHasACall(t *testing.T) {
	ids := operations(t)
	got := make([]string, 0, len(ids))
	for id := range calls(t) {
		got = append(got, id)
	}
	sort.Strings(ids)
	sort.Strings(got)
	if !reflect.DeepEqual(ids, got) {
		t.Fatalf("contract %v\nsdk %v", ids, got)
	}
}

// canonical re-encodes JSON so key order and number formatting do not matter.
func canonical(t testing.TB, raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("invalid json %s: %v", raw, err)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func TestRequestMatchesFixtureAndSuccessParses(t *testing.T) {
	for _, id := range operations(t) {
		id := id
		t.Run(id, func(t *testing.T) {
			f := loadOperation(t, id)
			code, body, headers := successOf(f)
			c, ft, _ := newClient(t, 4)
			ft.JSON(code, body, headers)
			result, err := calls(t)[id](context.Background(), c, f)
			if err != nil {
				t.Fatal(err)
			}
			req := ft.Last()
			if req.Method != f.Method || req.URL.Path != f.ResolvedPath {
				t.Fatalf("%s %s", req.Method, req.URL.Path)
			}
			want := url.Values{}
			for k, v := range f.Query {
				if v == nil {
					continue
				}
				switch x := v.(type) {
				case float64:
					want.Set(k, strconv.FormatFloat(x, 'f', -1, 64))
				case bool:
					want.Set(k, strconv.FormatBool(x))
				default:
					want.Set(k, fmt.Sprint(x))
				}
			}
			if got := req.URL.Query(); !reflect.DeepEqual(got, want) && !(len(got) == 0 && len(want) == 0) {
				t.Fatalf("query got %v want %v", got, want)
			}
			if req.Header.Get("Authorization") != "Bearer "+testKey {
				t.Fatal("authorization header")
			}
			if _, ok := f.Headers["Idempotency-Key"]; ok {
				if req.Header.Get("Idempotency-Key") != contractKey {
					t.Fatal("idempotency key not sent")
				}
			} else if req.Header.Get("Idempotency-Key") != "" {
				t.Fatal("unexpected idempotency key")
			}
			if len(f.Body) > 0 && string(f.Body) != "null" {
				if req.Header.Get("Content-Type") != "application/json" {
					t.Fatal("content type")
				}
				if canonical(t, ft.LastBody()) != canonical(t, f.Body) {
					t.Fatalf("body\n got %s\nwant %s", ft.LastBody(), f.Body)
				}
			} else if len(ft.LastBody()) != 0 {
				t.Fatalf("unexpected body %s", ft.LastBody())
			}

			if len(body) == 0 || string(body) == "null" {
				if result != nil {
					t.Fatalf("expected nil result, got %v", result)
				}
				return
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			switch {
			case raw["object"] == "list" && id != "listWebhookEndpoints":
				v := reflect.ValueOf(result).Elem()
				data := v.FieldByName("Data")
				if data.Len() != len(raw["data"].([]any)) || v.FieldByName("HasMore").Bool() != raw["has_more"].(bool) {
					t.Fatalf("page %v", result)
				}
			case id == "listWebhookEndpoints":
				if reflect.ValueOf(result).Len() != len(raw["data"].([]any)) {
					t.Fatal("endpoint count")
				}
			default:
				v := reflect.ValueOf(result).Elem()
				if idField := v.FieldByName("ID"); idField.IsValid() && idField.String() != raw["id"] {
					t.Fatalf("id %s", idField.String())
				}
				lr := v.FieldByName("LastResponse").Interface().(*paysafe.Response)
				if lr.RequestID != requestID {
					t.Fatal("request id")
				}
				// Round-trip: marshalling the model must reproduce every field the API sent.
				var back map[string]any
				_ = json.Unmarshal([]byte(mustJSON(t, result)), &back)
				for k, want := range raw {
					if got, ok := back[k]; !ok || !reflect.DeepEqual(got, want) {
						if k == "entries" || k == "deliveries" || k == "secret" {
							continue
						}
						t.Fatalf("field %s: got %v want %v", k, got, want)
					}
				}
			}
		})
	}
}

func TestEveryDocumentedErrorRaisesTheMappedError(t *testing.T) {
	for _, id := range operations(t) {
		f := loadOperation(t, id)
		for code, r := range f.Responses {
			if strings.HasPrefix(code, "2") {
				continue
			}
			status, _ := strconv.Atoi(strings.SplitN(code, "_", 2)[0])
			c, ft, _ := newClient(t, 0)
			ft.JSON(status, r.Body, r.Headers)
			_, err := calls(t)[id](context.Background(), c, f)
			var ae *paysafe.Error
			if !errors.As(err, &ae) {
				t.Fatalf("%s %s: %v", id, code, err)
			}
			var env struct {
				Error struct {
					RequestID string `json:"request_id"`
					DocURL    string `json:"doc_url"`
				} `json:"error"`
			}
			_ = json.Unmarshal(r.Body, &env)
			if ae.Status != status || ae.Code != r.Code || ae.RequestID != env.Error.RequestID || ae.DocURL != env.Error.DocURL {
				t.Fatalf("%s %s: %+v", id, code, ae)
			}
			if ra, ok := r.Headers["Retry-After"]; ok {
				if n, _ := strconv.Atoi(ra); ae.RetryAfter != n {
					t.Fatalf("%s %s: retry after %d", id, code, ae.RetryAfter)
				}
			}
		}
	}
}
