package paysafe

import (
	"context"
	"time"
)

var PaymentLinkStatuses = []string{"active", "paused", "archived"}

type PaymentLinks struct{ c *Client }

type CreatePaymentLinkParams struct {
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	Amount         int64     `json:"amount,omitempty"`
	MinAmount      int64     `json:"min_amount,omitempty"`
	MaxAmount      int64     `json:"max_amount,omitempty"`
	Currency       string    `json:"currency,omitempty"`
	CollectName    bool      `json:"collect_name,omitempty"`
	CollectEmail   bool      `json:"collect_email,omitempty"`
	MaxPayments    int64     `json:"max_payments,omitempty"`
	ExpiresAt      time.Time `json:"-"`
	SuccessMessage string    `json:"success_message,omitempty"`
	RedirectURL    string    `json:"redirect_url,omitempty"`
	Logo           string    `json:"logo,omitempty"`
	AccentColor    string    `json:"accent_color,omitempty"`
}

// UpdatePaymentLinkParams uses pointers so a field can be left untouched (nil) or cleared (pointer to empty value).
type UpdatePaymentLinkParams struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	MaxPayments    *int64  `json:"max_payments,omitempty"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
	Status         string  `json:"status,omitempty"`
	SuccessMessage *string `json:"success_message,omitempty"`
	RedirectURL    *string `json:"redirect_url,omitempty"`
	CollectName    *bool   `json:"collect_name,omitempty"`
	CollectEmail   *bool   `json:"collect_email,omitempty"`
	Logo           *string `json:"logo,omitempty"`
	AccentColor    *string `json:"accent_color,omitempty"`
}

type ListPaymentLinksParams struct {
	Limit  int
	Cursor string
	Status string
}

func (p *CreatePaymentLinkParams) validate() error {
	if err := validateString(&p.Title, "title", 2, 120); err != nil {
		return err
	}
	for _, f := range []struct {
		v        string
		name     string
		min, max int
	}{{p.Description, "description", 1, 1000}, {p.SuccessMessage, "success_message", 1, 255}, {p.Logo, "logo", 0, 240}, {p.AccentColor, "accent_color", 0, 32}} {
		if err := validateOptString(f.v, f.name, f.min, f.max); err != nil {
			return err
		}
	}
	for _, f := range []struct {
		v    int64
		name string
	}{{p.Amount, "amount"}, {p.MinAmount, "min_amount"}, {p.MaxAmount, "max_amount"}} {
		if f.v != 0 {
			if err := validateAmount(f.v, MinPaymentAmount, MaxPaymentAmount, f.name); err != nil {
				return err
			}
		}
	}
	if p.MaxPayments != 0 {
		if err := validateAmount(p.MaxPayments, 1, 1_000_000, "max_payments"); err != nil {
			return err
		}
	}
	if err := validateEnum(p.Currency, "currency", []string{"TZS"}); err != nil {
		return err
	}
	return validateURL(p.RedirectURL, "redirect_url", 500)
}

func (p *UpdatePaymentLinkParams) validate() error {
	if err := validateString(p.Title, "title", 2, 120); err != nil {
		return err
	}
	if err := validateString(p.Description, "description", 1, 1000); err != nil {
		return err
	}
	if err := validateString(p.SuccessMessage, "success_message", 1, 255); err != nil {
		return err
	}
	if err := validateString(p.Logo, "logo", 0, 240); err != nil {
		return err
	}
	if err := validateString(p.AccentColor, "accent_color", 0, 32); err != nil {
		return err
	}
	if p.MaxPayments != nil {
		if err := validateAmount(*p.MaxPayments, 1, 1_000_000, "max_payments"); err != nil {
			return err
		}
	}
	if p.ExpiresAt != nil {
		if err := validateDateString(*p.ExpiresAt, "expires_at"); err != nil {
			return err
		}
	}
	if p.RedirectURL != nil {
		if err := validateURL(*p.RedirectURL, "redirect_url", 500); err != nil {
			return err
		}
	}
	return validateEnum(p.Status, "status", PaymentLinkStatuses)
}

type createPaymentLinkBody struct {
	CreatePaymentLinkParams
	ExpiresAt string `json:"expires_at,omitempty"`
}

func (r *PaymentLinks) Create(ctx context.Context, params CreatePaymentLinkParams, opts *RequestOptions) (*PaymentLink, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	body := createPaymentLinkBody{CreatePaymentLinkParams: params, ExpiresAt: formatTime(params.ExpiresAt)}
	resp, err := r.c.do(ctx, "POST", apiPath("payment-links"), nil, body, opts)
	if err != nil {
		return nil, err
	}
	var out PaymentLink
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *PaymentLinks) List(ctx context.Context, params ListPaymentLinksParams, opts *RequestOptions) (*Page[PaymentLink], error) {
	q, err := listQuery(params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	if err := validateEnum(params.Status, "status", PaymentLinkStatuses); err != nil {
		return nil, err
	}
	setIf(q, "status", params.Status)
	resp, err := r.c.do(ctx, "GET", apiPath("payment-links"), q, nil, opts)
	if err != nil {
		return nil, err
	}
	return pageFromResponse[PaymentLink](resp, func(ctx context.Context, cursor string) (*Page[PaymentLink], error) {
		next := params
		next.Cursor = cursor
		return r.List(ctx, next, opts)
	})
}

func (r *PaymentLinks) Get(ctx context.Context, id string, opts *RequestOptions) (*PaymentLink, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "GET", apiPath("payment-links", id), nil, nil, opts)
	if err != nil {
		return nil, err
	}
	var out PaymentLink
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

// Update pauses, resumes, archives or edits a link. The amount cannot be changed; archiving is permanent.
func (r *PaymentLinks) Update(ctx context.Context, id string, params UpdatePaymentLinkParams, opts *RequestOptions) (*PaymentLink, error) {
	if err := validateID(id, "id"); err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	resp, err := r.c.do(ctx, "PATCH", apiPath("payment-links", id), nil, params, opts)
	if err != nil {
		return nil, err
	}
	var out PaymentLink
	if err := decode(resp, &out); err != nil {
		return nil, err
	}
	out.LastResponse = resp
	return &out, nil
}

func (r *PaymentLinks) Pause(ctx context.Context, id string, opts *RequestOptions) (*PaymentLink, error) {
	return r.Update(ctx, id, UpdatePaymentLinkParams{Status: "paused"}, opts)
}

func (r *PaymentLinks) Resume(ctx context.Context, id string, opts *RequestOptions) (*PaymentLink, error) {
	return r.Update(ctx, id, UpdatePaymentLinkParams{Status: "active"}, opts)
}

func (r *PaymentLinks) Archive(ctx context.Context, id string, opts *RequestOptions) (*PaymentLink, error) {
	return r.Update(ctx, id, UpdatePaymentLinkParams{Status: "archived"}, opts)
}
