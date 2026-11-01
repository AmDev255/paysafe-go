// Refunds part of a succeeded payment and waits for the result.
// Usage: go run ./examples/refund pay_... [amount]
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/refund <payment id> [amount]")
		os.Exit(2)
	}
	params := paysafe.CreateRefundParams{Payment: os.Args[1], Reason: "requested_by_customer"}
	if len(os.Args) > 2 {
		params.Amount, _ = strconv.ParseInt(os.Args[2], 10, 64)
	}
	client, err := paysafe.NewClient("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	refund, err := client.Refunds.Create(ctx, params, &paysafe.RequestOptions{IdempotencyKey: "refund-" + os.Args[1] + "-1"})
	if err != nil {
		var apiErr *paysafe.Error
		switch {
		case paysafe.IsInsufficientFunds(err) && errors.As(err, &apiErr):
			have, _ := apiErr.Available()
			need, _ := apiErr.Requested()
			fmt.Printf("Not enough available balance: have %d, need %d\n", have, need)
		case paysafe.IsInvalidState(err):
			fmt.Printf("Cannot refund: %v\n", err)
		default:
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	fmt.Printf("Refund %s %s\n", refund.ID, refund.Status)
	final, err := client.Refunds.WaitForTerminal(ctx, refund.ID, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	reason := ""
	if final.FailureReason != nil {
		reason = *final.FailureReason
	}
	fmt.Printf("Final: %s %s\n", final.Status, reason)
}
