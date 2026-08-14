---
title: 'Remover node bot dos botoes CTA rejeitados com 473'
type: 'bugfix'
created: '2026-08-14'
status: 'done'
baseline_commit: '2851555'
context:
  - 'docs/fork-delta.md'
---

# Remover node bot dos botoes CTA rejeitados com 473

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Mesmo com o protobuf CTA equivalente ao fork, um unico botao `url` continua sendo rejeitado pelo servidor WhatsApp com erro 473. O log confirma imagem nova, instancia conectada, destinatario valido e rejeicao durante o envio.

**Approach:** Manter o node `biz/native_flow/mixed` exigido pelo CTA e deixar de declarar `bot biz_bot=1` nesse ramo, evitando solicitar uma capacidade de bot que a conta vinculada pode nao possuir.

## Boundaries & Constraints

**Always:** Alterar apenas os nodes adicionais de CTA; manter protobuf, MessageSecret, params JSON, quoted, mentions, delay e node `biz/mixed` atuais.

**Ask First:** Remover tambem o node `biz`, modificar tokens de privacidade, alterar whatsmeow ou exigir novo pareamento da instancia.

**Never:** Mudar os nodes de Pix, reply, lista ou carrossel; enviar mensagens extras invisiveis; mascarar o erro 473 como sucesso.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| CTA em conversa individual | Botao `url`, `call` ou `copy` | Enviar `biz/native_flow/mixed` sem node `bot` | Propagar erro real do servidor |
| Pix individual | Botao `pix` | Preservar nodes `biz/payment_info` e `bot` existentes | Comportamento atual |
| CTA em grupo | Destino `@g.us` | Permanecer com `biz/mixed`, como ja ocorre | Comportamento atual |

</frozen-after-approval>

## Code Map

- `pkg/sendMessage/service/send_service.go` -- seleciona nodes XMPP adicionais de botoes.
- `pkg/sendMessage/service/send_service_test.go` -- valida a composicao dos nodes sem rede real.
- `docs/fork-delta.md` -- registra a excecao por capacidade da conta.

## Tasks & Acceptance

**Execution:**
- [x] `pkg/sendMessage/service/send_service.go` -- extrair a composicao de nodes e omitir `bot` somente para CTA.
- [x] `pkg/sendMessage/service/send_service_test.go` -- cobrir CTA individual/grupo e Pix individual.
- [x] `docs/fork-delta.md` -- documentar a decisao baseada no teste real com URL unica.

**Acceptance Criteria:**
- Given um CTA individual, when os nodes sao montados, then existe `biz/native_flow/mixed` e nao existe `bot`.
- Given um Pix individual, when os nodes sao montados, then o node `bot biz_bot=1` permanece.
- Given o pacote de envio, when o teste focado roda, then ele passa sem conexao real ao WhatsApp.

## Spec Change Log

## Design Notes

O erro 473 persistiu tanto no conjunto de CTAs quanto em URL isolada. Como o protobuf ja coincide com o fork, o node `bot` e o proximo diferencial de capacidade removivel sem desmontar a identificacao `biz/native_flow` da mensagem.

## Verification

**Commands:**
- `gofmt -w pkg/sendMessage/service/send_service.go pkg/sendMessage/service/send_service_test.go` -- expected: formatacao valida.
- `go test ./pkg/sendMessage/service` em Docker Go 1.25 -- expected: suite focada aprovada.
- `git diff --check` -- expected: sem erros de whitespace.

**Manual checks:**
- Repetir o envio de um unico botao URL em homologacao; confirmar ausencia de 473 e renderizacao no destino.
