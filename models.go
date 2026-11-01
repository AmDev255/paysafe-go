package paysafe

import "encoding/json"

// Models mirror the API JSON. Unknown fields are kept in Raw so forward-compatible data is never lost.

type Customer struct {
	ID           string    `json:"id"`
	Object       string    `json:"object"`
	Phone        string    `json:"phone"`
	Name         *string   `json:"name"`
	Email        *string   `json:"email"`
	TotalPaid    int64     `json:"total_paid"`
	PaymentCount int64     `json:"payment_count"`
	FirstSeenAt  string    `json:"first_seen_at"`
	LastSeenAt   string    `json:"last_seen_at"`
	LastResponse *Response `json:"-"`
}

type PaymentCustomer struct {
	Phone string  `json:"phone"`
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type PaymentFailure struct {
	Code    *string `json:"code"`
	Message *string `json:"message"`
}

type Payment struct {
	ID             string          `json:"id"`
	Object         string          `json:"object"`
	Status         string          `json:"status"`
	Amount         int64           `json:"amount"`
	Currency       string          `json:"currency"`
	AmountRefunded int64           `json:"amount_refunded"`
	Fee            int64           `json:"fee"`
	Net            int64           `json:"net"`
	Customer       PaymentCustomer `json:"customer"`
	Channel        *string         `json:"channel"`
	Reference      *string         `json:"reference"`
	Description    *string         `json:"description"`
	PaymentLink    *string         `json:"payment_link"`
	Metadata       map[string]any  `json:"metadata"`
	Failure        *PaymentFailure `json:"failure"`
	Livemode       bool            `json:"livemode"`
	ExpiresAt      *string         `json:"expires_at"`
	SucceededAt    *string         `json:"succeeded_at"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
	LastResponse   *Response       `json:"-"`
}

var paymentTerminal = map[string]bool{"succeeded": true, "failed": true, "cancelled": true, "expired": true, "partially_refunded": true, "refunded": true}
var paymentSuccessful = map[string]bool{"succeeded": true, "partially_refunded": true, "refunded": true}

func (p *Payment) IsTerminal() bool   { return paymentTerminal[p.Status] }
func (p *Payment) IsSuccessful() bool { return paymentSuccessful[p.Status] }

type PaymentLink struct {
	ID             string    `json:"id"`
	Object         string    `json:"object"`
	URL            string    `json:"url"`
	Title          string    `json:"title"`
	Description    *string   `json:"description"`
	AmountType     string    `json:"amount_type"`
	Amount         *int64    `json:"amount"`
	MinAmount      *int64    `json:"min_amount"`
	MaxAmount      *int64    `json:"max_amount"`
	Currency       string    `json:"currency"`
	CollectName    bool      `json:"collect_name"`
	CollectEmail   bool      `json:"collect_email"`
	MaxPayments    *int64    `json:"max_payments"`
	PaidCount      int64     `json:"paid_count"`
	PaidAmount     int64     `json:"paid_amount"`
	ExpiresAt      *string   `json:"expires_at"`
	Status         string    `json:"status"`
	State          string    `json:"state"`
	SuccessMessage *string   `json:"success_message"`
	RedirectURL    *string   `json:"redirect_url"`
	Logo           *string   `json:"logo"`
	LogoURL        *string   `json:"logo_url"`
	AccentColor    *string   `json:"accent_color"`
	Livemode       bool      `json:"livemode"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
	LastResponse   *Response `json:"-"`
}

type ControlNumberPayer struct {
	Name      string  `json:"name"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	Reference *string `json:"reference"`
}

type ControlNumber struct {
	ID              string               `json:"id"`
	Object          string               `json:"object"`
	Number          string               `json:"number"`
	Formatted       string               `json:"formatted"`
	Batch           *string              `json:"batch"`
	Customer        *string              `json:"customer"`
	Payer           ControlNumberPayer   `json:"payer"`
	Title           string               `json:"title"`
	Description     *string              `json:"description"`
	Reference       *string              `json:"reference"`
	PaymentType     *string              `json:"payment_type"`
	Mode            string               `json:"mode"`
	Amount          int64                `json:"amount"`
	PaidAmount      int64                `json:"paid_amount"`
	Balance         int64                `json:"balance"`
	ProgressPercent int64                `json:"progress_percent"`
	Currency        string               `json:"currency"`
	PaymentCount    int64                `json:"payment_count"`
	Status          string               `json:"status"`
	PaymentStatus   string               `json:"payment_status"`
	Notes           *string              `json:"notes"`
	Metadata        map[string]any       `json:"metadata"`
	IssuedAt        string               `json:"issued_at"`
	ExpiresAt       *string              `json:"expires_at"`
	PaidAt          *string              `json:"paid_at"`
	LastPaidAt      *string              `json:"last_paid_at"`
	CancelledAt     *string              `json:"cancelled_at"`
	CancelReason    *string              `json:"cancel_reason"`
	PayURL          string               `json:"pay_url"`
	Livemode        bool                 `json:"livemode"`
	Image           *string              `json:"image"`
	ImageURL        *string              `json:"image_url"`
	CreatedAt       string               `json:"created_at"`
	Entries         []ControlNumberEntry `json:"entries,omitempty"`
	LastResponse    *Response            `json:"-"`
}

var controlNumberTerminal = map[string]bool{"paid": true, "expired": true, "cancelled": true}

func (c *ControlNumber) IsTerminal() bool { return controlNumberTerminal[c.Status] }

type ControlNumberEntry struct {
	ID                string  `json:"id"`
	Object            string  `json:"object"`
	ControlNumber     string  `json:"control_number"`
	Source            string  `json:"source"`
	Payment           *string `json:"payment"`
	Amount            int64   `json:"amount"`
	Currency          string  `json:"currency"`
	Method            string  `json:"method"`
	ProviderReference *string `json:"provider_reference"`
	PaidAt            string  `json:"paid_at"`
	Status            string  `json:"status"`
	ReceiptNumber     *string `json:"receipt_number"`
	RecordedBy        *string `json:"recorded_by"`
	VerifiedBy        *string `json:"verified_by"`
	VerifiedAt        *string `json:"verified_at"`
	RejectedAt        *string `json:"rejected_at"`
	RejectReason      *string `json:"reject_reason"`
	ReversedAt        *string `json:"reversed_at"`
	ReverseReason     *string `json:"reverse_reason"`
	Note              *string `json:"note"`
	CreatedAt         string  `json:"created_at"`
}

type TransferParty struct {
	Handle       *string `json:"handle"`
	Name         string  `json:"name"`
	Organization string  `json:"organization"`
	Verified     bool    `json:"verified"`
}

type Transfer struct {
	ID           string        `json:"id"`
	Object       string        `json:"object"`
	Direction    string        `json:"direction"`
	Amount       int64         `json:"amount"`
	Currency     string        `json:"currency"`
	Note         *string       `json:"note"`
	Status       string        `json:"status"`
	From         TransferParty `json:"from"`
	To           TransferParty `json:"to"`
	Livemode     bool          `json:"livemode"`
	CreatedAt    string        `json:"created_at"`
	CompletedAt  *string       `json:"completed_at"`
	LastResponse *Response     `json:"-"`
}

var transferTerminal = map[string]bool{"completed": true, "rejected": true, "reversed": true}

func (t *Transfer) IsTerminal() bool { return transferTerminal[t.Status] }

type Refund struct {
	ID            string         `json:"id"`
	Object        string         `json:"object"`
	Payment       string         `json:"payment"`
	Amount        int64          `json:"amount"`
	Currency      string         `json:"currency"`
	Status        string         `json:"status"`
	Reason        *string        `json:"reason"`
	FailureReason *string        `json:"failure_reason"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     string         `json:"created_at"`
	CompletedAt   *string        `json:"completed_at"`
	LastResponse  *Response      `json:"-"`
}

var refundTerminal = map[string]bool{"succeeded": true, "failed": true, "rejected": true}

func (r *Refund) IsTerminal() bool   { return refundTerminal[r.Status] }
func (r *Refund) IsSuccessful() bool { return r.Status == "succeeded" }

type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type BalanceObject struct {
	Object       string    `json:"object"`
	Livemode     bool      `json:"livemode"`
	Available    []Money   `json:"available"`
	Pending      []Money   `json:"pending"`
	LastResponse *Response `json:"-"`
}

func amountFor(items []Money, currency string) int64 {
	for _, m := range items {
		if m.Currency == currency {
			return m.Amount
		}
	}
	return 0
}

func (b *BalanceObject) AvailableAmount(currency string) int64 {
	return amountFor(b.Available, currency)
}
func (b *BalanceObject) PendingAmount(currency string) int64 { return amountFor(b.Pending, currency) }

type Transaction struct {
	ID           string         `json:"id"`
	Object       string         `json:"object"`
	Type         string         `json:"type"`
	Direction    string         `json:"direction"`
	Bucket       string         `json:"bucket"`
	Amount       int64          `json:"amount"`
	BalanceAfter int64          `json:"balance_after"`
	Currency     string         `json:"currency"`
	Source       map[string]any `json:"source"`
	Description  *string        `json:"description"`
	CreatedAt    string         `json:"created_at"`
}

type WebhookEndpoint struct {
	ID             string   `json:"id"`
	Object         string   `json:"object"`
	Livemode       bool     `json:"livemode"`
	URL            string   `json:"url"`
	Description    *string  `json:"description"`
	Events         []string `json:"events"`
	Status         string   `json:"status"`
	FailureStreak  int64    `json:"failure_streak"`
	DisabledReason *string  `json:"disabled_reason"`
	DisabledAt     *string  `json:"disabled_at"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	// Secret is only present on the create response.
	Secret       string    `json:"secret,omitempty"`
	LastResponse *Response `json:"-"`
}

func (w *WebhookEndpoint) IsEnabled() bool { return w.Status == "ENABLED" }

type WebhookDelivery struct {
	ID             string  `json:"id"`
	Object         string  `json:"object"`
	EndpointID     string  `json:"endpoint_id"`
	EventID        string  `json:"event_id"`
	Attempt        int64   `json:"attempt"`
	Status         string  `json:"status"`
	IsReplay       bool    `json:"is_replay"`
	ResponseStatus *int64  `json:"response_status"`
	ResponseBody   *string `json:"response_body"`
	LatencyMs      *int64  `json:"latency_ms"`
	Error          *string `json:"error"`
	ScheduledAt    string  `json:"scheduled_at"`
	DeliveredAt    *string `json:"delivered_at"`
	NextRetryAt    *string `json:"next_retry_at"`
}

type Event struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	Type       string `json:"type"`
	APIVersion string `json:"api_version"`
	Livemode   bool   `json:"livemode"`
	Created    int64  `json:"created"`
	CreatedAt  string `json:"created_at"`
	Data       struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
	Deliveries   []WebhookDelivery `json:"deliveries,omitempty"`
	LastResponse *Response         `json:"-"`
}

// Payment decodes data.object as a Payment; ok is false for other event families.
func (e *Event) Payment() (*Payment, bool) {
	if !hasPrefix(e.Type, "payment.") {
		return nil, false
	}
	var p Payment
	return &p, json.Unmarshal(e.Data.Object, &p) == nil
}

func (e *Event) Refund() (*Refund, bool) {
	if !hasPrefix(e.Type, "refund.") {
		return nil, false
	}
	var r Refund
	return &r, json.Unmarshal(e.Data.Object, &r) == nil
}

func (e *Event) Transfer() (*Transfer, bool) {
	if !hasPrefix(e.Type, "transfer.") {
		return nil, false
	}
	var t Transfer
	return &t, json.Unmarshal(e.Data.Object, &t) == nil
}

func (e *Event) ControlNumber() (*ControlNumber, bool) {
	if !hasPrefix(e.Type, "control_number.") {
		return nil, false
	}
	var c ControlNumber
	return &c, json.Unmarshal(e.Data.Object, &c) == nil
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }

// EventTypes lists every documented webhook event type.
var EventTypes = []string{
	"payment.pending", "payment.processing", "payment.requires_action", "payment.succeeded", "payment.failed", "payment.expired", "payment.cancelled",
	"refund.pending", "refund.succeeded", "refund.failed",
	"payout.processing", "payout.paid", "payout.failed",
	"transfer.sent", "transfer.received", "transfer.in_review", "transfer.rejected", "transfer.reversed",
	"control_number.partially_paid", "control_number.paid", "control_number.expired", "control_number.cancelled",
}

// Helpers for optional fields in params.
func String(s string) *string { return &s }
func Int64(n int64) *int64    { return &n }
func Bool(b bool) *bool       { return &b }
