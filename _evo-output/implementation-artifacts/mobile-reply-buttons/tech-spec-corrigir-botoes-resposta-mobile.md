---
title: 'Entregar botões de resposta no WhatsApp móvel'
type: 'bugfix'
created: '2026-11-12'
status: 'done'
baseline_commit: '5ec7ec8048de9d9ec83e3ffb89b383b21cb596cb'
context:
  - 'docs/fork-delta.md'
---

# Entregar botões de resposta no WhatsApp móvel

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Mensagens `POST /send/button` contendo apenas botões `reply` são aceitas e aparecem no WhatsApp Web, mas não chegam ou não renderizam nos clientes móveis. O caminho atual usa o `ButtonsMessage` legado dentro de `DocumentWithCaptionMessage`, formato que o servidor e os clientes atuais tratam de maneira inconsistente.

**Approach:** Para botões de resposta textuais, enviar `InteractiveMessage` de nível superior com `NativeFlowMessage` e botões `quick_reply`, acompanhado do nó `<biz>` completo observado no cliente oficial e validado em dispositivo real no ecossistema `whatsmeow`. Manter CTA, Pix e o contrato HTTP existentes sem alteração.

## Boundaries & Constraints

**Always:** Preservar `id` e `displayText` de cada opção; manter título, descrição, rodapé, citação, menções, delay e `MessageSecret`; usar aleatoriedade criptográfica para `decision_id`; gerar um único nó `<biz>` sem `<bot>` no caminho `quick_reply`; continuar aceitando no máximo três respostas.

**Ask First:** Alterar o payload público de `/send/button`; remover suporte a cabeçalhos `imageUrl`/`videoUrl`; modificar CTA, Pix ou o formato dos eventos recebidos; adicionar fallback que possa duplicar mensagens.

**Never:** Reenviar automaticamente após aceite do servidor; usar o legado `ButtonsMessage` para respostas textuais sem mídia; enviar token, número real ou outro segredo em testes; afirmar compatibilidade definitiva sem teste em aparelho real.

## I/O & Edge-Case Matrix

| Cenário | Entrada / estado | Saída / comportamento esperado | Tratamento de erro |
|---|---|---|---|
| Sim/Não textual | 2 botões `reply`, sem mídia | `InteractiveMessage` superior com dois `quick_reply`, visível em Web e móvel | Erro do servidor continua propagado |
| Limites de resposta | 1 a 3 botões `reply` | IDs e textos preservados em JSON válido | Mais de 3 mantém erro atual |
| Reply com mídia | `imageUrl` ou `videoUrl` | Caminho legado preservado até existir solução específica para mídia | Não descartar mídia silenciosamente |
| CTA ou Pix | `url`, `call`, `copy` ou `pix` | Protobuf e nós de transporte atuais preservados | Validações atuais preservadas |
| Resposta citada | `quoted.messageId` preenchido | `ContextInfo` aplicado ao `InteractiveMessage` sem perder `MessageSecret` | Erro atual preservado |

</frozen-after-approval>

## Code Map

- `pkg/sendMessage/service/send_service.go` — composição dos botões, envelope protobuf, nós adicionais e envio.
- `pkg/sendMessage/service/send_service_test.go` — testes de formato, regressão de CTA/Pix e invariantes do nó `<biz>`.
- `docs/fork-delta.md` — registro da divergência intencional do fork e da limitação de mídia.

## Tasks & Acceptance

**Execution:**
- [x] `pkg/sendMessage/service/send_service.go` — extrair construtores determinísticos para o `InteractiveMessage` de resposta e nó `<biz>` completo; selecionar esse caminho apenas para replies sem mídia.
- [x] `pkg/sendMessage/service/send_service_test.go` — cobrir shape moderno, JSON escapado, título/descrição/rodapé, segredo, citação e atributos do nó; provar que CTA/Pix permanecem no caminho anterior.
- [x] `docs/fork-delta.md` — substituir a observação de falha conhecida pela estratégia atual e registrar que reply com mídia permanece legado.

**Acceptance Criteria:**
- Given uma requisição de botões “Sim” e “Não” sem mídia, when a mensagem é composta, then ela usa `InteractiveMessage` de nível superior, dois `quick_reply` e nenhum `ButtonsMessage`, `DocumentWithCaptionMessage` ou nó `<bot>`.
- Given o nó de transporte da resposta rápida, when ele é inspecionado, then contém `actual_actors=2`, `host_storage=2`, timestamp Unix, `native_flow v=9 name=mixed` e `quality_control` com `decision_id` hexadecimal de 40 caracteres.
- Given CTA, Pix ou reply com mídia, when a mensagem é composta, then o comportamento anterior permanece disponível sem perda silenciosa de dados.
- Given o conjunto de testes do serviço, when executado, then os testes novos e existentes passam.

## Spec Change Log

## Design Notes

O nó `<biz>` moderno segue as medições documentadas em `tulir/whatsmeow#1235`: sem ele o servidor aceita mas nenhum cliente renderiza; o nó legado recebe `405`; o nó parcial recebe `479`. A mudança fica localizada no caminho reply textual para reduzir regressões.

## Verification

**Commands:**
- `gofmt -w pkg/sendMessage/service/send_service.go pkg/sendMessage/service/send_service_test.go` — esperado: arquivos formatados.
- `go test ./pkg/sendMessage/service` — esperado: testes do serviço aprovados.
- `go test ./pkg/sendMessage/...` — esperado: pacote completo compilando e testes aprovados.
- `git diff --check` — esperado: diff sem erros de whitespace.

**Manual checks:**
- Enviar Sim/Não para Android ou iOS e confirmar renderização e retorno do `ButtonClick` com o ID correto; repetir no WhatsApp Web.

