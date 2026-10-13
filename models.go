package qint

import (
	"encoding/json"
	"time"
)

// Currency is a fiat settlement currency supported by Qint.
type Currency string

// Supported currencies: CHF, EUR, USD or GBP.
const (
	CurrencyCHF Currency = "CHF"
	CurrencyEUR Currency = "EUR"
	CurrencyUSD Currency = "USD"
	CurrencyGBP Currency = "GBP"
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
//
// The fee, credit, underpayment and reversal fields are optional pointers,
// nil when the response does not include them (or sends null):
//   - FeeAmount / NetAmount: Qint's fee and what the merchant is credited once
//     the payment settles, in Currency.
//   - SettledAmount: what the execution partner booked as realised.
//   - ReceivedAmount: the fiat amount actually paid when the payment settled
//     short within the merchant's accepted underpayment.
//   - UnderpaidBps: how short the payment was, in basis points of the invoice.
//   - CreditUsdt / CreditRate: the USDT credited to the merchant balance and
//     the rate it was booked at (fiat per 1 USDT).
//   - ReversedAt: set when the payment was reversed after settlement; the
//     status stays settled, but treat the payment as not paid.
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
	ReversedAt     *time.Time   `json:"reversedAt,omitempty"`
	FeeAmount      *json.Number `json:"feeAmount,omitempty"`
	NetAmount      *json.Number `json:"netAmount,omitempty"`
	SettledAmount  *json.Number `json:"settledAmount,omitempty"`
	ReceivedAmount *json.Number `json:"receivedAmount,omitempty"`
	UnderpaidBps   *int         `json:"underpaidBps,omitempty"`
	CreditUsdt     *json.Number `json:"creditUsdt,omitempty"`
	CreditRate     *json.Number `json:"creditRate,omitempty"`
}

// IntentList is one page of intents returned by ListIntents.
type IntentList struct {
	Items    []Intent `json:"items"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}
