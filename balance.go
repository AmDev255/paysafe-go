package paysafe

import "context"

type Balance struct{ c *Client }

type ListParams struct {
	Limit  int
	Cursor string
}

func (r *Balance) Get(ctx context.Context, opts *RequestOptions) (*BalanceObject, error) {
	resp, err := r.c.do(ctx, "GET", apiPath("balance"), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out BalanceObject
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

// Transactions lists ledger entries, newest first.
func (r *Balance) Transactions(ctx context.Context, params ListParams, opts *RequestOptions) (*Page[Transaction], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("transactions"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[Transaction](resp, func(ctx context.Context, cursor string) (*Page[Transaction], error) {
		next := params
		next.Cursor = cursor
		return r.Transactions(ctx, next, opts)
	})
}
