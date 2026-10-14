# Changelog

All notable changes to `github.com/SwizzX-GmbH/qint-go`. Versions follow
[semantic versioning](https://semver.org); a release is a `vX.Y.Z` git tag.

## v0.3.0 — 2026-10-14

### Added
- `ListCurrencies(ctx)` for the public `GET /api/v1/public/currencies`
  endpoint, with `CurrencyList`, `CurrencyAvailability` and the
  `EnabledCurrencies()` / `IsEnabled()` helpers. Cache the result ~30 s and
  hide currencies Qint has switched off.
- `SettlementAsset` (`SettlementAssetUSDT`, `SettlementAssetUSDC`) and
  `Intent.CreditAsset` — the stablecoin a payment was credited in (USDC for
  merchants registered in the EU, USDT otherwise).
- `Intent.Reversed` and `Intent.Underpaid` flags.
- `QintError.TraceID` (the problem's `traceId`).
- `ErrCodeSettlementAssetMismatch` (`settlement_asset_mismatch`, returned by
  the dashboard's payout endpoints).
- Webhook `EventTypePaymentStatus` / `EventTypePing` constants, `PingEvent`
  and `ParsePingEvent`.

### Changed
- Docs: `CreditUsdt` / `CreditRate` are in `CreditAsset` (the field name
  predates USDC); `FeeAmount` / `NetAmount` stay `nil` until settlement; a
  disabled currency is refused with `400`; per-payment limits note the agreed
  cap.

## v0.2.0 — 2026-10-14

- GBP, the accepted-underpayment / fee / credit / reversal intent fields and
  typed problem codes (`amount_out_of_range`, `rate_unavailable`).

## v0.1.0 — 2026-07-08

- Initial release: create / get / list intents, webhook signature
  verification, typed errors.
