// Package qint is the official Go client for the Qint merchant API.
//
// It is a thin, typed wrapper over the Qint public HTTP API: create payment
// intents, look them up, list them, and verify incoming webhook deliveries.
//
// # Quickstart
//
//	client := qint.NewClient("qk_live_...")
//
//	intent, err := client.CreateIntent(ctx, qint.CreateIntentParams{
//		Amount:    19.90,
//		Currency:  qint.CurrencyCHF,
//		Title:     "Order #1024",
//		ReturnURL: "https://shop.example/thanks",
//	})
//	if err != nil {
//		// non-2xx responses are returned as *qint.QintError
//	}
//	// Redirect the buyer to intent.CheckoutURL to complete payment.
//
// The default base URL is https://qint-api.fly.dev/api/v1 and can be
// overridden with WithBaseURL (an api.qint.ch host is planned).
//
// See https://docs.qint.ch for the full API reference.
package qint

// Version is the semantic version of this SDK.
const Version = "0.1.0"
