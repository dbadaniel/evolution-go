---
title: 'Adicionar pin de mensagem na API'
type: 'feature'
created: '2026-07-07'
status: 'done'
baseline_commit: '0582ce6'
context: []
---

# Adicionar pin de mensagem na API

<frozen-after-approval reason="human-owned intent - do not modify unless human renegotiates">

## Intent

**Problem:** A API hoje diferencia apenas pin/unpin de conversa em `/chat/pin` e `/chat/unpin`, mas nao expoe a acao de fixar uma mensagem especifica dentro de um chat. Isso limita automacoes que precisam destacar instrucoes, avisos ou links importantes no topo da conversa.

**Approach:** Criar endpoints pequenos em `/message/pin` e `/message/unpin` que recebam o chat, id da mensagem alvo, origem da mensagem e participante opcional para grupos. A implementacao deve usar `PinInChatMessage` do whatsmeow e ajustar a deteccao de atributo de edicao para enviar o no com `EditAttributePinInChat`.

## Boundaries & Constraints

**Always:** Manter o recurso separado de `/chat/pin`; validar `chat` e `messageId`; preservar suporte a chats diretos e grupos com `participant` opcional; retornar o id/timestamp da mensagem de controle enviada.

**Ask First:** Alterar assinatura publica de endpoints existentes; trocar nomes de campos ja usados por `/message/delete` ou `/message/edit`; adicionar duracao de pin se o protocolo local nao expuser esse campo claramente.

**Never:** Reaproveitar `appstate.BuildPin`, pois ele fixa conversa e nao mensagem; mexer nos fluxos de `/send/*`; assumir que pin de mensagem funciona sem teste real no WhatsApp.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|---------------|----------------------------|----------------|
| Pin de mensagem enviada pela propria instancia | `chat`, `messageId`, `fromMe=true` | Envia `PinInChatMessage` com tipo `PIN_FOR_ALL` para o chat alvo | Erro do WhatsApp retorna 500 com detalhe |
| Unpin de mensagem | Mesmo payload em `/message/unpin` | Envia `PinInChatMessage` com tipo `UNPIN_FOR_ALL` | Erro do WhatsApp retorna 500 com detalhe |
| Mensagem de grupo enviada por participante | Payload inclui `participant` | `MessageKey.Participant` e preenchido com JID valido | Participante invalido retorna erro de validacao |
| Payload incompleto | Falta `chat` ou `messageId` | Requisicao rejeitada antes do envio | 400 com campo obrigatorio |

</frozen-after-approval>

## Code Map

- `pkg/message/service/message_service.go` -- contratos, payload e envio da mensagem especial de pin/unpin.
- `pkg/message/handler/message_handler.go` -- handlers HTTP e validacao basica dos novos endpoints.
- `pkg/routes/routes.go` -- registro de `/message/pin` e `/message/unpin`.
- `whatsmeow-lib/send.go` -- experimento local para detectar `PinInChatMessage` e preencher atributo `edit=2`; **nao entra no build atual da API** porque `go.mod` usa `go.mau.fi/whatsmeow` oficial sem `replace`.
- `whatsmeow-lib/types/message.go` -- contem `EditAttributePinInChat`, mas so afeta a API se a dependencia local/fork versionado for usada.

## Current Build Status

- A API raiz tem o commit `793d01d feat: add message pin endpoints`, que adiciona `/message/pin` e `/message/unpin`.
- O diretorio `whatsmeow-lib` tem o commit local `0a49705 feat: support message pin edit attribute`, mas esse diretorio esta fora do build efetivo da API enquanto `go.mod` continuar apontando para `go.mau.fi/whatsmeow`.
- Portanto, o pin de mensagem deve ser considerado **experimental/incompleto** nesta branch: a rota pode existir e enviar `PinInChatMessage`, mas nao ha garantia de renderizacao/fixacao real no WhatsApp sem uma versao do whatsmeow que envie `edit=2`.
- Nao voltar a usar `replace go.mau.fi/whatsmeow => ./whatsmeow-lib` apenas por causa deste recurso sem decisao explicita, pois o objetivo principal da release e passkey/Meta security com a dependencia oficial.

## Tasks & Acceptance

**Execution:**
- [x] `whatsmeow-lib/send.go` -- reconhecer `PinInChatMessage` em `getEditAttribute` -- garantir envio com atributo compativel com pin de mensagem.
- [x] `pkg/message/service/message_service.go` -- adicionar payload e metodo `PinMessage` compartilhado por pin/unpin -- montar `MessageKey` com `chat`, `messageId`, `fromMe` e `participant`.
- [x] `pkg/message/handler/message_handler.go` -- adicionar handlers `PinMessage` e `UnpinMessage` -- validar campos obrigatorios e responder no padrao atual.
- [x] `pkg/routes/routes.go` -- registrar os endpoints no grupo `/message` -- expor a funcionalidade pela API.
- [x] Build/testes possiveis -- confirmar compilacao ou documentar bloqueio local conhecido.

**Acceptance Criteria:**
- Given uma mensagem existente em um chat direto, when `POST /message/pin` recebe `chat`, `messageId` e `fromMe`, then a API envia uma mensagem de controle `PinInChatMessage` para fixar a mensagem alvo.
- Given uma mensagem fixada, when `POST /message/unpin` recebe o mesmo alvo, then a API envia `UNPIN_FOR_ALL`.
- Given uma mensagem de grupo com participante informado, when o payload inclui `participant`, then o `MessageKey` enviado preserva esse participante.
- Given payload sem `chat` ou `messageId`, when a rota e chamada, then a API retorna 400 sem tentar enviar ao WhatsApp.

## Verification

**Commands:**
- `go test ./pkg/message/...` -- expected: compila ou falha apenas por limitacao local conhecida de CGO/webp, nao por erros do novo codigo.
- `go test ./whatsmeow-lib/...` -- expected: compila ou documenta bloqueios externos existentes.
- 2026-07-07: `go test ./pkg/message/...` passou.
- 2026-07-07: `go test ./...` dentro de `whatsmeow-lib` passou.
- 2026-07-07: `go test ./pkg/routes` falhou no bloqueio local conhecido de `github.com/chai2010/webp` no Windows, antes de validar rotas.
- 2026-07-07: `git diff --check` passou na raiz e em `whatsmeow-lib`.

**Manual checks:**
- Enviar mensagem normal, capturar o `messageId`, chamar `/message/pin` e verificar no celular se a mensagem aparece fixada dentro do chat.
- Chamar `/message/unpin` com o mesmo alvo e verificar se a mensagem deixa de aparecer fixada.

## Follow-Up Decision

- Decidir depois se o pin de mensagem deve ser retomado por uma destas vias:
  - upstream oficial do `go.mau.fi/whatsmeow` ja suportando `PinInChatMessage` com `edit=2`;
  - fork versionado do whatsmeow com patch pequeno e rastreavel;
  - `replace` local temporario apenas para teste controlado, nunca como mudanca silenciosa da release.
- Se a prioridade continuar sendo passkey, manter este recurso fora do escopo de release e nao usar os endpoints como evidencia de suporte completo.
