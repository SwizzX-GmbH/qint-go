package qint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// Webhook event types. Handlers should acknowledge (2xx) and ignore types they
// do not recognise, so future event types never break them.
const (
	EventTypePaymentStatus = "payment.status"
	EventTypePing          = "ping"
)

// PingEvent is the payload of the test event sent from the dashboard ("Send
// test event"). It is signed like any other delivery; verify it, return 2xx
// and ignore it.
type PingEvent struct {
	Type       string    `json:"type"`
	EndpointID string    `json:"endpointId"`
	SentAt     time.Time `json:"sentAt"`
}

// PaymentStatusEvent is the JSON payload delivered to a merchant webhook
// endpoint. Its Type is always "payment.status".
//
// Amount is decoded as json.Number to preserve the exact decimal value.
// Underpaid is a pointer so a missing field (nil) is distinguishable from an
// explicit false. ReceivedAmount (the fiat amount actually paid, in Currency)
// and UnderpaidBps (how short, in basis points of the invoice) are set when the
// payment settled short within the merchant's accepted underpayment, nil
// otherwise. ExpectedCryptoAmount / ReceivedCryptoAmount are 8-decimal-place
// strings. Reversed / ReversedAt flag a payment reversed after settlement.
type PaymentStatusEvent struct {
	Type                 string       `json:"type"`
	IntentID             string       `json:"intentId"`
	Status               IntentStatus `json:"status"`
	Amount               json.Number  `json:"amount"`
	Currency             Currency     `json:"currency"`
	AssetSymbol          string       `json:"assetSymbol"`
	InvoiceID            string       `json:"invoiceId,omitempty"`
	PaymentLinkID        string       `json:"paymentLinkId,omitempty"`
	OccurredAt           time.Time    `json:"occurredAt"`
	Underpaid            *bool        `json:"underpaid,omitempty"`
	ExpectedCryptoAmount string       `json:"expectedCryptoAmount,omitempty"`
	ReceivedCryptoAmount string       `json:"receivedCryptoAmount,omitempty"`
	ReceivedAmount       *json.Number `json:"receivedAmount,omitempty"`
	UnderpaidBps         *int         `json:"underpaidBps,omitempty"`
	Reversed             *bool        `json:"reversed,omitempty"`
	ReversedAt           *time.Time   `json:"reversedAt,omitempty"`
}

// signaturePrefix is the scheme label on the X-Qint-Signature header value.
const signaturePrefix = "sha256="

// VerifyWebhookSignature reports whether signatureHeader is a valid
// HMAC-SHA256 signature of rawBody produced with the endpoint's signing
// secret (whsec_...). signatureHeader is the raw X-Qint-Signature header
// value, i.e. "sha256=<hex>"; a bare "<hex>" is also accepted. The comparison
// is constant-time (crypto/hmac.Equal).
//
// Always verify against the exact raw request body bytes — before any JSON
// decoding — and reject the delivery if this returns false. Use the
// X-Qint-Event-Id header to deduplicate re-deliveries of the same event.
func VerifyWebhookSignature(rawBody []byte, signatureHeader, secret string) bool {
	if secret == "" || signatureHeader == "" {
		return false
	}
	provided := strings.TrimSpace(strings.TrimPrefix(signatureHeader, signaturePrefix))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(provided))
}

// ParseEvent unmarshals a webhook request body into a PaymentStatusEvent.
// Verify the signature with VerifyWebhookSignature first. Any event type
// decodes without error — check Type (EventTypePaymentStatus, EventTypePing,
// or something newer) before acting on the payment fields; use
// ParsePingEvent for the ping payload.
func ParseEvent(rawBody []byte) (*PaymentStatusEvent, error) {
	var event PaymentStatusEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParsePingEvent unmarshals a "ping" webhook body into a PingEvent.
func ParsePingEvent(rawBody []byte) (*PingEvent, error) {
	var event PingEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
