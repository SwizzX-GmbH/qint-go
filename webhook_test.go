package qint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	body := []byte(`{"type":"payment.status","intentId":"pi_1","status":"confirmed"}`)
	secret := "whsec_test_secret"
	sig := sign(body, secret)

	if !VerifyWebhookSignature(body, sig, secret) {
		t.Error("valid signature was rejected")
	}
	if VerifyWebhookSignature([]byte(`{"tampered":true}`), sig, secret) {
		t.Error("tampered body was accepted")
	}
	if VerifyWebhookSignature(body, "sha256=deadbeef", secret) {
		t.Error("tampered signature was accepted")
	}
	if VerifyWebhookSignature(body, sig, "whsec_wrong_secret") {
		t.Error("wrong secret was accepted")
	}
	if VerifyWebhookSignature(body, "", secret) {
		t.Error("empty signature was accepted")
	}
	if VerifyWebhookSignature(body, sig, "") {
		t.Error("empty secret was accepted")
	}
}

func TestVerifyWebhookSignatureAcceptsBareHex(t *testing.T) {
	body := []byte(`hello world`)
	secret := "whsec_x"
	bare := strings.TrimPrefix(sign(body, secret), signaturePrefix)

	if !VerifyWebhookSignature(body, bare, secret) {
		t.Error("bare hex signature (no sha256= prefix) was rejected")
	}
}

func TestParseEvent(t *testing.T) {
	raw := []byte(`{
		"type":"payment.status","intentId":"pi_9","status":"settled",
		"amount":42.5,"currency":"CHF","assetSymbol":"USDT",
		"invoiceId":"in_1","occurredAt":"2026-07-07T12:00:00Z",
		"underpaid":false,"expectedCryptoAmount":"42.50000000",
		"receivedCryptoAmount":"42.50000000"
	}`)

	ev, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if ev.Type != "payment.status" {
		t.Errorf("Type = %q, want payment.status", ev.Type)
	}
	if ev.IntentID != "pi_9" {
		t.Errorf("IntentID = %q, want pi_9", ev.IntentID)
	}
	if ev.Status != StatusSettled {
		t.Errorf("Status = %q, want settled", ev.Status)
	}
	if ev.AssetSymbol != "USDT" {
		t.Errorf("AssetSymbol = %q, want USDT", ev.AssetSymbol)
	}
	if ev.InvoiceID != "in_1" {
		t.Errorf("InvoiceID = %q, want in_1", ev.InvoiceID)
	}
	if ev.Amount.String() != "42.5" {
		t.Errorf("Amount = %q, want 42.5", ev.Amount.String())
	}
	if ev.Underpaid == nil || *ev.Underpaid {
		t.Errorf("Underpaid = %v, want non-nil false", ev.Underpaid)
	}
	if ev.OccurredAt.IsZero() {
		t.Error("OccurredAt is zero, want parsed timestamp")
	}
}

func TestParseEventInvalidJSON(t *testing.T) {
	if _, err := ParseEvent([]byte(`{not json`)); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
