package paysafe

import "context"

type WebhookEndpoints struct{ c *Client }

type CreateWebhookEndpointParams struct {
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	Events      []string `json:"events,omitempty"` // event types or ["*"]
}

type UpdateWebhookEndpointParams struct {
	URL         string   `json:"url,omitempty"`
	Description *string  `json:"description,omitempty"`
	Events      []string `json:"events,omitempty"`
	Status      string   `json:"status,omitempty"` // ENABLED or DISABLED
}

var eventTypeSet = func() map[string]bool {
	m := map[string]bool{"*": true}
	for _, t := range EventTypes {
		m[t] = true
	}
	return m
}()

func validateEvents(events []string) error {
	if events == nil {
		return nil
	}
	if len(events) == 0 {
		return invalid("events", `events must be a non-empty list of event types or ["*"]`)
	}
	for _, e := range events {
		if !eventTypeSet[e] {
			return invalid("events", "unknown event type %s", e)
		}
	}
	return nil
}

// List returns every endpoint of the project; the API does not paginate this resource.
func (r *WebhookEndpoints) List(ctx context.Context, opts *RequestOptions) ([]WebhookEndpoint, error) {
	resp, err := r.c.do(ctx, "GET", apiPath("webhook-endpoints"), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Data []WebhookEndpoint `json:"data"`
	}
	if err := decode(resp, &raw); err != nil {
		return nil, err
	}
	if raw.Data == nil {
		raw.Data = []WebhookEndpoint{}
	}
	return raw.Data, nil
}

// Create registers an endpoint. The signing secret is only returned here; store it immediately.
func (r *WebhookEndpoints) Create(ctx context.Context, params CreateWebhookEndpointParams, opts *RequestOptions) (*WebhookEndpoint, error) {
	if params.URL == "" {
		return nil, invalid("url", "url is required")
	}
	if err := validateURL(params.URL, "url", 500); err != nil {
		return nil, err
	}
	if err := validateOptString(params.Description, "description", 0, 255); err != nil {
		return nil, err
	}
	if err := validateEvents(params.Events); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "POST", apiPath("webhook-endpoints"), nil, params, opts)
	if err != nil {
		return nil, err
	}
	var out WebhookEndpoint
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *WebhookEndpoints) Get(ctx context.Context, id string, opts *RequestOptions) (*WebhookEndpoint, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("webhook-endpoints", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out WebhookEndpoint
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *WebhookEndpoints) Update(ctx context.Context, id string, params UpdateWebhookEndpointParams, opts *RequestOptions) (*WebhookEndpoint, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	if err := validateURL(params.URL, "url", 500); err != nil {
		return nil, err
	}
	if err := validateString(params.Description, "description", 0, 255); err != nil {
		return nil, err
	}
	if err := validateEnum(params.Status, "status", []string{"ENABLED", "DISABLED"}); err != nil {
		return nil, err
	}
	if err := validateEvents(params.Events); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "PATCH", apiPath("webhook-endpoints", id), nil, params, opts)
	if err != nil {
		return nil, err
	}
	var out WebhookEndpoint
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *WebhookEndpoints) Delete(ctx context.Context, id string, opts *RequestOptions) error {
	if err := validateID(id, "id"); err != nil {
		return err
	}
	_, err := r.c.do(ctx, "DELETE", apiPath("webhook-endpoints", id), nil, nil, opts)
	return err
}
