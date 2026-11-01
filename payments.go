package paysafe

import (
	"context"
	"time"
)

const (
	MinPaymentAmount = 1000
	MaxPaymentAmount = 10_000_000
)

var PaymentChannels = []string{"mpesa", "tigopesa", "airtel", "halopesa"}
var PaymentStatuses = []string{"created", "pending", "processing", "requires_action", "succeeded", "failed", "cancelled", "expired", "partially_refunded", "refunded"}

type Payments struct{ c *Client }

type CreateCustomerParams struct {
	Phone string `json:"phone"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type CreatePaymentParams struct {
	Amount      int64                `json:"amount"`
	Currency    string               `json:"currency,omitempty"`
	Customer    CreateCustomerParams `json:"customer"`
	Reference   string               `json:"reference,omitempty"`
	Description string               `json:"description,omitempty"`
	Metadata    map[string]any       `json:"metadata,omitempty"`
	ExpiresIn   int                  `json:"expires_in,omitempty"`
	Channel     string               `json:"channel,omitempty"`
}

type ListPaymentsParams struct {
	Limit         int
	Cursor        string
	Status        string
	Reference     string
	CustomerPhone string
	CreatedGte    time.Time
	CreatedLte    time.Time
}

type CancelPaymentParams struct {
	Reason string `json:"reason,omitempty"`
}

func (p *CreatePaymentParams) validate() error {
	if err := validateAmount(p.Amount, MinPaymentAmount, MaxPaymentAmount, "amount"); err != nil {
		return err
	}
	if err := validatePhone(p.Customer.Phone, "customer.phone"); err != nil {
		return err
	}
	if err := validateOptString(p.Customer.Name, "customer.name", 1, 100); err != nil {
		return err
	}
	if err := validateOptString(p.Customer.Email, "customer.email", 3, 255); err != nil {
		return err
	}
	if err := validateEnum(p.Currency, "currency", []string{"TZS"}); err != nil {
		return err
	}
	if err := validateEnum(p.Channel, "channel", PaymentChannels); err != nil {
		return err
	}
	if err := validateOptString(p.Reference, "reference", 1, 120); err != nil {
		return err
	}
	if err := validateOptString(p.Description, "description", 1, 255); err != nil {
		return err
	}
	if err := validateMetadata(p.Metadata, "metadata"); err != nil {
		return err
	}
	if p.ExpiresIn != 0 {
		if err := validateAmount(int64(p.ExpiresIn), 60, 900, "expires_in"); err != nil {
			return err
		}
	}
	return nil
}

// Create starts a mobile-money charge (202 Accepted). An Idempotency-Key is generated when opts has none;
// read it from payment.LastResponse.IdempotencyKey and persist it before retrying yourself.
func (r *Payments) Create(ctx context.Context, params CreatePaymentParams, opts *RequestOptions) (*Payment, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "POST", apiPath("payments"), nil, params, withKey(opts))
	if err != nil {
		return nil, err
	}
	var out Payment
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *Payments) List(ctx context.Context, params ListPaymentsParams, opts *RequestOptions) (*Page[Payment], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	if err := validateEnum(params.Status, "status", PaymentStatuses); err != nil {
		return nil, err
	}
	setIf(q, "status", params.Status)
	setIf(q, "reference", params.Reference)
	setIf(q, "customer_phone", params.CustomerPhone)
	setIf(q, "created_gte", formatTime(params.CreatedGte))
	setIf(q, "created_lte", formatTime(params.CreatedLte))
	resp, err := r.c.do(ctx, "GET", apiPath("payments"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[Payment](resp, func(ctx context.Context, cursor string) (*Page[Payment], error) {
		next := params
		next.Cursor = cursor
		return r.List(ctx, next, opts)
	})
}

func (r *Payments) Get(ctx context.Context, id string, opts *RequestOptions) (*Payment, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("payments", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out Payment
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

// Cancel works for payments in created or pending; later states return invalid_state. The API requires
// an Idempotency-Key here, so one is generated when opts has none.
func (r *Payments) Cancel(ctx context.Context, id string, params *CancelPaymentParams, opts *RequestOptions) (*Payment, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	if params == nil {
		params = &CancelPaymentParams{}
	}
	if err := validateOptString(params.Reason, "reason", 0, 255); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "POST", apiPath("payments", id, "cancel"), nil, params, withKey(opts))
	if err != nil {
		return nil, err
	}
	var out Payment
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

// WaitForTerminal polls Get until the payment is terminal (default every 2 s, up to 2 min).
func (r *Payments) WaitForTerminal(ctx context.Context, id string, opts *WaitOptions) (*Payment, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	return poll(ctx, r.c, func(ctx context.Context) (*Payment, error) { return r.Get(ctx, id, nil) },
		func(p *Payment) string { return p.Status }, func(p *Payment) bool { return p.IsTerminal() },
		WaitOptions{Interval: 2 * time.Second, Timeout: 120 * time.Second}, opts, "payment "+id)
}
