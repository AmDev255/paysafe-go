// Proves the key works with a read-only call and prints the rate-limit budget.
// Usage: PAYSAFE_SECRET_KEY=paysafe_test_... go run ./examples/auth
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	client, err := paysafe.NewClient("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	balance, err := client.Balance.Get(context.Background(), nil)
	if err != nil {
		var apiErr *paysafe.Error
		if errors.As(err, &apiErr) {
			fmt.Fprintf(os.Stderr, "API error %s (%d): %s [%s]\n", apiErr.Code, apiErr.Status, apiErr.Message, apiErr.RequestID)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
	fmt.Printf("Environment : %s\n", client.Environment)
	fmt.Printf("Available   : TZS %d\n", balance.AvailableAmount("TZS"))
	fmt.Printf("Pending     : TZS %d\n", balance.PendingAmount("TZS"))
	if rl := balance.LastResponse.RateLimit; rl != nil {
		fmt.Printf("Rate limit  : %d remaining of %d\n", rl.Remaining, rl.Limit)
	}
	fmt.Printf("Request id  : %s\n", balance.LastResponse.RequestID)
}
