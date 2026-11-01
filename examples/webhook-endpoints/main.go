// Registers a webhook endpoint, stores its secret, disables it and deletes it again.
// Usage: go run ./examples/webhook-endpoints https://your-server.example/paysafe/webhook
package main

import (
	"context"
	"fmt"
	"os"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/webhook-endpoints <https url>")
		os.Exit(2)
	}
	client, err := paysafe.NewClient("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	endpoint, err := client.WebhookEndpoints.Create(ctx, paysafe.CreateWebhookEndpointParams{
		URL:         os.Args[1],
		Description: "Order fulfilment",
		Events:      []string{"payment.succeeded", "payment.failed", "refund.succeeded"},
	}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// The secret is returned once. Put it in your secret store as PAYSAFE_WEBHOOK_SECRET.
	fmt.Printf("Endpoint %s created; secret starts with %.20s…\n", endpoint.ID, endpoint.Secret)

	list, _ := client.WebhookEndpoints.List(ctx, nil)
	for _, item := range list {
		fmt.Printf("%s %s %s streak=%d\n", item.ID, item.Status, item.URL, item.FailureStreak)
	}

	if disabled, err := client.WebhookEndpoints.Update(ctx, endpoint.ID, paysafe.UpdateWebhookEndpointParams{Status: "DISABLED"}, nil); err == nil {
		fmt.Printf("Now %s\n", disabled.Status)
	}
	if err := client.WebhookEndpoints.Delete(ctx, endpoint.ID, nil); err == nil {
		fmt.Println("Deleted")
	}
}
