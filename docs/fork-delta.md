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
- CTA, Pix, media-bearing reply buttons, and list messages retain their
  `DocumentWithCaptionMessage` wrapping with `MessageSecret` where upstream
  uses it. Text-only reply buttons are the exception described below.
- Reply-only buttons without media use a top-level native-flow
  `InteractiveMessage` with `quick_reply` buttons. Its top-level
  `MessageContextInfo` carries `MessageSecret`, `DeviceListMetadata`, and
  `DeviceListMetadataVersion=2`. The send includes the
  full official-client-style `biz` node (`actual_actors`, `host_storage`, Unix
  `privacy_mode_ts`, `native_flow v=9 name=mixed`, and `quality_control` with a
  cryptographically random decision ID), with no `bot` node. This follows the
  device-validated transport shape documented in `tulir/whatsmeow#1235`. The
  `ViewOnceMessage` wrapper was removed after the wrapped variant displayed a
  load error in WhatsApp for Windows. The top-level variant has now been
  confirmed by manual test on both a mobile phone and WhatsApp for Windows,
  using two text-only `reply` buttons.
- Reply buttons with `imageUrl` or `videoUrl` remain on the legacy,
  media-capable `ButtonsMessage` inside `DocumentWithCaptionMessage`. This only
  preserves the previous path: it does not guarantee remote download/upload
  success or mobile rendering for media replies.
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

Ported 2026-07-31 (re-evaluation after the passkey/security pass above):

- `ForwardingScore`: `SendDataStruct`, `TextStruct`, and `MediaStruct` gained an
  optional `ForwardingScore *uint32` field. `SendMessage` applies it (plus
  `ContextInfo.IsForwarded`) to whichever message type's `ContextInfo` was set,
  mirroring upstream's block exactly (same message types upstream covers;
  upstream does not apply it to `ButtonsMessage` either, so neither do we).
  Exposed publicly only on `/send/text` and `/send/media`, matching upstream.
- PDF document thumbnails: added `(s *sendService) makePDFThumbnail`, a
  method-form port of upstream's standalone `makePDFThumbnail` that shells out
  to `pdftoppm` (poppler-utils) to rasterize page 1, then reuses the existing
  local `resizeThumbnail` helper for the final JPEG encode instead of also
  porting upstream's separate `makeJPEGThumbnail` (redundant with
  `resizeThumbnail`, which already covers the generic-image-thumbnail case).
  Wired into the `document` case of both `SendMediaFile` and `SendMediaUrl`
  when `mimeType == "application/pdf"`. `Dockerfile` final stage now installs
  `poppler-utils` for `pdftoppm`.
- Message pin support (`/message/pin` and `/message/unpin`): the blocking
  assumption behind the original deferral -- that only the abandoned local
  `whatsmeow-lib` fork could send `PinInChatMessage`/`edit=2` -- no longer
  holds. The official `go.mau.fi/whatsmeow` version already pinned in `go.mod`
  (`v0.0.0-20260630180629-b572e5bcb92b`) detects `PinInChatMessage` natively in
  `send.go`'s `getEditAttribute`. The endpoint code (`793d01d`) was already
  correct against the official client; no code changes were needed. Still
  unverified: an actual WhatsApp smoke test confirming the message shows
  pinned on a phone. See
  `_evo-output/implementation-artifacts/upstream-0.7.2-meta-security/deferred-work.md`.

Ported 2026-08-13 after parity review against `upstream/main` (`9337afc`):

- Reactions use a fresh envelope ID and retain the referenced message ID only
  inside `MessageKey`; recipients and group participants use canonical JIDs.
- Raw protocol operations (typing, read and played receipts) canonicalize JIDs.
- `AlwaysOnline=false` is respected on connection; chat presence supports a
  bounded keepalive delay and explicit `paused` cleanup.
- Incoming messages are persisted when configured, including Meta ad referral
  metadata; later receipt updates preserve referral data and persistence errors
  are logged.
- Added `/message/markplayed` and webhook/global event support for `Picture` and
  `UserAbout`.

Preserved Expertsa behavior during this port:

- Empty/whitespace/duplicate receipt IDs continue to be filtered before
  persistence, deduplication and webhook dispatch, and all valid delivered IDs
  in a grouped receipt are processed.
- Local list transport, button workaround, link preview and pin endpoints were
  not replaced by their upstream counterparts.

Ported selectively from `NathanAshford/evolution-go-custom` for CTA error 473:

- CTA buttons (`url`, `call`, and `copy`) keep the existing native-flow button
  parameters, `DocumentWithCaptionMessage`, `MessageSecret`, `MessageVersion=1`,
  `MessageParamsJSON`, and `biz/mixed` transport nodes, but their nested
  `InteractiveMessage` now contains only `Body`, `Footer`, and
  `NativeFlowMessage`. The redundant `Header` and empty internal `ContextInfo`
  were removed because that larger payload was rejected by WhatsApp with 473.
- A real send with a single URL CTA was still rejected with 473 after slimming
  the protobuf payload. CTA transport now keeps `biz/native_flow/mixed` but omits
  `bot biz_bot=1`, avoiding a bot capability that the linked account may not
  have.
- Pix uses a top-level `InteractiveMessage`, a `payment_info` button with
  static-key `payment_settings`, `MessageParamsJSON={}`, `MessageVersion=1`,
  `MessageSecret`, device-list metadata v2, and a flat
  `<biz native_flow_name="payment_info"/>` node, with the fuller no-amount
  order payload used by WPPConnect. The complete send was confirmed on a phone
  and WhatsApp Web/Desktop through the local stage. The `order_details` params, nested
  `biz > interactive > native_flow` node, and `bot biz_bot=1` from Evolution
  API's Baileys-specific PR were tested here but did not produce a delivery
  receipt; that combination was discarded. CPF Pix keys are checked and
  normalized before sending. A
  `DeviceSentMessage` experiment was also reverted because it stopped delivery.

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
