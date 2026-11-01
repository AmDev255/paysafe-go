# PaySafe Go SDK

Official Go library for the [PaySafe](https://paysafe.co.tz) payments API. Accept mobile-money payments in Tanzania, issue refunds, control numbers and payment links, move money between projects, and receive signed webhooks.

Part of the [PaySafe SDK directory](https://github.com/AmDev255/paysafe-sdks). API reference: https://paysafe.co.tz/docs.

## Features

- Every served PaySafe API operation (29 operations, 10 resources) with typed request and response models
- Sandbox and production selected automatically from the key prefix
- Safe retries that honour `Retry-After`, idempotency keys that survive retries, cursor pagination helpers
- Polling helpers that wait for a payment, refund or transfer to reach a terminal state
- Webhook signature verification with secret rotation support and constant-time comparison
- One structured exception per documented error code, every error carries the request id
- Configurable timeouts, base URL, logger and HTTP client

## Installation

```bash
go get github.com/AmDev255/paysafe-go
```

## Requirements

Go 1.21 or newer. No dependencies outside the standard library.

## Authentication

Create a secret key in the dashboard under API keys. Test keys start with `paysafe_test_`, live keys with `paysafe_live_`. Keep keys on the server; never ship them in browsers or mobile apps. The SDK reads `PAYSAFE_SECRET_KEY` when no key is passed.

## Sandbox setup

Use a `paysafe_test_` key. The sandbox is the same API with a simulated provider; the last four digits of the customer phone choose the outcome (`0000` succeeds, `1111` is declined, `4444` expires, see the [Sandbox guide](https://paysafe.co.tz/docs/sandbox)).

## Production setup

Switch to a `paysafe_live_` key once your organisation is verified. Register a live webhook endpoint and add your server's address to the project's IP allowlist. Nothing else changes.

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"os"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	client, err := paysafe.NewClient(os.Getenv("PAYSAFE_SECRET_KEY")) // paysafe_test_… selects the sandbox
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	payment, err := client.Payments.Create(ctx, paysafe.CreatePaymentParams{
		Amount:    10000, // TZS, integer
		Customer:  paysafe.CreateCustomerParams{Phone: "+255712340000", Name: "Amina Juma"},
		Reference: "order-8812",
	}, &paysafe.RequestOptions{IdempotencyKey: "order-8812-attempt-1"}) // persist the key before calling
	if err != nil {
		panic(err)
	}
	fmt.Println(payment.ID, payment.Status) // 202 Accepted: still pending

	final, err := client.Payments.WaitForTerminal(ctx, payment.ID, nil) // or wait for the webhook
	if err == nil && final.IsSuccessful() {
		fmt.Println("Paid, net TZS", final.Net)
	}
}
```

## Supported API resources

| Resource | Operations |
|---|---|
| Payments | create, list, get, cancel, waitForTerminal |
| Payment links | create, list, get, update |
| Control numbers | create, list, get, cancel |
| Refunds | create, list, get, waitForTerminal |
| Balance | get, transactions |
| Customers | list, get |
| Transfers | create, list, get, waitForTerminal |
| Webhook endpoints | list, create, get, update, delete |
| Events | list, get |
| Webhooks | verifySignature, constructEvent |

Planned by PaySafe and not yet served (not included): `ping`, payouts (`createPayout`, `getPayout`), `rotateWebhookEndpointSecret`.

## Usage examples

Runnable programs live in [`examples/`](examples/), one `main` package each: authentication, creating and polling a payment, listing with pagination, refunds, payment links, control numbers, transfers, webhook endpoints, events, customers and balance, and a `net/http` webhook receiver.

Every resource follows the same shape: `Create`, `List`, `Get`, plus resource-specific actions. Every method takes a `context.Context` and an optional `*RequestOptions`. Lists return a `*Page[T]` with `Data`, `HasMore`, `NextCursor`, `Next(ctx)`, `ForEach(ctx, fn)` and `All(ctx)`.

```go
// Pagination
page, _ := client.Payments.List(ctx, paysafe.ListPaymentsParams{Status: "succeeded", Limit: 100}, nil)
_ = page.ForEach(ctx, func(p paysafe.Payment) error { return nil }) // every payment across all pages

// Refunds
refund, _ := client.Refunds.Create(ctx, paysafe.CreateRefundParams{Payment: payment.ID, Amount: 5000, Reason: "requested_by_customer"}, nil)

// Payment links and control numbers
link, _ := client.PaymentLinks.Create(ctx, paysafe.CreatePaymentLinkParams{Title: "Workshop ticket", Amount: 25000}, nil)
number, _ := client.ControlNumbers.Create(ctx, paysafe.CreateControlNumberParams{PayerName: "Amina Juma", Title: "Term 2 fees", Amount: 450000}, nil)

// Transfers, balance, customers, events
transfer, _ := client.Transfers.Create(ctx, paysafe.CreateTransferParams{To: "merchant_shop", Amount: 5000}, nil)
balance, _ := client.Balance.Get(ctx, nil)
customer, _ := client.Customers.Get(ctx, "cus_…", nil)
event, _ := client.Events.Get(ctx, "evt_…", nil)

// Webhook endpoints (the secret is returned once)
endpoint, _ := client.WebhookEndpoints.Create(ctx, paysafe.CreateWebhookEndpointParams{URL: "https://example.com/paysafe", Events: []string{"payment.succeeded"}}, nil)
```

Models are plain structs with JSON tags matching the API; nullable fields are pointers. Every returned object carries `LastResponse` with `Status`, `RequestID`, `RateLimit`, `IdempotentReplayed` and the `IdempotencyKey` that was sent. `RequestOptions` holds `IdempotencyKey`, `RequestID`, `Timeout`, `MaxRetries` and extra `Headers`. The client is safe for concurrent use.

Retries: 429 and 503 wait for `Retry-After`; `idempotency_in_progress` waits one second; 5xx and network errors are retried only when the request is idempotent (GET, PATCH, DELETE, or a POST with an idempotency key). The SDK generates a UUID key for `Payments.Create`, `Refunds.Create`, `Transfers.Create` and `Payments.Cancel` when you do not pass one; pass your own key derived from the order to survive process restarts. Cancelling the context stops retries and polling.

Security: ids, filters, headers and amounts are validated before any request; path and query values are percent-encoded; the base URL must be https (plain http only for localhost or `WithAllowInsecureHTTP`); redirects are never followed; response bodies are capped at 16 MB; the key never appears in errors or logs.

## Error handling

Every non-2xx response is a `*paysafe.Error` with `Status`, `Code`, `Message`, `RequestID`, `Param`, `Details`, `DocURL`, `RetryAfter` and `RateLimit`. Classify with `errors.As` or the helpers:

```go
_, err := client.Payments.Cancel(ctx, id, nil, nil)
var apiErr *paysafe.Error
var connErr *paysafe.ConnectionError
var reqErr *paysafe.InvalidRequestError
switch {
case paysafe.IsValidation(err):    // 400: fix the request (apiErr.Param names the field)
case paysafe.IsInvalidState(err):  // 409: read the object and decide again
case paysafe.IsRateLimited(err):   // 429 after retries: back off apiErr.RetryAfter seconds
case errors.As(err, &apiErr):      // anything else from the API; apiErr.Error() is "code (status) message [request_id]"
case errors.As(err, &connErr):     // no response at all (DNS, TLS, timeout: connErr.Timeout)
case errors.As(err, &reqErr):      // rejected locally before any request was sent
}
```

Helpers: IsValidation, IsAuthentication, IsPermission (plus `IPNotAllowed()` and `ObservedIP()` on the error), IsFeatureDisabled, IsNotFound, IsInsufficientFunds (`Available()`, `Requested()`), IsConflict, IsIdempotencyConflict, IsIdempotencyInProgress, IsInvalidState, IsRateLimited (`Scope()`), IsProviderError, IsProviderUnavailable, IsServiceUnavailable, IsInternal. Other types: `ConnectionError`, `ResponseTooLargeError`, `SignatureError` (with `Reason`), `PollingTimeoutError` (with `LastSeen`), `ConfigError`, `InvalidRequestError`. Unknown codes map by HTTP status.

## Webhook integration

Verify the raw body before parsing it. `ConstructEvent` checks the `PaySafe-Signature` header (HMAC-SHA256 over `t.body`, 300 s tolerance, constant-time comparison, several `v1=` values during rotation) and returns a typed `*Event`:

```go
rawBody, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
event, err := paysafe.ConstructEvent(os.Getenv("PAYSAFE_WEBHOOK_SECRET"), rawBody, r.Header.Get(paysafe.SignatureHeader), nil)
if err != nil {
	w.WriteHeader(http.StatusBadRequest)
	return
}
if p, ok := event.Payment(); ok && event.Type == "payment.succeeded" {
	markPaid(*p.Reference) // deduplicate on event.ID first
}
```

`VerifySignature` returns a bool instead of an error; `VerifySignatureAny` and `ConstructEventAny` accept several secrets. `examples/verify-webhook` is a complete `net/http` receiver with deduplication. Check `event.Livemode` and expect deliveries to repeat or arrive out of order.

## Testing

```bash
gofmt -l . && go vet ./... && go test ./...
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

Unit, contract, network, security and webhook tests run offline against the fixtures in `tests/fixtures/` that are shared by every PaySafe SDK (HTTP is faked with a `RoundTripper`). The integration tests run only when `PAYSAFE_SECRET_KEY` holds a `paysafe_test_` key (and optionally `PAYSAFE_BASE_URL`); they refuse live keys and use the sandbox phone suffixes documented at https://paysafe.co.tz/docs/sandbox. Examples run with `go run ./examples/<name>`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Report security issues as described in [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
