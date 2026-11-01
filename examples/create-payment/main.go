// Creates a TZS 10,000 payment with a persisted idempotency key, then polls until it is terminal.
// Usage: go run ./examples/create-payment [+255712340000]
// Sandbox suffixes: 0000 succeeds, 1111 is declined, 4444 expires.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	phone := "+255712340000"
	if len(os.Args) > 1 {
		phone = os.Args[1]
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	orderID := "order-" + hex.EncodeToString(b)

	// Store the key with the order before sending, so a crash and restart reuses it instead of charging twice.
	keyFile := filepath.Join(os.TempDir(), "paysafe-"+orderID+".key")
	key := orderID + "-attempt-1"
	if data, err := os.ReadFile(keyFile); err == nil {
		key = string(data)
	}
	_ = os.WriteFile(keyFile, []byte(key), 0o600)
	defer os.Remove(keyFile)

	client, err := paysafe.NewClient("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	payment, err := client.Payments.Create(ctx, paysafe.CreatePaymentParams{
		Amount:      10000,
		Currency:    "TZS",
		Customer:    paysafe.CreateCustomerParams{Phone: phone, Name: "Amina Juma"},
		Reference:   orderID,
		Description: "Order " + orderID,
		Metadata:    map[string]any{"order_id": orderID},
	}, &paysafe.RequestOptions{IdempotencyKey: key})
	if err != nil {
		fail(err)
	}
	replayed := ""
	if payment.LastResponse.IdempotentReplayed {
		replayed = ", replayed"
	}
	fmt.Printf("Created %s status=%s (HTTP %d%s)\n", payment.ID, payment.Status, payment.LastResponse.Status, replayed)

	final, err := client.Payments.WaitForTerminal(ctx, payment.ID, &paysafe.WaitOptions{OnUpdate: func(s string) { fmt.Printf("  %s\n", s) }})
	if err != nil {
		var pe *paysafe.PollingTimeoutError
		if errors.As(err, &pe) {
			fmt.Println("Still pending after the timeout; rely on the webhook")
			return
		}
		fail(err)
	}
	if final.IsSuccessful() {
		fmt.Printf("Paid: amount %d fee %d net %d\n", final.Amount, final.Fee, final.Net)
	} else {
		code := ""
		if final.Failure != nil && final.Failure.Code != nil {
			code = *final.Failure.Code
		}
		fmt.Printf("Not paid: %s %s\n", final.Status, code)
	}
}

func fail(err error) {
	var apiErr *paysafe.Error
	if errors.As(err, &apiErr) {
		fmt.Fprintf(os.Stderr, "API error %s (%d): %s [%s]\n", apiErr.Code, apiErr.Status, apiErr.Message, apiErr.RequestID)
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
