# Evolution GO Fork Delta

This file tracks the intentional differences between this fork and the original
upstream repository. Keep it updated when porting upstream releases so future
syncs can distinguish upstream code from Expertsa-specific decisions.

## Baseline

- Integration branch: `sync-upstream-0.7.2-meta-security`
- Local preservation commit: `59f0101 ours: preserve button rendering WIP`
- Upstream target: `0.7.2` (`9337afc sync: 0.7.2 from main`)

## Upstream 0.7.2 Being Ported

- Replace the local `whatsmeow-lib` submodule with official
  `go.mau.fi/whatsmeow v0.0.0-20260630180629-b572e5bcb92b`.
- Add passkey/WebAuthn pairing support for WhatsApp accounts that require the
  Meta passkey flow during device linking.
- Replace `GetQRChannel` pairing with `events.QR` handling so the socket remains
  alive during passkey ceremonies.
- Fix `POST /instance/pair` so pairing errors are surfaced instead of returning
  an empty `PairingCode`.
- Fix `GET /instance/status` so disconnected instances return a disconnected
  status instead of a client lookup error.

## Expertsa Decisions

- Do not do a raw merge of upstream `0.7.2`.
- Do not rename the module/import path globally to
  `github.com/evolution-foundation/evolution-go` in this integration pass.
- Preserve the local button-rendering work from commit `59f0101` and reconcile it
  explicitly with upstream's Meta-compatible interactive message changes.
- Keep generated manager/swagger updates out of the first security/passkey port
  unless they are needed for compile or API contract correctness.

## Sensitive Files

- `pkg/sendMessage/service/send_service.go`
  - Contains local Expertsa work in commit `59f0101`.
  - Upstream `0.7.2` also changes button/list/carousel rendering heavily with
    `DocumentWithCaptionMessage`, `MessageSecret`, and `AdditionalNodes`.
- `pkg/whatsmeow/service/whatsmeow.go`
  - Central pairing/event loop. Upstream passkey work must be ported carefully
    without introducing concurrent map writes or QR teardown regressions.
- `go.mod` / `go.sum`
  - Must move to official `go.mau.fi/whatsmeow` while keeping the local module
    path stable for now.

## Commit Label Convention

- `upstream:` direct port from upstream behavior.
- `ours:` local Expertsa behavior or reconciliation decision.
- `docs:` explanatory or tracking artifacts.
