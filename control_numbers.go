package paysafe

import "context"

const MaxControlNumberAmount = 1_000_000_000_000

var (
	ControlNumberStatuses = []string{"active", "paid", "expired", "cancelled"}
	ControlNumberStates   = []string{"unpaid", "partial", "paid", "overdue"}
	ControlNumberModes    = []string{"full", "instalment"}
)

type ControlNumbers struct{ c *Client }

type Instalment struct {
	Amount int64  `json:"amount"`
	DueAt  string `json:"due_at"`
}

type CreateControlNumberParams struct {
	PayerName      string         `json:"payer_name"`
	PayerPhone     string         `json:"payer_phone,omitempty"`
	PayerEmail     string         `json:"payer_email,omitempty"`
	PayerReference string         `json:"payer_reference,omitempty"`
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	Reference      string         `json:"reference,omitempty"`
	PaymentType    string         `json:"payment_type,omitempty"`
	Mode           string         `json:"mode,omitempty"`
	Amount         int64          `json:"amount"`
	Currency       string         `json:"currency,omitempty"`
	ExpiresAt      string         `json:"expires_at,omitempty"` // RFC 3339 or YYYY-MM-DD
	Notes          string         `json:"notes,omitempty"`
	Instalments    []Instalment   `json:"instalments,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Image          string         `json:"image,omitempty"`
}

type ListControlNumbersParams struct {
	Limit    int
	Cursor   string
	Status   string
	State    string
	Batch    string
	Customer string
	Q        string
}

func (p *CreateControlNumberParams) validate() error {
	for _, f := range []struct {
		v        string
		name     string
		min, max int
	}{{p.PayerName, "payer_name", 1, 120}, {p.Title, "title", 1, 150}} {
		v := f.v
		if err := validateString(&v, f.name, f.min, f.max); err != nil {
			return err
		}
	}
	for _, f := range []struct {
		v        string
		name     string
		min, max int
	}{{p.PayerPhone, "payer_phone", 0, 20}, {p.PayerEmail, "payer_email", 0, 255}, {p.PayerReference, "payer_reference", 0, 120}, {p.Description, "description", 0, 500}, {p.Reference, "reference", 0, 120}, {p.PaymentType, "payment_type", 0, 60}, {p.Notes, "notes", 0, 1000}, {p.Image, "image", 0, 240}} {
		if err := validateOptString(f.v, f.name, f.min, f.max); err != nil {
			return err
		}
	}
	if err := validateEnum(p.Mode, "mode", ControlNumberModes); err != nil {
		return err
	}
	if err := validateEnum(p.Currency, "currency", []string{"TZS"}); err != nil {
		return err
	}
	if err := validateAmount(p.Amount, 1, MaxControlNumberAmount, "amount"); err != nil {
		return err
	}
	if err := validateMetadata(p.Metadata, "metadata"); err != nil {
		return err
	}
	if err := validateDateString(p.ExpiresAt, "expires_at"); err != nil {
		return err
	}
	for i, in := range p.Instalments {
		if err := validateAmount(in.Amount, 1, MaxControlNumberAmount, "instalments.amount"); err != nil {
			return err
		}
		if in.DueAt == "" {
			return invalid("instalments", "instalments[%d].due_at is required", i)
		}
		if err := validateDateString(in.DueAt, "instalments.due_at"); err != nil {
			return err
		}
	}
	return nil
}

// Create issues a 12-digit pay-by-reference number (201). The number itself is assigned by the server.
func (r *ControlNumbers) Create(ctx context.Context, params CreateControlNumberParams, opts *RequestOptions) (*ControlNumber, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "POST", apiPath("control-numbers"), nil, params, opts)
	if err != nil {
		return nil, err
	}
	var out ControlNumber
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *ControlNumbers) List(ctx context.Context, params ListControlNumbersParams, opts *RequestOptions) (*Page[ControlNumber], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	if err := validateEnum(params.Status, "status", ControlNumberStatuses); err != nil {
		return nil, err
	}
	if err := validateEnum(params.State, "state", ControlNumberStates); err != nil {
		return nil, err
	}
	if err := validateOptString(params.Q, "q", 0, 80); err != nil {
		return nil, err
	}
	setIf(q, "status", params.Status)
	setIf(q, "state", params.State)
	setIf(q, "batch", params.Batch)
	setIf(q, "customer", params.Customer)
	setIf(q, "q", params.Q)
	resp, err := r.c.do(ctx, "GET", apiPath("control-numbers"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[ControlNumber](resp, func(ctx context.Context, cursor string) (*Page[ControlNumber], error) {
		next := params
		next.Cursor = cursor
		return r.List(ctx, next, opts)
	})
}

// Get returns the number with every payment entry recorded against it (Entries).
func (r *ControlNumbers) Get(ctx context.Context, id string, opts *RequestOptions) (*ControlNumber, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("control-numbers", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out ControlNumber
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

// Cancel stops an active number; the reason (3 to 500 characters) is kept on the number.
func (r *ControlNumbers) Cancel(ctx context.Context, id, reason string, opts *RequestOptions) (*ControlNumber, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	if err := validateString(&reason, "reason", 3, 500); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "POST", apiPath("control-numbers", id, "cancel"), nil, map[string]string{"reason": reason}, opts)
	if err != nil {
		return nil, err
	}
	var out ControlNumber
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}
