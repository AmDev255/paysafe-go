package paysafe

import "context"

type Customers struct{ c *Client }

type ListCustomersParams struct {
	Limit  int
	Cursor string
	Search string
}

func (r *Customers) List(ctx context.Context, params ListCustomersParams, opts *RequestOptions) (*Page[Customer], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	if err := validateOptString(params.Search, "search", 0, 120); err != nil {
		return nil, err
	}
	setIf(q, "search", params.Search)
	resp, err := r.c.do(ctx, "GET", apiPath("customers"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[Customer](resp, func(ctx context.Context, cursor string) (*Page[Customer], error) {
		next := params
		next.Cursor = cursor
		return r.List(ctx, next, opts)
	})
}

func (r *Customers) Get(ctx context.Context, id string, opts *RequestOptions) (*Customer, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("customers", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out Customer
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}
