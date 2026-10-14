package qint

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Machine-readable error codes the API puts in a problem body's "code"
// member (see QintError.Code).
const (
	// ErrCodeAmountOutOfRange: HTTP 400 on create — the amount is outside the
	// per-payment limits (CHF 10 to CHF 100'000 or the equivalent). MinChf,
	// MaxChf and AmountChf carry the numbers.
	ErrCodeAmountOutOfRange = "amount_out_of_range"
	// ErrCodeRateUnavailable: HTTP 503 on create — no exchange rate was
	// available to judge the limits. Retry shortly.
	ErrCodeRateUnavailable = "rate_unavailable"
	// ErrCodeSettlementAssetMismatch: HTTP 400 — a crypto payout was requested
	// in an asset other than the merchant's settlement asset (USDT, or USDC
	// for EU merchants). Returned by the payout endpoints of the dashboard
	// API, not by the merchant API this client wraps; listed for completeness.
	ErrCodeSettlementAssetMismatch = "settlement_asset_mismatch"
)

// QintError is returned for any non-2xx API response. It carries the HTTP
// status code together with the RFC 7807 problem-details fields returned by
// the API. Use errors.As to extract it:
//
//	var qerr *qint.QintError
//	if errors.As(err, &qerr) && qerr.Code == qint.ErrCodeAmountOutOfRange {
//		// qerr.MinChf, qerr.MaxChf, qerr.AmountChf
//	}
type QintError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Type is the problem-details "type" URI, if present.
	Type string
	// Title is the short, human-readable summary of the problem type.
	Title string
	// Detail is the human-readable explanation specific to this occurrence.
	Detail string
	// TraceID is the request correlation id ("traceId") — include it in
	// support requests.
	TraceID string
	// Code is the machine-readable error code, if any — for example
	// ErrCodeAmountOutOfRange or ErrCodeRateUnavailable.
	Code string
	// MinChf, MaxChf and AmountChf are set on amount_out_of_range: the
	// allowed range and the requested amount, in CHF. nil otherwise.
	MinChf    *json.Number
	MaxChf    *json.Number
	AmountChf *json.Number
	// Extensions holds every top-level problem member beyond type, title,
	// status, detail and instance (the API serialises extensions flat), with
	// numbers decoded as json.Number. nil when the body was not a JSON object.
	Extensions map[string]any
	// Body is the raw response body, retained for debugging.
	Body string
}

// problemDetails mirrors the RFC 7807 application/problem+json body.
type problemDetails struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Detail    string       `json:"detail"`
	TraceID   string       `json:"traceId"`
	Code      string       `json:"code"`
	MinChf    *json.Number `json:"minChf"`
	MaxChf    *json.Number `json:"maxChf"`
	AmountChf *json.Number `json:"amountChf"`
}

var standardProblemMembers = map[string]bool{
	"type": true, "title": true, "status": true, "detail": true, "instance": true,
}

func newQintError(status int, body []byte) *QintError {
	e := &QintError{StatusCode: status, Body: string(body)}
	var p problemDetails
	if err := json.Unmarshal(body, &p); err == nil {
		e.Type = p.Type
		e.Title = p.Title
		e.Detail = p.Detail
		e.TraceID = p.TraceID
		e.Code = p.Code
		e.MinChf = p.MinChf
		e.MaxChf = p.MaxChf
		e.AmountChf = p.AmountChf
	}
	var all map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&all); err == nil && all != nil {
		e.Extensions = map[string]any{}
		for k, v := range all {
			if !standardProblemMembers[k] {
				e.Extensions[k] = v
			}
		}
	}
	return e
}

// Error implements the error interface.
func (e *QintError) Error() string {
	switch {
	case e.Detail != "":
		return fmt.Sprintf("qint: HTTP %d: %s", e.StatusCode, e.Detail)
	case e.Title != "":
		return fmt.Sprintf("qint: HTTP %d: %s", e.StatusCode, e.Title)
	default:
		return fmt.Sprintf("qint: unexpected HTTP status %d", e.StatusCode)
	}
}
