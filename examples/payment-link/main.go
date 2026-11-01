// Creates a hosted payment link, lists links, then pauses and archives the new one.
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
	link, err := client.PaymentLinks.Create(ctx, paysafe.CreatePaymentLinkParams{
		Title:        "Workshop ticket",
		Description:  "One seat, Saturday session",
		Amount:       25000,
		CollectName:  true,
		CollectEmail: true,
		MaxPayments:  50,
		ExpiresAt:    time.Now().Add(14 * 24 * time.Hour),
	}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Share this: %s (%s)\n", link.URL, link.ID)

	page, err := client.PaymentLinks.List(ctx, paysafe.ListPaymentLinksParams{Status: "active", Limit: 10}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, item := range page.Data {
		max := "∞"
		if item.MaxPayments != nil {
			max = fmt.Sprint(*item.MaxPayments)
		}
		fmt.Printf("%s %s paid %d/%s\n", item.ID, item.Title, item.PaidCount, max)
	}

	if paused, err := client.PaymentLinks.Pause(ctx, link.ID, nil); err == nil {
		fmt.Printf("Paused: %s state=%s\n", paused.Status, paused.State)
	}
	if archived, err := client.PaymentLinks.Archive(ctx, link.ID, nil); err == nil {
		fmt.Printf("Archived: %s\n", archived.Status)
	}
}
