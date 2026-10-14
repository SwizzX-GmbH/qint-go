# qint-go

Official Go client for the [Qint](https://docs.qint.ch) merchant API — a thin,
typed wrapper for creating payment intents, reading them back, and verifying
webhooks.

- Zero third-party dependencies (standard library only)
- `context.Context` on every request
- Typed models, typed errors, and constant-time webhook verification
- Go 1.21+

## Install

```sh
go get github.com/SwizzX-GmbH/qint-go@v0.3.0
```

> **Released.** `v0.1.0`, `v0.2.0` and `v0.3.0` are tagged and resolvable
> through the public Go module proxy. See [CHANGELOG.md](./CHANGELOG.md) for
> what each version added. Go has no central registry to push to and needs no
> publishing credentials — modules are served from Git and cached by
> `proxy.golang.org`, so tagging *is* releasing. See [PUBLISH.md](./PUBLISH.md)
> for how to cut the next version.

```go
import qint "github.com/SwizzX-GmbH/qint-go"
```

## Quickstart (30 seconds)

```go
package main

import (
	"context"
	"fmt"
	"log"

	qint "github.com/SwizzX-GmbH/qint-go"
)

func main() {
	client := qint.NewClient("qk_live_...")

	// 1. Create a payment intent.
	intent, err := client.CreateIntent(context.Background(), qint.CreateIntentParams{
		Amount:         19.90,
		Currency:       qint.CurrencyCHF,
		Title:          "Order #1024",
		IdempotencyKey: "order-1024", // required — unique per merchant; safe to retry
		ReturnURL:      "https://shop.example/thanks",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Redirect the buyer to the hosted checkout.
	fmt.Println("Send the buyer to:", intent.CheckoutURL)

	// 3. Learn the outcome by polling GetIntent (below) or, preferably, by
	//    receiving a webhook (see "Webhooks").
	latest, err := client.GetIntent(context.Background(), intent.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Status:", latest.Status)
}
```

### Configuration

```go
client := qint.NewClient("qk_live_...",
	qint.WithBaseURL("https://api.qint.ch/api/v1"), // default
	qint.WithTimeout(15*time.Second),                    // default 30s
	// qint.WithHTTPClient(myClient),                    // full control over transport
)
```

The client sends `Authorization: Bearer <apiKey>` on every request.

## API

| Method | Description | Scope |
| --- | --- | --- |
| `CreateIntent(ctx, CreateIntentParams) (*Intent, error)` | Create a payment intent | Write |
| `GetIntent(ctx, id string) (*Intent, error)` | Fetch one intent by id | Read |
| `ListIntents(ctx, ListIntentsParams) (*IntentList, error)` | List intents (paged, newest first) | Read |
| `ListCurrencies(ctx) (*CurrencyList, error)` | Which fiat currencies are offered for new payments right now | none (public) |

```go
type CreateIntentParams struct {
	Amount         float64  // required, decimal, e.g. 19.90
	Currency       Currency // required: CurrencyCHF | CurrencyEUR | CurrencyUSD | CurrencyGBP
	IdempotencyKey string   // required — unique per merchant; replays instead of duplicating on retry
	Title          string   // optional
	ReturnURL      string   // optional https URL, <=500 chars
}

type ListIntentsParams struct {
	Status   IntentStatus // optional filter
	Page     int          // optional (default 1)
	PageSize int          // optional (default 20, max 100)
}
```

`Intent` statuses are the seven lowercase values:
`initiated`, `pending`, `confirmed`, `settled`, `failed`, `expired`, `cancelled`
(exported as `StatusInitiated`, `StatusPending`, ... `StatusCancelled`).

`Intent.Amount` and `PaymentStatusEvent.Amount` are decoded as `json.Number` to
preserve the exact decimal the server sent — use `.String()` or `.Float64()`.

`Intent` also has optional pointer fields that are `nil` when the response does
not include them (or sends `null`): `ReversedAt`, `FeeAmount`, `NetAmount`
(both `nil` until the payment settles), `SettledAmount`, `ReceivedAmount`,
`UnderpaidBps`, `CreditUsdt`, `CreditRate` and `CreditAsset`, plus the
`Reversed` and `Underpaid` flags (`false` when absent). Unknown response fields
are ignored, so newer API versions never break decoding.

### Settlement asset

Your balance is held in a stablecoin: **USDC** for merchants registered in the
European Union (USDT is not authorised under MiCA), **USDT** for everyone else.
Once a payment is credited, `Intent.CreditAsset` says which one
(`qint.SettlementAssetUSDT` / `qint.SettlementAssetUSDC`), `CreditUsdt` is the
amount credited in that asset (the field name predates USDC) and `CreditRate`
the rate, in units of the intent's currency per 1 unit of `CreditAsset`.

### Currencies

Qint can switch a fiat currency off for new payments (and back on) at any
time. `ListCurrencies` calls the public `GET /api/v1/public/currencies`
endpoint; cache the result for about 30 seconds and offer only the enabled
ones:

```go
list, err := client.ListCurrencies(ctx)
if err != nil {
	log.Fatal(err)
}
for _, c := range list.EnabledCurrencies() { // e.g. [CHF EUR USD GBP]
	fmt.Println("offer", c)
}
if !list.IsEnabled(qint.CurrencyGBP) {
	// hide GBP at checkout
}
```

Creating an intent in a disabled currency returns a `400` `*QintError` with no
`Code` and a `Detail` such as `"GBP is currently disabled on Qint. currency
must be one of: CHF, EUR, USD."`. Intents already created in that currency keep
working.

### Errors

Any non-2xx response is returned as a typed `*QintError` carrying the HTTP
status, the API's RFC 7807 problem-details `detail` message, the
machine-readable `Code` (when sent), the `TraceID` (quote it to support) and
every other problem member in `Extensions`:

```go
intent, err := client.CreateIntent(ctx, params)
if err != nil {
	var qerr *qint.QintError
	if errors.As(err, &qerr) {
		// qerr.StatusCode, qerr.Title, qerr.Detail
		if qerr.StatusCode == 403 {
			log.Fatalf("missing scope: %s", qerr.Detail)
		}
	}
	log.Fatal(err)
}
```

### Payment limits

Each payment must be between CHF 10 and CHF 100'000 (or the equivalent in EUR,
USD or GBP, or your individually agreed cap). Outside that range `CreateIntent` returns a `400` `*QintError` with
`Code == qint.ErrCodeAmountOutOfRange` (`"amount_out_of_range"`) and the numbers
in `MinChf`, `MaxChf` and `AmountChf` (`*json.Number`). If no exchange rate is
available to judge a non-CHF amount it returns a `503` with
`Code == qint.ErrCodeRateUnavailable` (`"rate_unavailable"`) — retry in a minute.

`qint.ErrCodeSettlementAssetMismatch` (`"settlement_asset_mismatch"`) is also
defined: the dashboard's payout endpoints return it when a crypto payout is
requested in an asset other than your settlement asset. The merchant API this
client wraps does not return it.

```go
if errors.As(err, &qerr) && qerr.Code == qint.ErrCodeAmountOutOfRange {
	log.Printf("%s (allowed CHF %s–%s)", qerr.Detail, qerr.MinChf, qerr.MaxChf)
}
```

### Underpaid payments

A payment that comes up short but within your accepted underpayment still
settles. The `payment.status` webhook then carries `Underpaid` = true,
`ReceivedAmount` (the fiat amount actually paid, in the intent's currency) and
`UnderpaidBps` (how short, in basis points of the invoice); both are `nil`
otherwise. `Intent` declares the same two fields. A non-nil `ReversedAt` means
the payment was reversed after settlement: treat it as not paid even though its
status stays `settled`.

## Webhooks

Configure an endpoint in the Qint dashboard to receive a signing secret
(`whsec_...`). Each delivery is a JSON `POST` with headers:

- `X-Qint-Signature: sha256=<hex HMAC-SHA256 of the raw body>`
- `X-Qint-Event-Id: <id>` — use it to **deduplicate** re-deliveries

Always verify the signature against the **exact raw request body bytes** before
trusting the payload, and acknowledge quickly with a 2xx.

```go
func handleQintWebhook(signingSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Constant-time HMAC-SHA256 verification.
		if !qint.VerifyWebhookSignature(body, r.Header.Get("X-Qint-Signature"), signingSecret) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		event, err := qint.ParseEvent(body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Dedupe on the event id before doing any work.
		eventID := r.Header.Get("X-Qint-Event-Id")
		if alreadyProcessed(eventID) {
			w.WriteHeader(http.StatusOK)
			return
		}

		if event.Type != qint.EventTypePaymentStatus {
			w.WriteHeader(http.StatusOK) // ping or a newer event type: ack and ignore
			return
		}

		switch event.Status {
		case qint.StatusSettled:
			// fulfill the order for event.IntentID
		case qint.StatusFailed, qint.StatusExpired, qint.StatusCancelled:
			// release / cancel
		}

		w.WriteHeader(http.StatusOK) // ack fast
	}
}
```

Event types are `qint.EventTypePaymentStatus` (`"payment.status"`) and
`qint.EventTypePing` (`"ping"`, the dashboard's **Send test event**; decode it
with `qint.ParsePingEvent` to get `EndpointID` and `SentAt`). `ParseEvent`
decodes any type without error — check `event.Type`, act on
`payment.status`, and acknowledge (2xx) and ignore anything else so future
event types never break your handler.

`PaymentStatusEvent` fields: `Type` (`"payment.status"`), `IntentID`, `Status`,
`Amount`, `Currency`, `AssetSymbol`, `InvoiceID?`, `PaymentLinkID?`,
`OccurredAt`, `Underpaid?`, `ReceivedAmount?`, `UnderpaidBps?`,
`ExpectedCryptoAmount?`, `ReceivedCryptoAmount?`, `Reversed?`, `ReversedAt?`.

## Development

```sh
go test ./...      # runs fully offline (httptest + stdlib)
go vet ./...
gofmt -l .
```

## Links

- API reference & docs: <https://docs.qint.ch>
- Releasing new versions: [PUBLISH.md](./PUBLISH.md)

## License

[MIT](./LICENSE) © SwizzX GmbH
