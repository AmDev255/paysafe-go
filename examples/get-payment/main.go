// Retrieves a payment by id and waits for a terminal state.
// Usage: go run ./examples/get-payment pay_01J7A9Q0K3M4N5P6R7S8T9V0W1
package main

import (
	"context"
	"fmt"
	"os"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/get-payment <payment id>")
		os.Exit(2)
	}
	client, err := paysafe.NewClient("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	payment, err := client.Payments.Get(ctx, os.Args[1], nil)
	if paysafe.IsNotFound(err) {
		fmt.Fprintln(os.Stderr, "No such payment in this project and environment")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s %s TZS %d for %s\n", payment.ID, payment.Status, payment.Amount, payment.Customer.Phone)
	if !payment.IsTerminal() {
		if payment, err = client.Payments.WaitForTerminal(ctx, payment.ID, nil); err == nil {
			fmt.Printf("Final status: %s\n", payment.Status)
		}
	}
}
