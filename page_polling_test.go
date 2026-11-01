package paysafe_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	paysafe "github.com/AmDev255/paysafe-go"
)

func withStatus(t testing.TB, body json.RawMessage, status string) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	m["status"] = status
	return m
}

func withID(t testing.TB, body json.RawMessage, id string) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	m["id"] = id
	return m
}

func TestAutoPagingKeepsFilters(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	_, payment, _ := successOf(loadOperation(t, "getPayment"))
	ft.JSON(200, map[string]any{"object": "list", "data": []any{withID(t, payment, "pay_1"), withID(t, payment, "pay_2")}, "has_more": true, "next_cursor": "c2"}, nil)
	ft.JSON(200, map[string]any{"object": "list", "data": []any{withID(t, payment, "pay_3")}, "has_more": false, "next_cursor": nil}, nil)
	page, err := c.Payments.List(context.Background(), paysafe.ListPaymentsParams{Status: "succeeded", Limit: 2}, nil)
	if err != nil || len(page.Data) != 2 || !page.HasMore {
		t.Fatalf("%v %v", page, err)
	}
	all, err := page.All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, p := range all {
		ids = append(ids, p.ID)
	}
	if len(ids) != 3 || ids[0] != "pay_1" || ids[2] != "pay_3" {
		t.Fatalf("ids %v", ids)
	}
	if len(ft.Requests) != 2 || ft.Requests[1].URL.RawQuery != "cursor=c2&limit=2&status=succeeded" {
		t.Fatalf("second request %s", ft.Requests[1].URL.RawQuery)
	}
}

func TestLastPageHasNoNext(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	ft.JSON(200, map[string]any{"object": "list", "data": []any{}, "has_more": false, "next_cursor": nil}, nil)
	page, err := c.Customers.List(context.Background(), paysafe.ListCustomersParams{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next, err := page.Next(context.Background()); next != nil || err != nil {
		t.Fatalf("next %v %v", next, err)
	}
	if mustJSON(t, page) != `{"object":"list","data":[],"has_more":false,"next_cursor":null}` {
		t.Fatal(mustJSON(t, page))
	}
}

func TestWaitForTerminalReportsUpdates(t *testing.T) {
	c, ft, sl := newClient(t, 4)
	_, payment, _ := successOf(loadOperation(t, "getPayment"))
	for _, s := range []string{"pending", "processing", "processing", "succeeded"} {
		ft.JSON(200, withStatus(t, payment, s), nil)
	}
	var seen []string
	final, err := c.Payments.WaitForTerminal(context.Background(), "pay_01J7A9Q0K3M4N5P6R7S8T9V0W1", &paysafe.WaitOptions{OnUpdate: func(s string) { seen = append(seen, s) }})
	if err != nil || final.Status != "succeeded" || !final.IsSuccessful() {
		t.Fatalf("%v %v", final, err)
	}
	if len(seen) != 3 || seen[1] != "processing" || len(sl.Durations) != 3 || sl.Durations[0] != 2*time.Second {
		t.Fatalf("seen %v sleeps %v", seen, sl.Durations)
	}
}

func TestWaitTimesOutWithLastObject(t *testing.T) {
	c, ft, _ := newClient(t, 4)
	_, payment, _ := successOf(loadOperation(t, "getPayment"))
	for i := 0; i < 4; i++ {
		ft.JSON(200, withStatus(t, payment, "processing"), nil)
	}
	_, err := c.Payments.WaitForTerminal(context.Background(), "pay_01J7A9Q0K3M4N5P6R7S8T9V0W1", &paysafe.WaitOptions{Interval: 2 * time.Second, Timeout: 6 * time.Second})
	var pe *paysafe.PollingTimeoutError
	if !errors.As(err, &pe) || pe.LastSeen.(*paysafe.Payment).Status != "processing" || len(ft.Requests) != 4 {
		t.Fatalf("got %v after %d requests", err, len(ft.Requests))
	}
}

func TestWaitCanBeCancelled(t *testing.T) {
	c, ft, _ := newClient(t, 4, paysafe.WithTestSleep(func(ctx context.Context, d time.Duration) error { return ctx.Err() }))
	_, refund, _ := successOf(loadOperation(t, "getRefund"))
	ft.JSON(200, withStatus(t, refund, "pending"), nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Refunds.WaitForTerminal(ctx, "rfd_01J7A9Q0K3M4N5P6R7S8T9V0W1", nil)
	var pe *paysafe.PollingTimeoutError
	if !errors.As(err, &pe) {
		t.Fatalf("got %v", err)
	}
}

func TestWaitSlowsDownAfterThirtySeconds(t *testing.T) {
	c, ft, sl := newClient(t, 4)
	_, transfer, _ := successOf(loadOperation(t, "getTransfer"))
	for i := 0; i < 20; i++ {
		ft.JSON(200, withStatus(t, transfer, "in_review"), nil)
	}
	ft.JSON(200, withStatus(t, transfer, "completed"), nil)
	if _, err := c.Transfers.WaitForTerminal(context.Background(), "trf_01J7A9Q0K3M4N5P6R7S8T9V0W1", nil); err != nil {
		t.Fatal(err)
	}
	if sl.Durations[0] != 2*time.Second || sl.Durations[len(sl.Durations)-1] != 5*time.Second {
		t.Fatalf("sleeps %v", sl.Durations)
	}
}
