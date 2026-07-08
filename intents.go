package qint

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// CreateIntentParams are the inputs to CreateIntent. Amount and Currency are
// required; the remaining fields are optional.
type CreateIntentParams struct {
	// Amount is the fiat amount to charge as a decimal, e.g. 19.90.
	Amount float64 `json:"amount"`
	// Currency is the settlement currency (CurrencyCHF, CurrencyEUR or
	// CurrencyUSD).
	Currency Currency `json:"currency"`
	// Title is an optional human-readable description shown at checkout.
	Title string `json:"title,omitempty"`
	// IdempotencyKey, when set, makes CreateIntent safe to retry: the API
	// replays the original intent (HTTP 200) instead of creating a duplicate.
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	// ReturnURL is an optional https URL (max 500 chars) the buyer is sent
	// back to after checkout.
	ReturnURL string `json:"returnUrl,omitempty"`
}

// CreateIntent creates a new payment intent and returns it. Send the buyer to
// the returned Intent.CheckoutURL to complete payment. Requires an API key
// with the Write scope.
func (c *Client) CreateIntent(ctx context.Context, params CreateIntentParams) (*Intent, error) {
	var intent Intent
	if err := c.do(ctx, http.MethodPost, "/intents", nil, params, &intent); err != nil {
		return nil, err
	}
	return &intent, nil
}

// GetIntent fetches a single payment intent by its id (pi_...). Requires the
// Read scope.
func (c *Client) GetIntent(ctx context.Context, id string) (*Intent, error) {
	var intent Intent
	path := "/intents/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &intent); err != nil {
		return nil, err
	}
	return &intent, nil
}

// ListIntentsParams are the optional filters and pagination for ListIntents.
// A zero-valued field is omitted from the request, letting the API apply its
// defaults.
type ListIntentsParams struct {
	// Status, when set, filters to a single status.
	Status IntentStatus
	// Page is the 1-based page number (API default 1).
	Page int
	// PageSize is the page size (API default 20, max 100).
	PageSize int
}

// ListIntents returns one page of intents, most recent first. Requires the
// Read scope.
func (c *Client) ListIntents(ctx context.Context, params ListIntentsParams) (*IntentList, error) {
	query := url.Values{}
	if params.Status != "" {
		query.Set("status", string(params.Status))
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.PageSize > 0 {
		query.Set("pageSize", strconv.Itoa(params.PageSize))
	}

	var list IntentList
	if err := c.do(ctx, http.MethodGet, "/intents", query, nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}
