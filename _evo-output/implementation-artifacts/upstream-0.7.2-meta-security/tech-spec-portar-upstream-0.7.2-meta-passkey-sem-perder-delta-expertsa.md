---
title: 'Portar upstream 0.7.2 Meta/passkey sem perder delta Expertsa'
type: 'feature'
created: '2026-07-06'
status: 'in-progress'
baseline_commit: '59f01015705de30fd468b6879c82e1e904fffeb5'
context: []
---

# Portar upstream 0.7.2 Meta/passkey sem perder delta Expertsa

<frozen-after-approval reason="human-owned intent - do not modify unless human renegotiates">

## Intent

**Problem:** O upstream `0.7.2` trouxe uma mudanca critica para compatibilidade com atualizacoes da Meta/WhatsApp: remove o fork `whatsmeow-lib`, usa `go.mau.fi/whatsmeow` oficial com suporte a passkey/WebAuthn, e altera o fluxo de QR/pairing. Nossa branch tambem tem alteracoes proprias em `pkg/sendMessage/service/send_service.go`, entao um merge bruto pode apagar correcoes Expertsa ou introduzir regressao em mensagens interativas.

**Approach:** Portar o nucleo funcional do upstream `0.7.2` em commits pequenos e rotulados por origem, mantendo imports/modulo locais quando possivel e documentando o delta. Primeiro entram dependencia `whatsmeow`, passkey, QR/pair/status e configuracao; depois `send_service.go` e renderizacao interativa sao reconciliados de forma consciente com o commit `ours: preserve button rendering WIP`.

## Boundaries & Constraints

**Always:** Preservar o commit `59f0101` como baseline das alteracoes nossas. Separar commits entre `upstream:` e `ours:` quando a origem/decisao for diferente. Manter a branch `sync-upstream-0.7.2-meta-security` como trilha de integracao, sem mexer em branches antigas. Priorizar comportamento de pareamento QR normal, pareamento passkey, `POST /instance/pair` e `GET /instance/status`.

**Ask First:** Trocar globalmente o module path/imports para `github.com/evolution-foundation/evolution-go`. Remover fisicamente arquivos grandes/binarios ou submodule com acao destrutiva fora do indice Git. Aceitar mudancas extensas no manager/dist ou swagger gerado. Sobrescrever a estrategia local de mensagens interativas sem mostrar o conflito e a decisao.

**Never:** Nao fazer merge bruto de `0.7.2`. Nao descartar alteracoes locais nao relacionadas. Nao esconder conflitos em `send_service.go` com uma substituicao total. Nao depender de bypass headless de passkey: o fluxo requer autenticador real via navegador em `web.whatsapp.com`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| QR normal | Instancia nova sem passkey exigido | QR continua sendo emitido, persistido e enviado a webhook/filas | Ao expirar ou atingir max count, limpa QR e emite `QRTimeout` sem escrita concorrente em maps |
| Passkey required | WhatsApp emite `PairPasskeyRequest` | API cria token curto, retorna stage/openUrl em QR poll, e endpoints publicos conduzem response/confirm | Token ausente/expirado retorna 404; erro de whatsmeow vira stage `error` |
| Pair phone | `POST /instance/pair` com client ausente/desconectado | Instancia inicia, espera conexao e retorna pairing code real | Erro de `PairPhone` nao vira 200 vazio; retorna erro explicito |
| Status disconnected | Instancia existente sem client ativo | `GET /instance/status` retorna 200 com `Connected=false`, `LoggedIn=false` | Nao força erro 400 por falta de client |
| Interactive messages | Nosso WIP e upstream 0.7.2 alteram wrappers/nodes | Decisao explicita preserva campos `Id` locais e usa abordagem compativel com Meta quando portada | Qualquer divergencia de renderizacao fica documentada e testada antes de commit final |

</frozen-after-approval>

## Code Map

- `go.mod`, `go.sum`, `.gitmodules`, `whatsmeow-lib` -- troca do fork/submodule para `go.mau.fi/whatsmeow` oficial e dependencias de seguranca/compatibilidade.
- `pkg/whatsmeow/service/whatsmeow.go` -- fluxo central: QR via `events.QR`, eventos `PairPasskey*`, store de passkey, envio de response/confirm e ajustes de webhook.
- `pkg/passkey/ceremony/store.go` -- estado efemero e thread-safe da cerimonia WebAuthn.
- `pkg/passkey/handler/passkey_handler.go` -- endpoints publicos chamados pela extensao no browser.
- `cmd/evolution-go/main.go` -- registra rotas publicas de passkey junto das rotas de licenca.
- `pkg/instance/service/instance_service.go` -- status desconectado, pair robusto e retorno de passkey no QR polling.
- `pkg/routes/routes.go`, `pkg/instance/handler/instance_handler.go` -- ajustes de rotas/swagger que nao devem puxar renomeacao global sem necessidade.
- `passkey-helper/*`, `.env.example` -- suporte operacional para navegador e `PASSKEY_PUBLIC_URL`.
- `pkg/sendMessage/service/send_service.go` -- area de maior conflito; reconciliar WIP Expertsa com wrappers/nodes do upstream.
- `docs/fork-delta.md` -- novo inventario do que foi portado do upstream e do que e decisao Expertsa.

## Tasks & Acceptance

**Execution:**
- [x] `docs/fork-delta.md` -- criar inventario inicial de deltas -- manter claro o que e upstream e o que e nosso.
- [x] `go.mod`, `go.sum`, `.gitmodules`, `whatsmeow-lib` -- portar dependencia oficial do `whatsmeow` sem renomear module path local -- habilitar passkey mantendo minimo blast radius.
- [x] `pkg/passkey/**`, `passkey-helper/**`, `.env.example` -- adicionar fluxo de cerimonia e helper -- suportar contas bloqueadas por passkey.
- [x] `pkg/whatsmeow/service/whatsmeow.go` -- portar QR/event/passkey core com imports locais -- manter socket vivo durante passkey e evitar concorrencia em maps.
- [x] `cmd/evolution-go/main.go`, `pkg/instance/service/instance_service.go` -- registrar rotas e portar pair/status/passkey QR info -- expor o fluxo ao manager/API.
- [x] `pkg/sendMessage/service/send_service.go` -- reconciliar o WIP com a abordagem 0.7.2 para mensagens interativas -- evitar regressao em botoes/listas/carrossel.
- [x] Executar build/testes possiveis -- confirmar compilacao e unidade basica dentro dos limites atuais do ambiente.

**Acceptance Criteria:**
- Given a branch with commit `59f0101`, when integration commits are reviewed, then local button-rendering WIP remains reachable and distinguishable from upstream ports.
- Given `go.mod`, when dependencies are resolved, then the code uses official `go.mau.fi/whatsmeow v0.0.0-20260630180629-b572e5bcb92b` without `replace ./whatsmeow-lib`.
- Given a passkey-required pairing event, when the browser helper polls the ceremony endpoint, then it receives stage/publicKey/code transitions and can submit response/confirm.
- Given a disconnected instance without active client, when status is requested, then the API returns disconnected state instead of an error.
- Given `send_service.go` conflicts, when final code is reviewed, then preserved Expertsa fields/behavior and upstream Meta-compatible rendering decisions are documented.

## Spec Change Log

- 2026-07-06: Reconciliado `pkg/sendMessage/service/send_service.go` em passos pequenos:
  `AdditionalNodes`, business/bot nodes, wrappers `DocumentWithCaptionMessage`,
  `MessageSecret`, `ButtonsMessage` para replies, listas modernas e fallback
  defensivo de copy button. Mantidos os comportamentos Expertsa ja existentes
  para carrossel, metadata de link e thumbnails locais.
- 2026-07-06: Itens upstream ainda candidatos a avaliacao separada:
  `ForwardingScore` e paridade fina dos helpers de thumbnail do upstream.
- 2026-07-07: Teste real em `devogo.expertsa.com.br`: reply buttons passaram de
  `server returned error 405` para envio aceito como `ButtonsMessage` com receipt
  e webhook apos omitir `AdditionalNodes` nesse caso. Renderizacao visual do
  botao no WhatsApp ainda nao ocorreu, mas foi classificada como pendencia ja
  existente na versao anterior, nao regressao nova desta integracao. Carrossel
  foi validado como funcional.
- 2026-07-07: Restaurado o delta local antigo de lista: o node `biz/list` usa
  `type=product_list`, alinhado ao changelog v0.7.0 e ao fork
  `marcelotadeujr/whatsmeow`, em vez de `type=single_select` do upstream puro.
- 2026-07-07: Corrigida regressao em `/send/link`: o preview automatico agora
  valida/converte a imagem para JPEG antes de preencher `JPEGThumbnail`, define
  `previewType` e dimensoes quando ha thumbnail valida, e evita enviar bytes de
  formatos nao suportados que podem deixar o app em "aguardando mensagem". A
  causa estrutural tambem foi corrigida: `SendMessage` nao deve zerar
  `ContextInfo` existente de `ExtendedTextMessage`, pois isso apaga
  `ExternalAdReply` do preview de link.

## Design Notes

Porting should be path-scoped rather than commit-cherry-pick because upstream `0.7.2` mixes functional changes with org rename and generated assets. Keep `github.com/EvolutionAPI/evolution-go` imports unless a specific file truly requires the new module path; this reduces blast radius and avoids a repo-wide import churn PR.

## Verification

**Commands:**
- `go test ./pkg/whatsmeow/service ./pkg/instance/service ./pkg/passkey/ceremony ./pkg/passkey/handler` -- passed on 2026-07-06.
- `go test ./...` -- currently blocked outside touched packages by `github.com/chai2010/webp` build errors (`undefined: webpGetInfo`, etc.).
- `go test ./pkg/sendMessage/service/...` -- attempted after reconciliation; blocked by the same `github.com/chai2010/webp`/CGO environment issue before package validation.
- `go build ./cmd/evolution-go` -- expected: compile succeeds with official whatsmeow dependency.
- `git diff --check` -- passed on 2026-07-06 after each `send_service.go` step.

**Manual checks (if no CLI):**
- Review QR normal, passkey-required pairing, pair-phone error handling, and button/list/carousel payload shape before release.
- 2026-07-07: `/send/carousel` passed manual smoke test. `/send/button` reply
  returns success/receipt/webhook as `ButtonsMessage`, but still needs a separate
  rendering investigation because the visual button was not delivered to the
  recipient app.
- 2026-07-07: `/send/list` initial smoke with `single_select` did not render;
  after restoring the `product_list` transport node, the manual smoke test passed.
- 2026-07-07: `/send/link` automatico apresentou preview incompleto/regressivo
  no app; patches aplicados para normalizar thumbnail JPEG e preservar
  `ContextInfo.ExternalAdReply` durante `SendMessage`. Requer rebuild e novo
  smoke test no Docker.
