package qint

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func numStr(n *json.Number) string {
	if n == nil {
		return "<nil>"
	}
	return n.String()
}

func TestCreateIntentGBP(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["currency"] != "GBP" {
			t.Errorf("currency = %v, want GBP", body["currency"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"pi_gbp","status":"initiated","amount":25,"currency":"GBP","checkoutUrl":"https://checkout.qint.ch/pay/pi_gbp","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:30:00Z"}`)
	})

	intent, err := client.CreateIntent(context.Background(), CreateIntentParams{
		Amount:         25,
		Currency:       CurrencyGBP,
		IdempotencyKey: "order-gbp",
	})
	if err != nil {
		t.Fatalf("CreateIntent: %v", err)
	}
	if intent.Currency != CurrencyGBP {
		t.Errorf("Currency = %q, want GBP", intent.Currency)
	}
}

func TestGetIntentFeeUnderpaymentAndReversalFields(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"pi_short","status":"settled","amount":100,"currency":"GBP","checkoutUrl":"https://checkout.qint.ch/pay/pi_short","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:30:00Z","settledAt":"2026-07-07T10:20:00Z","feeAmount":0.98,"netAmount":97.02,"settledAmount":null,"receivedAmount":98.00,"underpaidBps":200,"creditUsdt":108.9,"creditRate":0.8909,"reversedAt":"2026-07-08T09:00:00Z"}`)
	})

	intent, err := client.GetIntent(context.Background(), "pi_short")
	if err != nil {
		t.Fatalf("GetIntent: %v", err)
	}
	checks := map[string][2]string{
		"FeeAmount":      {numStr(intent.FeeAmount), "0.98"},
		"NetAmount":      {numStr(intent.NetAmount), "97.02"},
		"SettledAmount":  {numStr(intent.SettledAmount), "<nil>"},
		"ReceivedAmount": {numStr(intent.ReceivedAmount), "98.00"},
		"CreditUsdt":     {numStr(intent.CreditUsdt), "108.9"},
		"CreditRate":     {numStr(intent.CreditRate), "0.8909"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %s, want %s", name, c[0], c[1])
		}
	}
	if intent.UnderpaidBps == nil || *intent.UnderpaidBps != 200 {
		t.Errorf("UnderpaidBps = %v, want 200", intent.UnderpaidBps)
	}
	if intent.ReversedAt == nil || intent.ReversedAt.Day() != 8 {
		t.Errorf("ReversedAt = %v, want 2026-07-08", intent.ReversedAt)
	}
}

func TestGetIntentWithoutNewFields(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"pi_abc","status":"initiated","amount":5,"currency":"CHF","checkoutUrl":"https://checkout.qint.ch/pay/pi_abc","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:30:00Z"}`)
	})

	intent, err := client.GetIntent(context.Background(), "pi_abc")
	if err != nil {
		t.Fatalf("GetIntent: %v", err)
	}
	if intent.FeeAmount != nil || intent.NetAmount != nil || intent.SettledAmount != nil ||
		intent.ReceivedAmount != nil || intent.UnderpaidBps != nil || intent.CreditUsdt != nil ||
		intent.CreditRate != nil || intent.ReversedAt != nil {
		t.Errorf("optional fields should be nil when absent: %+v", intent)
	}
}

func TestAmountOutOfRangeError(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"type":"https://tools.ietf.org/html/rfc9110#section-15.5.1","title":"Amount out of range","status":400,"detail":"Payments must be between CHF 10 and CHF 100'000 (or the equivalent). This amount is CHF 5.00.","traceId":"00-abc-def-00","code":"amount_out_of_range","minChf":10,"maxChf":100000,"amountChf":5.00}`)
	})

	_, err := client.CreateIntent(context.Background(), CreateIntentParams{
		Amount: 5, Currency: CurrencyCHF, IdempotencyKey: "order-5",
	})
	var qerr *QintError
	if !errors.As(err, &qerr) {
		t.Fatalf("error type = %T, want *QintError", err)
	}
	if qerr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", qerr.StatusCode)
	}
	if qerr.Code != ErrCodeAmountOutOfRange {
		t.Errorf("Code = %q, want amount_out_of_range", qerr.Code)
	}
	if numStr(qerr.MinChf) != "10" || numStr(qerr.MaxChf) != "100000" || numStr(qerr.AmountChf) != "5.00" {
		t.Errorf("limits = %s/%s/%s, want 10/100000/5.00", numStr(qerr.MinChf), numStr(qerr.MaxChf), numStr(qerr.AmountChf))
	}
	if qerr.Detail != "Payments must be between CHF 10 and CHF 100'000 (or the equivalent). This amount is CHF 5.00." {
		t.Errorf("Detail = %q", qerr.Detail)
	}
	if qerr.Extensions["traceId"] != "00-abc-def-00" || qerr.Extensions["code"] != "amount_out_of_range" {
		t.Errorf("Extensions = %v", qerr.Extensions)
	}
	if qerr.Extensions["minChf"] != json.Number("10") {
		t.Errorf("Extensions[minChf] = %#v, want json.Number(10)", qerr.Extensions["minChf"])
	}
	if _, ok := qerr.Extensions["title"]; ok {
		t.Error("Extensions should not contain standard members")
	}
}

func TestRateUnavailableError(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"title":"Exchange rate unavailable","status":503,"detail":"Exchange rate unavailable — try again in a minute.","code":"rate_unavailable"}`)
	})

	_, err := client.CreateIntent(context.Background(), CreateIntentParams{
		Amount: 50, Currency: CurrencyGBP, IdempotencyKey: "order-50",
	})
	var qerr *QintError
	if !errors.As(err, &qerr) {
		t.Fatalf("error type = %T, want *QintError", err)
	}
	if qerr.StatusCode != http.StatusServiceUnavailable || qerr.Code != ErrCodeRateUnavailable {
		t.Errorf("got %d %q, want 503 rate_unavailable", qerr.StatusCode, qerr.Code)
	}
	if qerr.MinChf != nil {
		t.Errorf("MinChf = %v, want nil", qerr.MinChf)
	}
}

func TestParseEventAcceptedUnderpayment(t *testing.T) {
	raw := []byte(`{
		"type":"payment.status","intentId":"pi_short","status":"settled",
		"amount":100,"currency":"GBP","assetSymbol":"USDT",
		"underpaid":true,"receivedAmount":98.00,"underpaidBps":200,
		"expectedCryptoAmount":"126.58227848","receivedCryptoAmount":"124.05063291",
		"invoiceId":null,"paymentLinkId":null,"reversed":false,"reversedAt":null,
		"occurredAt":"2026-07-07T12:00:00Z"
	}`)
	ev, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if ev.Currency != CurrencyGBP {
		t.Errorf("Currency = %q, want GBP", ev.Currency)
	}
	if ev.Underpaid == nil || !*ev.Underpaid {
		t.Errorf("Underpaid = %v, want true", ev.Underpaid)
	}
	if numStr(ev.ReceivedAmount) != "98.00" {
		t.Errorf("ReceivedAmount = %s, want 98.00", numStr(ev.ReceivedAmount))
	}
	if ev.UnderpaidBps == nil || *ev.UnderpaidBps != 200 {
		t.Errorf("UnderpaidBps = %v, want 200", ev.UnderpaidBps)
	}
	if ev.ReceivedCryptoAmount != "124.05063291" {
		t.Errorf("ReceivedCryptoAmount = %q", ev.ReceivedCryptoAmount)
	}
	if ev.Reversed == nil || *ev.Reversed || ev.ReversedAt != nil {
		t.Errorf("Reversed/ReversedAt = %v/%v, want false/nil", ev.Reversed, ev.ReversedAt)
	}
}

func TestParseEventWithoutUnderpaymentFields(t *testing.T) {
	ev, err := ParseEvent([]byte(`{"type":"payment.status","intentId":"pi_9","status":"pending","amount":42.5,"currency":"CHF","occurredAt":"2026-07-07T12:00:00Z"}`))
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if ev.ReceivedAmount != nil || ev.UnderpaidBps != nil || ev.Reversed != nil || ev.ReversedAt != nil {
		t.Errorf("optional fields should be nil when absent: %+v", ev)
	}
}
