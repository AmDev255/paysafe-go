package paysafe

import (
	"context"
	"time"
)

type Events struct{ c *Client }

type ListEventsParams struct {
	Limit      int
	Cursor     string
	Type       string
	From       time.Time
	To         time.Time
	ResourceID string
}

func (r *Events) List(ctx context.Context, params ListEventsParams, opts *RequestOptions) (*Page[Event], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	if err := validateEnum(params.Type, "type", EventTypes); err != nil {
		return nil, err
	}
	if params.ResourceID != "" {
		if err := validateID(params.ResourceID, "resourceId"); err != nil {
			return nil, err
		}
	}
	setIf(q, "type", params.Type)
	setIf(q, "from", formatTime(params.From))
	setIf(q, "to", formatTime(params.To))
	setIf(q, "resourceId", params.ResourceID)
	resp, err := r.c.do(ctx, "GET", apiPath("events"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[Event](resp, func(ctx context.Context, cursor string) (*Page[Event], error) {
		next := params
		next.Cursor = cursor
		return r.List(ctx, next, opts)
	})
}

// Get returns the event plus every delivery attempt to your endpoints (Deliveries).
func (r *Events) Get(ctx context.Context, id string, opts *RequestOptions) (*Event, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("events", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out Event
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}
