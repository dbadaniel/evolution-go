# Deferred Work - upstream-0.7.2-meta-security

## Message pin support

- Status: deferred / experimental.
- Root API commit: `793d01d feat: add message pin endpoints`.
- Local library experiment: `whatsmeow-lib` commit `0a49705 feat: support message pin edit attribute`.
- Important: the API build currently uses official `go.mau.fi/whatsmeow` from `go.mod`, with no `replace` to `./whatsmeow-lib`. Therefore the local `whatsmeow-lib` experiment does not affect the Docker image/runtime.
- Do not treat `/message/pin` and `/message/unpin` as fully supported until the active whatsmeow dependency sends `PinInChatMessage` with the required `edit=2` behavior and a real WhatsApp smoke test confirms the message is visually pinned.
- Preferred future paths:
  - confirm official upstream support and update `go.mau.fi/whatsmeow`;
  - use a versioned fork with the minimal patch;
  - use a temporary `replace go.mau.fi/whatsmeow => ./whatsmeow-lib` only for an explicit controlled test.
- Release focus remains passkey / Meta security. Do not reintroduce the local `whatsmeow-lib` dependency silently just to finish this feature.
