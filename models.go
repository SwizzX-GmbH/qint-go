package qint

import (
	"encoding/json"
	"time"
)

// Currency is a fiat settlement currency supported by Qint.
type Currency string

// Supported settlement currencies.
const (
	CurrencyCHF Currency = "CHF"
	CurrencyEUR Currency = "EUR"
	CurrencyUSD Currency = "USD"
)

// IntentStatus is the lifecycle status of a payment intent. The API always
// returns lowercase values.
type IntentStatus string

// The seven payment-intent statuses.
const (
	StatusInitiated IntentStatus = "initiated"
	StatusPending   IntentStatus = "pending"
	StatusConfirmed IntentStatus = "confirmed"
	StatusSettled   IntentStatus = "settled"
	StatusFailed    IntentStatus = "failed"
	StatusExpired   IntentStatus = "expired"
	StatusCancelled IntentStatus = "cancelled"
)

// Intent is a Qint payment intent.
//
// Amount is decoded as json.Number to preserve the exact decimal value the
// API returned (call Amount.String() or Amount.Float64()). CryptoAmount is an
// 8-decimal-place string. Optional timestamps are pointers that are nil until
// the intent reaches the corresponding state.
type Intent struct {
	ID             string       `json:"id"`
	Status         IntentStatus `json:"status"`
	Amount         json.Number  `json:"amount"`
	Currency       Currency     `json:"currency"`
	Title          string       `json:"title,omitempty"`
	AssetSymbol    string       `json:"assetSymbol,omitempty"`
	CryptoAmount   string       `json:"cryptoAmount,omitempty"`
	DepositAddress string       `json:"depositAddress,omitempty"`
	CheckoutURL    string       `json:"checkoutUrl"`
	ReturnURL      string       `json:"returnUrl,omitempty"`
	CreatedAt      time.Time    `json:"createdAt"`
	ExpiresAt      time.Time    `json:"expiresAt"`
	ConfirmedAt    *time.Time   `json:"confirmedAt,omitempty"`
	SettledAt      *time.Time   `json:"settledAt,omitempty"`
}

// IntentList is one page of intents returned by ListIntents.
type IntentList struct {
	Items    []Intent `json:"items"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}
