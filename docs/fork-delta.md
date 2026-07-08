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

## Send Message Reconciliation

Ported from upstream `0.7.2`:

- `SendMessage` can pass `AdditionalNodes` to whatsmeow send options.
- Button/list interactive sends now add Meta-compatible business/bot nodes.
- CTA, Pix, reply buttons, and list messages are wrapped through
  `DocumentWithCaptionMessage` with `MessageSecret` where upstream does so.
- Reply-only buttons use `ButtonsMessage` instead of forcing all replies through
  native-flow `InteractiveMessage`.
- Reply-only buttons intentionally omit `AdditionalNodes`: real WhatsApp testing
  showed both `native_flow` and `buttons` biz nodes return `405` with the current
  stack. The message is now accepted as `ButtonsMessage`, but visual rendering
  remains a pre-existing button-rendering issue to investigate separately.
- Copy buttons now keep Expertsa input compatibility but use safer fallbacks for
  `id` and `copy_code`.
- Quoted-message support recognizes nested interactive/list/buttons payloads
  inside `DocumentWithCaptionMessage`.
- List sends preserve the old Expertsa/`whatsmeow-lib` compatibility tweak:
  the injected `biz/list` node uses `type=product_list` even though the protobuf
  remains a single-select `ListMessage`. This mirrors the
  `marcelotadeujr/whatsmeow` fork discussion around list rendering.

Preserved from Expertsa:

- Local module/import path remains `github.com/EvolutionAPI/evolution-go`.
- Local carousel behavior from the preservation commit remains in place.
- Carousel was manually validated after the upstream sync and remained functional.
- Existing link metadata handling remains richer than upstream for timeout,
  user-agent, OpenGraph priority, and relative image URL resolution.
- Existing local thumbnail resize path remains in place.
- The legacy `product_list` transport-node behavior for lists is preserved from
  the pre-official-`whatsmeow` fork delta.

Deferred for a separate decision:

- Whether to port upstream `ForwardingScore` behavior.
- Whether to replace or merge local thumbnail helpers with upstream's JPEG/PDF
  thumbnail helper style.
- Message pin support (`/message/pin` and `/message/unpin`) is not part of the
  passkey/security release acceptance yet. The API endpoint commit exists in the
  root repository, but the transport-level `edit=2` experiment was made only in
  the local `whatsmeow-lib` directory. Because the root `go.mod` intentionally
  uses official `go.mau.fi/whatsmeow` without `replace`, that local library patch
  is not included in the API build. Revisit later via upstream support, a
  versioned fork, or an explicit temporary `replace`; do not silently return to
  the local `whatsmeow-lib` while the main goal remains passkey compatibility.

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
