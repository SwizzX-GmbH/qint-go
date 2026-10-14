package qint

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestListCurrencies(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/public/currencies" {
			t.Errorf("got %s %s, want GET /api/v1/public/currencies", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"currencies":[{"code":"CHF","enabled":true},{"code":"EUR","enabled":true},{"code":"USD","enabled":false},{"code":"GBP","enabled":true},{"code":"JPY","enabled":true,"future":1}],"extra":"ignored"}`)
	})

	list, err := client.ListCurrencies(context.Background())
	if err != nil {
		t.Fatalf("ListCurrencies: %v", err)
	}
	if len(list.Currencies) != 5 {
		t.Fatalf("len = %d, want 5", len(list.Currencies))
	}
	want := []Currency{CurrencyCHF, CurrencyEUR, CurrencyGBP, Currency("JPY")}
	if got := list.EnabledCurrencies(); !reflect.DeepEqual(got, want) {
		t.Errorf("EnabledCurrencies = %v, want %v", got, want)
	}
	if list.IsEnabled(CurrencyUSD) {
		t.Error("IsEnabled(USD) = true, want false")
	}
	if !list.IsEnabled(CurrencyGBP) {
		t.Error("IsEnabled(GBP) = false, want true")
	}
	if list.IsEnabled(Currency("XXX")) {
		t.Error("IsEnabled(unknown) = true, want false")
	}
	var nilList *CurrencyList
	if nilList.EnabledCurrencies() != nil || nilList.IsEnabled(CurrencyCHF) {
		t.Error("nil list should be empty")
	}
}

func TestGetIntentCreditAssetAndFlags(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"pi_eu","status":"settled","amount":100,"currency":"EUR","checkoutUrl":"https://checkout.qint.ch/pay/pi_eu","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:15:00Z","reversed":true,"underpaid":true,"feeAmount":0.5,"netAmount":99.5,"creditUsdt":108.9,"creditRate":0.9137,"creditAsset":"USDC","someFutureField":{"x":1}}`)
	})
	intent, err := client.GetIntent(context.Background(), "pi_eu")
	if err != nil {
		t.Fatalf("GetIntent: %v", err)
	}
	if intent.CreditAsset == nil || *intent.CreditAsset != SettlementAssetUSDC {
		t.Errorf("CreditAsset = %v, want USDC", intent.CreditAsset)
	}
	if !intent.Reversed || !intent.Underpaid {
		t.Errorf("Reversed=%v Underpaid=%v, want true/true", intent.Reversed, intent.Underpaid)
	}
	if numStr(intent.FeeAmount) != "0.5" || numStr(intent.NetAmount) != "99.5" {
		t.Errorf("fee/net = %s/%s", numStr(intent.FeeAmount), numStr(intent.NetAmount))
	}
}

func TestGetIntentNullEconomicsBeforeSettlement(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"pi_new","status":"initiated","amount":10,"currency":"CHF","checkoutUrl":"https://checkout.qint.ch/pay/pi_new","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:15:00Z","reversed":false,"underpaid":false,"feeAmount":null,"netAmount":null,"creditUsdt":null,"creditRate":null,"creditAsset":null}`)
	})
	intent, err := client.GetIntent(context.Background(), "pi_new")
	if err != nil {
		t.Fatalf("GetIntent: %v", err)
	}
	if intent.FeeAmount != nil || intent.NetAmount != nil || intent.CreditAsset != nil || intent.CreditUsdt != nil {
		t.Errorf("expected nil economics, got %+v", intent)
	}
}

func TestUnknownSettlementAssetDecodes(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":"pi_x","status":"settled","amount":10,"currency":"CHF","checkoutUrl":"u","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:15:00Z","creditAsset":"EURC"}`)
	})
	intent, err := client.GetIntent(context.Background(), "pi_x")
	if err != nil {
		t.Fatalf("GetIntent: %v", err)
	}
	if intent.CreditAsset == nil || *intent.CreditAsset != "EURC" {
		t.Errorf("CreditAsset = %v, want EURC", intent.CreditAsset)
	}
}

func TestDisabledCurrencyError(t *testing.T) {
	const detail = "GBP is currently disabled on Qint. currency must be one of: CHF, EUR, USD."
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"type":"https://tools.ietf.org/html/rfc9110#section-15.5.1","title":"An error occurred while processing your request.","status":400,"detail":"`+detail+`","traceId":"00-abc-def-00"}`)
	})
	_, err := client.CreateIntent(context.Background(), CreateIntentParams{Amount: 20, Currency: CurrencyGBP, IdempotencyKey: "k"})
	var qerr *QintError
	if !errors.As(err, &qerr) {
		t.Fatalf("err = %v, want *QintError", err)
	}
	if qerr.StatusCode != 400 || qerr.Detail != detail || qerr.Code != "" {
		t.Errorf("got status=%d detail=%q code=%q", qerr.StatusCode, qerr.Detail, qerr.Code)
	}
	if qerr.TraceID != "00-abc-def-00" {
		t.Errorf("TraceID = %q", qerr.TraceID)
	}
	if qerr.Error() != "qint: HTTP 400: "+detail {
		t.Errorf("Error() = %q", qerr.Error())
	}
}

func TestSettlementAssetMismatchCode(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"title":"Wrong payout asset","status":400,"detail":"Your balance settles in USDC.","code":"settlement_asset_mismatch"}`)
	})
	_, err := client.GetIntent(context.Background(), "pi_x")
	var qerr *QintError
	if !errors.As(err, &qerr) || qerr.Code != ErrCodeSettlementAssetMismatch {
		t.Fatalf("err = %v, want code %s", err, ErrCodeSettlementAssetMismatch)
	}
}

func TestParsePingEvent(t *testing.T) {
	body := []byte(`{"type":"ping","endpointId":"7d1f6c3e-0000-4000-8000-000000000001","sentAt":"2026-10-14T12:00:00+00:00"}`)
	ping, err := ParsePingEvent(body)
	if err != nil {
		t.Fatalf("ParsePingEvent: %v", err)
	}
	if ping.Type != EventTypePing || ping.EndpointID == "" || ping.SentAt.IsZero() {
		t.Errorf("ping = %+v", ping)
	}
	// ParseEvent tolerates a ping (and any other type) without error.
	ev, err := ParseEvent(body)
	if err != nil || ev.Type != EventTypePing {
		t.Errorf("ParseEvent(ping) = %+v, %v", ev, err)
	}
	ev, err = ParseEvent([]byte(`{"type":"payout.paid","payoutId":"po_1"}`))
	if err != nil || ev.Type != "payout.paid" {
		t.Errorf("ParseEvent(unknown) = %+v, %v", ev, err)
	}
}
