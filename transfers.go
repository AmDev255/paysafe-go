package paysafe

import (
	"context"
	"math"
	"time"
)

type Transfers struct{ c *Client }

type CreateTransferParams struct {
	To       string `json:"to"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency,omitempty"`
	Note     string `json:"note,omitempty"`
}

// Create moves money from this project's available balance to another PaySafe project (201).
func (r *Transfers) Create(ctx context.Context, params CreateTransferParams, opts *RequestOptions) (*Transfer, error) {
	if err := validateString(&params.To, "to", 3, 31); err != nil {
		return nil, err
	}
	if err := validateAmount(params.Amount, 1, math.MaxInt64, "amount"); err != nil {
		return nil, err
	}
	if err := validateEnum(params.Currency, "currency", []string{"TZS"}); err != nil {
		return nil, err
	}
	if err := validateOptString(params.Note, "note", 0, 140); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "POST", apiPath("transfers"), nil, params, withKey(opts))
	if err != nil {
		return nil, err
	}
	var out Transfer
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *Transfers) List(ctx context.Context, params ListParams, opts *RequestOptions) (*Page[Transfer], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("transfers"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[Transfer](resp, func(ctx context.Context, cursor string) (*Page[Transfer], error) {
		next := params
		next.Cursor = cursor
		return r.List(ctx, next, opts)
	})
}

func (r *Transfers) Get(ctx context.Context, id string, opts *RequestOptions) (*Transfer, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("transfers", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out Transfer
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *Transfers) WaitForTerminal(ctx context.Context, id string, opts *WaitOptions) (*Transfer, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	return poll(ctx, r.c, func(ctx context.Context) (*Transfer, error) { return r.Get(ctx, id, nil) },
		func(x *Transfer) string { return x.Status }, func(x *Transfer) bool { return x.IsTerminal() },
		WaitOptions{Interval: 2 * time.Second, Timeout: 120 * time.Second}, opts, "transfer "+id)
}
