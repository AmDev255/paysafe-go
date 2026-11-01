// Lists succeeded payments from the last 7 days, walking every page.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	client, err := paysafe.NewClient("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	page, err := client.Payments.List(ctx, paysafe.ListPaymentsParams{Status: "succeeded", CreatedGte: time.Now().Add(-7 * 24 * time.Hour), Limit: 50}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var total int64
	err = page.ForEach(ctx, func(p paysafe.Payment) error {
		ref := ""
		if p.Reference != nil {
			ref = *p.Reference
		}
		fmt.Printf("%s %s TZS %d %s\n", p.CreatedAt, p.ID, p.Amount, ref)
		total += p.Amount
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Total: TZS %d\n", total)
}
