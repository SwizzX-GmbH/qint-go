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

// SettlementAsset is the stablecoin a merchant's balance is held in.
// Merchants registered in the European Union settle in USDC (USDT is not
// authorised under MiCA); all other merchants settle in USDT. Unknown future
// values still decode (the type is a plain string).
type SettlementAsset string

// Known settlement assets.
const (
	SettlementAssetUSDT SettlementAsset = "USDT"
	SettlementAssetUSDC SettlementAsset = "USDC"
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
//     the payment settles, in Currency (null until settled).
//   - SettledAmount: what the execution partner booked as realised.
//   - ReceivedAmount: the fiat amount actually paid when the payment settled
//     short within the merchant's accepted underpayment.
//   - UnderpaidBps: how short the payment was, in basis points of the invoice.
//   - CreditUsdt / CreditRate: the amount credited to the merchant balance,
//     in CreditAsset (6 decimal places), and the rate it was booked at (units
//     of Currency per 1 CreditAsset). The name CreditUsdt predates USDC: for
//     a merchant settling in USDC the credit is in USDC.
//   - CreditAsset: the stablecoin the credit was booked in (USDT, or USDC for
//     merchants registered in the EU); nil until credited.
//   - ReversedAt: set when the payment was reversed after settlement; the
//     status stays settled, but treat the payment as not paid.
//
// Reversed mirrors ReversedAt != nil; Underpaid is true when the payment
// settled (or was detected) short of the quoted amount. Both are false when
// the response omits them.
type Intent struct {
	ID             string           `json:"id"`
	Status         IntentStatus     `json:"status"`
	Amount         json.Number      `json:"amount"`
	Currency       Currency         `json:"currency"`
	Title          string           `json:"title,omitempty"`
	AssetSymbol    string           `json:"assetSymbol,omitempty"`
	CryptoAmount   string           `json:"cryptoAmount,omitempty"`
	DepositAddress string           `json:"depositAddress,omitempty"`
	CheckoutURL    string           `json:"checkoutUrl"`
	ReturnURL      string           `json:"returnUrl,omitempty"`
	CreatedAt      time.Time        `json:"createdAt"`
	ExpiresAt      time.Time        `json:"expiresAt"`
	ConfirmedAt    *time.Time       `json:"confirmedAt,omitempty"`
	SettledAt      *time.Time       `json:"settledAt,omitempty"`
	ReversedAt     *time.Time       `json:"reversedAt,omitempty"`
	FeeAmount      *json.Number     `json:"feeAmount,omitempty"`
	NetAmount      *json.Number     `json:"netAmount,omitempty"`
	SettledAmount  *json.Number     `json:"settledAmount,omitempty"`
	ReceivedAmount *json.Number     `json:"receivedAmount,omitempty"`
	UnderpaidBps   *int             `json:"underpaidBps,omitempty"`
	CreditUsdt     *json.Number     `json:"creditUsdt,omitempty"`
	CreditRate     *json.Number     `json:"creditRate,omitempty"`
	CreditAsset    *SettlementAsset `json:"creditAsset,omitempty"`
	Reversed       bool             `json:"reversed,omitempty"`
	Underpaid      bool             `json:"underpaid,omitempty"`
}

// IntentList is one page of intents returned by ListIntents.
type IntentList struct {
	Items    []Intent `json:"items"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

// CurrencyAvailability says whether one fiat currency is offered for new
// payments right now. Code is a plain string so currencies added later still
// decode; compare it with the Currency constants via Currency(c.Code).
type CurrencyAvailability struct {
	Code    string `json:"code"`
	Enabled bool   `json:"enabled"`
}

// CurrencyList is the response of ListCurrencies: every fiat currency Qint
// knows, in display order.
type CurrencyList struct {
	Currencies []CurrencyAvailability `json:"currencies"`
}

// EnabledCurrencies returns the currencies currently offered for new
// payments, in display order.
func (l *CurrencyList) EnabledCurrencies() []Currency {
	if l == nil {
		return nil
	}
	out := make([]Currency, 0, len(l.Currencies))
	for _, c := range l.Currencies {
		if c.Enabled {
			out = append(out, Currency(c.Code))
		}
	}
	return out
}

// IsEnabled reports whether the given currency is listed and enabled.
func (l *CurrencyList) IsEnabled(currency Currency) bool {
	if l == nil {
		return false
	}
	for _, c := range l.Currencies {
		if Currency(c.Code) == currency {
			return c.Enabled
		}
	}
	return false
}
