// Reads the balance, the latest ledger entries and recent customers.
package main

import (
	"context"
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
	ctx := context.Background()
	balance, err := client.Balance.Get(ctx, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Available TZS %d, pending TZS %d\n", balance.AvailableAmount("TZS"), balance.PendingAmount("TZS"))

	if page, err := client.Balance.Transactions(ctx, paysafe.ListParams{Limit: 5}, nil); err == nil {
		for _, tx := range page.Data {
			fmt.Printf("%s %-6s %-9s %8d → %d\n", tx.CreatedAt, tx.Direction, tx.Bucket, tx.Amount, tx.BalanceAfter)
		}
	}
	if page, err := client.Customers.List(ctx, paysafe.ListCustomersParams{Limit: 5}, nil); err == nil {
		for _, c := range page.Data {
			fmt.Printf("%s %s paid %d times, TZS %d\n", c.ID, c.Phone, c.PaymentCount, c.TotalPaid)
		}
	}
}
