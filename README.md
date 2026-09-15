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
go get github.com/SwizzX-GmbH/qint-go@v0.1.0
```

> **Released.** `v0.1.0` is tagged and resolvable through the public Go module
> proxy (verified 2026-09-15). Go has no central registry to push to and needs no
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

```go
type CreateIntentParams struct {
	Amount         float64  // required, decimal, e.g. 19.90
	Currency       Currency // required: CurrencyCHF | CurrencyEUR | CurrencyUSD
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

### Errors

Any non-2xx response is returned as a typed `*QintError` carrying the HTTP
status and the API's RFC 7807 problem-details `detail` message:

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

`PaymentStatusEvent` fields: `Type` (`"payment.status"`), `IntentID`, `Status`,
`Amount`, `Currency`, `AssetSymbol`, `InvoiceID?`, `PaymentLinkID?`,
`OccurredAt`, `Underpaid?`, `ExpectedCryptoAmount?`, `ReceivedCryptoAmount?`.

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
