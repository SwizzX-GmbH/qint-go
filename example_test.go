package qint_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	qint "github.com/SwizzX-GmbH/qint-go"
)

// Example shows the 30-second flow: create an intent, then send the buyer to
// the hosted checkout.
func Example() {
	client := qint.NewClient("qk_live_...")

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

	// Redirect the buyer to the hosted checkout to complete payment.
	fmt.Println(intent.CheckoutURL)
}

// ExampleVerifyWebhookSignature_handler verifies an incoming webhook against
// the raw request body, then parses it. Deduplicate on the X-Qint-Event-Id
// header and always acknowledge quickly with a 2xx.
func ExampleVerifyWebhookSignature_handler() {
	const signingSecret = "whsec_..." // from the Qint dashboard

	http.HandleFunc("/webhooks/qint", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if !qint.VerifyWebhookSignature(body, r.Header.Get("X-Qint-Signature"), signingSecret) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		event, err := qint.ParseEvent(body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		eventID := r.Header.Get("X-Qint-Event-Id")
		// TODO: skip if eventID was already processed (dedupe), then handle
		// event.Status for event.IntentID.
		_ = eventID
		_ = event

		w.WriteHeader(http.StatusOK) // ack fast
	})
}
