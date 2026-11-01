// Lists recent payment events and shows the delivery attempts for the newest one.
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
	events, err := client.Events.List(ctx, paysafe.ListEventsParams{Type: "payment.succeeded", From: time.Now().Add(-24 * time.Hour), Limit: 10}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, e := range events.Data {
		id := ""
		if p, ok := e.Payment(); ok {
			id = p.ID
		}
		fmt.Printf("%s %s %s\n", e.CreatedAt, e.ID, id)
	}
	if len(events.Data) > 0 {
		detail, err := client.Events.Get(ctx, events.Data[0].ID, nil)
		if err == nil {
			for _, d := range detail.Deliveries {
				status, latency := "-", "-"
				if d.ResponseStatus != nil {
					status = fmt.Sprint(*d.ResponseStatus)
				}
				if d.LatencyMs != nil {
					latency = fmt.Sprint(*d.LatencyMs)
				}
				fmt.Printf("  attempt %d %s HTTP %s %sms\n", d.Attempt, d.Status, status, latency)
			}
		}
	}
}
