package paysafe

import (
	"context"
	"math"
	"time"
)

var (
	RefundReasons  = []string{"requested_by_customer", "duplicate", "fraudulent", "other"}
	RefundStatuses = []string{"pending", "processing", "succeeded", "failed", "rejected"}
)

type Refunds struct{ c *Client }

type CreateRefundParams struct {
	Payment  string         `json:"payment"`
	Amount   int64          `json:"amount,omitempty"` // 0 refunds the remaining balance
	Reason   string         `json:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ListRefundsParams struct {
	Limit   int
	Cursor  string
	Payment string
	Status  string
}

// Create refunds a succeeded payment (202 Accepted). An Idempotency-Key is generated when opts has none.
func (r *Refunds) Create(ctx context.Context, params CreateRefundParams, opts *RequestOptions) (*Refund, error) {
	if err := validateID(params.Payment, "payment"); err != nil {
		return nil, err
	}
	if params.Amount != 0 {
		if err := validateAmount(params.Amount, 1, math.MaxInt64, "amount"); err != nil {
			return nil, err
		}
	}
	if err := validateEnum(params.Reason, "reason", RefundReasons); err != nil {
		return nil, err
	}
	if err := validateMetadata(params.Metadata, "metadata"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "POST", apiPath("refunds"), nil, params, withKey(opts))
	if err != nil {
		return nil, err
	}
	var out Refund
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *Refunds) List(ctx context.Context, params ListRefundsParams, opts *RequestOptions) (*Page[Refund], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	if err := validateEnum(params.Status, "status", RefundStatuses); err != nil {
		return nil, err
	}
	if params.Payment != "" {
		if err := validateID(params.Payment, "payment"); err != nil {
			return nil, err
		}
	}
	setIf(q, "payment", params.Payment)
	setIf(q, "status", params.Status)
	resp, err := r.c.do(ctx, "GET", apiPath("refunds"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[Refund](resp, func(ctx context.Context, cursor string) (*Page[Refund], error) {
		next := params
		next.Cursor = cursor
		return r.List(ctx, next, opts)
	})
}

func (r *Refunds) Get(ctx context.Context, id string, opts *RequestOptions) (*Refund, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("refunds", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out Refund
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *Refunds) WaitForTerminal(ctx context.Context, id string, opts *WaitOptions) (*Refund, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	return poll(ctx, r.c, func(ctx context.Context) (*Refund, error) { return r.Get(ctx, id, nil) },
		func(x *Refund) string { return x.Status }, func(x *Refund) bool { return x.IsTerminal() },
		WaitOptions{Interval: time.Second, Timeout: 60 * time.Second}, opts, "refund "+id)
}
