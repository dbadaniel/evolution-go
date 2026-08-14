# Deferred Work - upstream-0.7.2-meta-security

## Message pin support

- Status: re-evaluated 2026-07-31 -- likely no longer blocked, but still needs a real WhatsApp confirmation before being called fully supported.
- Root API commit: `793d01d feat: add message pin endpoints`.
- Local library experiment: `whatsmeow-lib` commit `0a49705 feat: support message pin edit attribute` (still not referenced by `go.mod`, irrelevant to the current build).
- Correction: the official `go.mau.fi/whatsmeow v0.0.0-20260630180629-b572e5bcb92b` pinned in `go.mod` (the exact version this API builds against) already detects `PinInChatMessage` and returns `types.EditAttributePinInChat` (`send.go`, `getEditAttribute`), i.e. it already sends the `edit=2` behavior this deferral was waiting on. The original 2026-07-07 note assumed this was still missing from the official dependency; that assumption does not hold for the currently pinned version.
- `/message/pin` and `/message/unpin` are fully wired end to end (handler -> service -> official whatsmeow `SendMessage` with `PinInChatMessage`) and should work in principle.
- Remaining gap: nobody has run the real WhatsApp smoke test (send a message, capture `messageId`, call `/message/pin`, confirm on a phone that it shows pinned in the chat; then `/message/unpin` and confirm it clears). Do not mark this feature as fully supported in `docs/fork-delta.md` until that manual check passes.

## Manager polling flicker source fix

- Status: bundle corrigido em 2026-08-13 com o patch conhecido de `f79c5c4` e
  cache-busting no nome do asset.
- Pendencia: replicar a logica `hasLoaded` no repositorio-fonte externo do
  Evolution GO Manager. Este repositorio contem apenas `manager/dist`; uma nova
  compilacao feita a partir de uma fonte sem a correcao pode reintroduzir o
  flicker.
- Validacao futura: executar smoke visual na tela `/manager/instances` por mais
  de dois ciclos de polling, incluindo falha e recuperacao da API.

## Link metadata HTML response limit

- Status: preexistente, identificado durante a revisao do preview de links em
  2026-08-14.
- Pendencia: `fetchHTMLLinkMetadata` entrega o corpo HTML diretamente ao parser
  sem um limite de leitura. Avaliar um teto suficientemente alto e um fallback
  que preserve paginas validas grandes, sem reintroduzir a regressao de rejeitar
  todo HTML acima do limite pequeno usado exclusivamente pelo oEmbed.
