package paysafe

import (
	"context"
	"encoding/json"
)

// Page is one page of a cursor-paginated list.
type Page[T any] struct {
	Data         []T     `json:"data"`
	HasMore      bool    `json:"has_more"`
	NextCursor   *string `json:"next_cursor"`
	LastResponse *Response
	next         func(ctx context.Context, cursor string) (*Page[T], error)
}

func pageFromResponse[T any](resp *Response, next func(ctx context.Context, cursor string) (*Page[T], error)) (*Page[T], error) {
	var raw struct {
		Data       []T     `json:"data"`
		HasMore    bool    `json:"has_more"`
		NextCursor *string `json:"next_cursor"`
	}
	if err := decode(resp, &raw); err != nil {
		return nil, err
	}
	if raw.Data == nil {
		raw.Data = []T{}
	}
	return &Page[T]{Data: raw.Data, HasMore: raw.HasMore, NextCursor: raw.NextCursor, LastResponse: resp, next: next}, nil
}

// Next fetches the following page, or returns nil when this was the last one.
func (p *Page[T]) Next(ctx context.Context) (*Page[T], error) {
	if !p.HasMore || p.NextCursor == nil || p.next == nil {
		return nil, nil
	}
	return p.next(ctx, *p.NextCursor)
}

// ForEach calls fn for every item on this page and all following pages; a non-nil error stops the walk.
func (p *Page[T]) ForEach(ctx context.Context, fn func(T) error) error {
	page := p
	for page != nil {
		for _, item := range page.Data {
			if err := fn(item); err != nil {
				return err
			}
		}
		next, err := page.Next(ctx)
		if err != nil {
			return err
		}
		page = next
	}
	return nil
}

// All collects every item across pages. Prefer ForEach for large lists.
func (p *Page[T]) All(ctx context.Context) ([]T, error) {
	var out []T
	err := p.ForEach(ctx, func(item T) error {
		out = append(out, item)
		return nil
	})
	return out, err
}

func (p *Page[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Object     string  `json:"object"`
		Data       []T     `json:"data"`
		HasMore    bool    `json:"has_more"`
		NextCursor *string `json:"next_cursor"`
	}{"list", p.Data, p.HasMore, p.NextCursor})
}
