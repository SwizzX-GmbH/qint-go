package qint

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testClient spins up an httptest server running handler and returns a Client
// pointed at it (with a realistic /api/v1 base path).
func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("qk_live_test", WithBaseURL(srv.URL+"/api/v1"))
}

func TestCreateIntent(t *testing.T) {
	var gotBody map[string]any
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/intents" {
			t.Errorf("path = %s, want /api/v1/intents", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer qk_live_test" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer qk_live_test")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}
		data, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, &gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{
			"id":"pi_123","status":"initiated","amount":19.90,"currency":"CHF",
			"title":"Order 42","checkoutUrl":"https://checkout.qint.ch/pi_123",
			"returnUrl":"https://shop.example/thanks",
			"createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:15:00Z"
		}`)
	})

	intent, err := client.CreateIntent(context.Background(), CreateIntentParams{
		Amount:         19.90,
		Currency:       CurrencyCHF,
		Title:          "Order 42",
		IdempotencyKey: "idem-1",
		ReturnURL:      "https://shop.example/thanks",
	})
	if err != nil {
		t.Fatalf("CreateIntent: %v", err)
	}

	// Request body shaping.
	if gotBody["amount"] != 19.9 {
		t.Errorf("body amount = %v, want 19.9", gotBody["amount"])
	}
	if gotBody["currency"] != "CHF" {
		t.Errorf("body currency = %v, want CHF", gotBody["currency"])
	}
	if gotBody["title"] != "Order 42" {
		t.Errorf("body title = %v, want Order 42", gotBody["title"])
	}
	if gotBody["idempotencyKey"] != "idem-1" {
		t.Errorf("body idempotencyKey = %v, want idem-1", gotBody["idempotencyKey"])
	}
	if gotBody["returnUrl"] != "https://shop.example/thanks" {
		t.Errorf("body returnUrl = %v", gotBody["returnUrl"])
	}

	// Response decoding.
	if intent.ID != "pi_123" {
		t.Errorf("intent.ID = %q, want pi_123", intent.ID)
	}
	if intent.Status != StatusInitiated {
		t.Errorf("intent.Status = %q, want initiated", intent.Status)
	}
	if intent.CheckoutURL != "https://checkout.qint.ch/pi_123" {
		t.Errorf("intent.CheckoutURL = %q", intent.CheckoutURL)
	}
	if intent.Amount.String() != "19.90" {
		t.Errorf("intent.Amount = %q, want 19.90", intent.Amount.String())
	}
}

func TestCreateIntentOmitsEmptyOptionalFields(t *testing.T) {
	var gotBody map[string]any
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"pi_1","status":"initiated","amount":5,"currency":"EUR","checkoutUrl":"https://checkout.qint.ch/pi_1","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:15:00Z"}`)
	})

	if _, err := client.CreateIntent(context.Background(), CreateIntentParams{
		Amount:   5,
		Currency: CurrencyEUR,
	}); err != nil {
		t.Fatalf("CreateIntent: %v", err)
	}

	for _, k := range []string{"title", "idempotencyKey", "returnUrl"} {
		if _, present := gotBody[k]; present {
			t.Errorf("body unexpectedly includes empty optional field %q", k)
		}
	}
}

func TestGetIntent(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/intents/pi_abc" {
			t.Errorf("path = %s, want /api/v1/intents/pi_abc", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"pi_abc","status":"settled","amount":5,"currency":"EUR","assetSymbol":"USDT","cryptoAmount":"5.12345678","checkoutUrl":"https://checkout.qint.ch/pi_abc","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:15:00Z","confirmedAt":"2026-07-07T10:03:00Z","settledAt":"2026-07-07T10:05:00Z"}`)
	})

	intent, err := client.GetIntent(context.Background(), "pi_abc")
	if err != nil {
		t.Fatalf("GetIntent: %v", err)
	}
	if intent.Status != StatusSettled {
		t.Errorf("status = %q, want settled", intent.Status)
	}
	if intent.CryptoAmount != "5.12345678" {
		t.Errorf("cryptoAmount = %q", intent.CryptoAmount)
	}
	if intent.ConfirmedAt == nil {
		t.Error("ConfirmedAt = nil, want non-nil")
	}
	if intent.SettledAt == nil {
		t.Error("SettledAt = nil, want non-nil")
	}
}

func TestListIntents(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/intents" {
			t.Errorf("path = %s, want /api/v1/intents", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("status") != "pending" {
			t.Errorf("status = %q, want pending", q.Get("status"))
		}
		if q.Get("page") != "2" {
			t.Errorf("page = %q, want 2", q.Get("page"))
		}
		if q.Get("pageSize") != "50" {
			t.Errorf("pageSize = %q, want 50", q.Get("pageSize"))
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"items":[{"id":"pi_1","status":"pending","amount":10,"currency":"CHF","checkoutUrl":"https://checkout.qint.ch/pi_1","createdAt":"2026-07-07T10:00:00Z","expiresAt":"2026-07-07T10:15:00Z"}],"total":1,"page":2,"pageSize":50}`)
	})

	list, err := client.ListIntents(context.Background(), ListIntentsParams{
		Status:   StatusPending,
		Page:     2,
		PageSize: 50,
	})
	if err != nil {
		t.Fatalf("ListIntents: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("list = %+v, want total 1 with 1 item", list)
	}
	if list.Items[0].ID != "pi_1" {
		t.Errorf("items[0].ID = %q, want pi_1", list.Items[0].ID)
	}
	if list.Page != 2 || list.PageSize != 50 {
		t.Errorf("page/pageSize = %d/%d, want 2/50", list.Page, list.PageSize)
	}
}

func TestListIntentsOmitsUnsetParams(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if raw := r.URL.RawQuery; raw != "" {
			t.Errorf("RawQuery = %q, want empty", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"items":[],"total":0,"page":1,"pageSize":20}`)
	})

	if _, err := client.ListIntents(context.Background(), ListIntentsParams{}); err != nil {
		t.Fatalf("ListIntents: %v", err)
	}
}

func TestAPIErrorProblemDetails(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"type":"about:blank","title":"Forbidden","status":403,"detail":"This API key lacks the Write scope."}`)
	})

	_, err := client.CreateIntent(context.Background(), CreateIntentParams{
		Amount:   1,
		Currency: CurrencyCHF,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var qerr *QintError
	if !errors.As(err, &qerr) {
		t.Fatalf("error type = %T, want *QintError", err)
	}
	if qerr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want 403", qerr.StatusCode)
	}
	if qerr.Detail != "This API key lacks the Write scope." {
		t.Errorf("Detail = %q", qerr.Detail)
	}
	if qerr.Title != "Forbidden" {
		t.Errorf("Title = %q, want Forbidden", qerr.Title)
	}
}
