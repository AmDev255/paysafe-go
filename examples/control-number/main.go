// Issues a control number for an invoice, reads it back with its payment entries, then cancels it.
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
	number, err := client.ControlNumbers.Create(ctx, paysafe.CreateControlNumberParams{
		PayerName:      "Amina Juma",
		PayerPhone:     "+255712340000",
		PayerReference: "STU-2024-0091",
		Title:          "Term 2 school fees",
		Reference:      "INV-2026-0042",
		PaymentType:    "fees",
		Mode:           "instalment",
		Amount:         450000,
		ExpiresAt:      "2026-12-31",
		Instalments: []paysafe.Instalment{
			{Amount: 225000, DueAt: "2026-11-30T00:00:00Z"},
			{Amount: 225000, DueAt: "2026-12-31T00:00:00Z"},
		},
	}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Control number %s (%s) pay at %s\n", number.Formatted, number.ID, number.PayURL)

	detail, err := client.ControlNumbers.Get(ctx, number.ID, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Balance TZS %d, %d entries, %s\n", detail.Balance, len(detail.Entries), detail.PaymentStatus)
	for _, e := range detail.Entries {
		fmt.Printf("  %s %s %d via %s\n", e.PaidAt, e.Status, e.Amount, e.Method)
	}

	page, err := client.ControlNumbers.List(ctx, paysafe.ListControlNumbersParams{State: "unpaid", Limit: 5}, nil)
	if err == nil {
		for _, item := range page.Data {
			fmt.Printf("%s %s owes %d\n", item.Formatted, item.Payer.Name, item.Balance)
		}
	}

	if cancelled, err := client.ControlNumbers.Cancel(ctx, number.ID, "Issued in error during SDK example", nil); err == nil {
		fmt.Printf("Cancelled: %s\n", cancelled.Status)
	}
}
