---
title: 'Corrigir envio de botoes CTA rejeitado com erro 473'
type: 'bugfix'
created: '2026-08-14'
status: 'done'
baseline_commit: 'c53d771'
context:
  - 'docs/fork-delta.md'
---

# Corrigir envio de botoes CTA rejeitado com erro 473

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O endpoint `POST /send/button` retorna `server returned error 473` ao enviar uma combinacao valida de botoes CTA `url`, `call` e `copy`. O payload atual acrescenta `Header` e `ContextInfo` vazio ao `InteractiveMessage`, diferindo da composicao mais simples usada pelo fork `NathanAshford/evolution-go-custom`.

**Approach:** Portar somente a composicao do ramo CTA do fork: enviar `Body`, `Footer` e `NativeFlowMessage` dentro de `DocumentWithCaptionMessage`, mantendo `MessageSecret`, botoes `cta_url`/`cta_call`/`cta_copy` e os nodes `biz/mixed` ja existentes.

## Boundaries & Constraints

**Always:** Preservar o contrato JSON atual, `MessageSecret`, `DocumentWithCaptionMessage`, `MessageVersion=1`, `MessageParamsJSON`, nodes `biz/mixed` e `bot` para conversa individual, suporte a quoted/mentions/delay e os formatos atuais dos parametros de cada CTA.

**Ask First:** Alterar os ramos `reply` ou `pix`, trocar a versao do whatsmeow ou mudar endpoints de lista/carrossel.

**Never:** Copiar o arquivo inteiro do fork, restaurar codigo removido do projeto, misturar botao `reply` com CTA ou depender de uma chamada real ao WhatsApp nos testes automatizados.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| CTAs agrupados | `url` + `call` + `copy` | Payload enxuto igual ao ramo CTA do fork, sem `Header` e sem `ContextInfo` interno vazio | Erros de envio continuam propagados |
| Footer vazio | CTA valido sem footer | Mensagem continua valida sem fabricar texto | N/A |
| Reply ou Pix | Payload existente | Composicao permanece inalterada | Validacoes atuais permanecem |

</frozen-after-approval>

## Code Map

- `pkg/sendMessage/service/send_service.go` -- monta o protobuf e os nodes XMPP de `/send/button`.
- `pkg/sendMessage/service/send_service_test.go` -- testes focados da forma do payload CTA.
- `docs/fork-delta.md` -- registra a decisao de compatibilidade do fork.

## Tasks & Acceptance

**Execution:**
- [x] `pkg/sendMessage/service/send_service.go` -- extrair/montar o payload CTA enxuto e usa-lo apenas quando nao for `reply` nem `pix`.
- [x] `pkg/sendMessage/service/send_service_test.go` -- validar estrutura, botoes, parametros, segredo e ausencia dos campos associados ao payload rejeitado.
- [x] `docs/fork-delta.md` -- registrar o porte seletivo do ramo CTA e sua motivacao pelo erro 473.

**Acceptance Criteria:**
- Given botoes `url`, `call` e `copy`, when o payload e montado, then ele contem os tres NativeFlow buttons, `Body`, `Footer` e `MessageSecret`, sem `InteractiveMessage.Header` nem `InteractiveMessage.ContextInfo` vazio.
- Given um payload `reply` ou `pix`, when enviado pelo endpoint, then seu ramo existente nao e alterado por esta correcao.
- Given o pacote de envio, when os testes focados rodam em Linux, then compilacao e testes passam sem rede real.

## Spec Change Log

## Design Notes

O porte sera seletivo porque CTA/Pix/reply tem contratos diferentes. A evidencia pratica e o erro 473 do servidor; a diferenca estrutural relevante do fork e a remocao de `Header` e `ContextInfo` interno no ramo CTA, mantendo o restante do transporte ja presente localmente.

## Verification

**Commands:**
- `gofmt -w pkg/sendMessage/service/send_service.go pkg/sendMessage/service/send_service_test.go` -- expected: formatacao valida.
- `go test ./pkg/sendMessage/service` em container Go 1.25 -- expected: suite focada aprovada.
- `git diff --check` -- expected: sem erros de whitespace.

**Manual checks:**
- Repetir o curl `url` + `call` + `copy` em homologacao e confirmar que nao retorna 473 e que a mensagem renderiza no destinatario.
