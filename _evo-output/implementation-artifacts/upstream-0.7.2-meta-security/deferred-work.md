# Deferred Work - upstream-0.7.2-meta-security

## Message pin support

- Status: re-evaluated 2026-07-31 -- likely no longer blocked, but still needs a real WhatsApp confirmation before being called fully supported.
- Root API commit: `793d01d feat: add message pin endpoints`.
- Local library experiment: `whatsmeow-lib` commit `0a49705 feat: support message pin edit attribute` (still not referenced by `go.mod`, irrelevant to the current build).
- Correction: the official `go.mau.fi/whatsmeow v0.0.0-20260630180629-b572e5bcb92b` pinned in `go.mod` (the exact version this API builds against) already detects `PinInChatMessage` and returns `types.EditAttributePinInChat` (`send.go`, `getEditAttribute`), i.e. it already sends the `edit=2` behavior this deferral was waiting on. The original 2026-07-07 note assumed this was still missing from the official dependency; that assumption does not hold for the currently pinned version.
- `/message/pin` and `/message/unpin` are fully wired end to end (handler -> service -> official whatsmeow `SendMessage` with `PinInChatMessage`) and should work in principle.
- Remaining gap: nobody has run the real WhatsApp smoke test (send a message, capture `messageId`, call `/message/pin`, confirm on a phone that it shows pinned in the chat; then `/message/unpin` and confirm it clears). Do not mark this feature as fully supported in `docs/fork-delta.md` until that manual check passes.
