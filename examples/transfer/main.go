// Sends TZS 5,000 to another PaySafe project by handle and lists recent transfers.
// Usage: go run ./examples/transfer <recipient handle>
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/transfer <recipient handle>")
		os.Exit(2)
	}
	to := os.Args[1]
	client, err := paysafe.NewClient("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	transfer, err := client.Transfers.Create(ctx, paysafe.CreateTransferParams{To: to, Amount: 5000, Note: "Shared delivery cost"},
		&paysafe.RequestOptions{IdempotencyKey: "transfer-" + to + "-" + time.Now().Format("2006-01-02")})
	if err != nil {
		var apiErr *paysafe.Error
		if paysafe.IsInsufficientFunds(err) && errors.As(err, &apiErr) {
			have, _ := apiErr.Available()
			fmt.Printf("Not enough available balance (have %d)\n", have)
		} else {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Transfer %s %s to %s (%s)\n", transfer.ID, transfer.Status, transfer.To.Name, transfer.To.Organization)
		if !transfer.IsTerminal() {
			if transfer, err = client.Transfers.WaitForTerminal(ctx, transfer.ID, nil); err == nil {
				fmt.Printf("Review finished: %s\n", transfer.Status)
			}
		}
	}

	page, err := client.Transfers.List(ctx, paysafe.ListParams{Limit: 5}, nil)
	if err == nil {
		for _, item := range page.Data {
			fmt.Printf("%s %s %d %s\n", item.CreatedAt, item.Direction, item.Amount, item.Status)
		}
	}
}
