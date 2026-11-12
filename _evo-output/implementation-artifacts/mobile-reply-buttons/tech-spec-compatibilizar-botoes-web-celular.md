---
title: 'Compatibilizar botões de resposta no Web e celular'
type: 'bugfix'
created: '2026-11-12'
status: 'in-review'
baseline_commit: '613b551b09cb818215db128c727706ad8abfd20f'
context:
  - 'docs/fork-delta.md'
---

# Compatibilizar botões de resposta no Web e celular

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O primeiro ajuste de `reply` confirmou em aparelho real que `InteractiveMessage` com `quick_reply` e o nó `<biz>` completo chega e renderiza no celular, mas a mesma mensagem não aparece no WhatsApp Web. O formato legado fazia o inverso, portanto nenhum dos envelopes atuais atende simultaneamente os dois clientes.

**Approach:** Preservar o protobuf native-flow e o nó de transporte já aceitos pelo celular, envolvendo o conteúdo em `ViewOnceMessage` com `MessageContextInfo`, `MessageSecret`, `DeviceListMetadata` e `DeviceListMetadataVersion=2` no conteúdo interno. Esse é o envelope amplamente usado para mensagens interativas multidispositivo e continua enviando exatamente uma mensagem.

## Boundaries & Constraints

**Always:** Preservar os botões `quick_reply`, IDs, textos, corpo, rodapé, citação, menções e o `<biz>` completo validado no celular; manter um único envio; colocar os metadados no mesmo nível protobuf do `InteractiveMessage`; atualizar helpers de unwrapping e testes para o novo envelope.

**Ask First:** Alterar novamente o nó `<biz>`; enviar dois formatos; criar fallback automático; mudar o payload HTTP ou o evento de clique; alterar CTA, Pix ou reply com mídia.

**Never:** Voltar ao `ButtonsMessage` legado para replies textuais; afirmar compatibilidade Web + móvel antes do novo teste real; duplicar a mensagem para tentar alcançar clientes diferentes; afetar o backup ou a imagem Docker atual durante a implementação.

## I/O & Edge-Case Matrix

| Cenário | Entrada / estado | Saída / comportamento esperado | Tratamento de erro |
|---|---|---|---|
| Sim/Não textual | Dois replies sem mídia | Um `ViewOnceMessage` contendo `InteractiveMessage` native-flow | Erro do servidor propagado sem retry |
| Cliente móvel | Android/iOS atualizado | Mensagem e botões continuam visíveis | Reprovação manual bloqueia conclusão |
| Cliente Web | WhatsApp Web atualizado | A mesma mensagem e botões ficam visíveis | Reprovação manual exige nova investigação |
| Reply citado/mencionado | `quoted` ou menções presentes | `ContextInfo` interno preserva todos os campos | Não substituir campos já definidos |
| CTA/Pix/mídia | Tipos fora do reply textual | Caminhos atuais permanecem inalterados | Comportamento anterior preservado |

</frozen-after-approval>

## Code Map

- `pkg/sendMessage/service/send_service.go` — construção e unwrapping do envelope interativo.
- `pkg/sendMessage/service/send_service_test.go` — assertions do wrapper, metadados, citação e regressões.
- `docs/fork-delta.md` — resultado real do primeiro teste e novo envelope multidispositivo.

## Tasks & Acceptance

**Execution:**
- [x] `pkg/sendMessage/service/send_service.go` — envolver somente replies textuais modernos em `ViewOnceMessage`, mover metadados para a mensagem interna e tornar helpers compatíveis com o wrapper.
- [x] `pkg/sendMessage/service/send_service_test.go` — substituir expectativa top-level, validar versão 2, segredo, native-flow, quote/mentions e ausência de envelope legado.
- [x] `docs/fork-delta.md` — registrar que top-level funcionou no móvel e falhou no Web, documentando a segunda variante.

**Acceptance Criteria:**
- Given um reply textual, when o protobuf é construído, then existe exatamente um `ViewOnceMessage` com `InteractiveMessage`, metadados versão 2 e nenhum `ButtonsMessage` ou `DocumentWithCaptionMessage`.
- Given citação ou menções, when o envio aplica `ContextInfo`, then os campos são anexados ao `InteractiveMessage` interno sem remover `MessageSecret` nem metadados.
- Given CTA, Pix ou reply com mídia, when composto, then o diff não muda seus construtores ou seleção de caminho.
- Given o stage local atualizado, when o mesmo curl é enviado, then uma única mensagem renderiza no celular e no WhatsApp Web e o clique conserva `sim`/`nao`.

## Spec Change Log

## Design Notes

O `ViewOnceMessage` aqui é apenas o envelope future-proof usado para transportar conteúdo interativo; não transforma o conteúdo em mídia de visualização única. O nó `<biz>` continua externo ao protobuf em `AdditionalNodes`.

## Verification

**Commands:**
- `gofmt -w pkg/sendMessage/service/send_service.go pkg/sendMessage/service/send_service_test.go` — esperado: arquivos formatados.
- `go test ./pkg/sendMessage/service` — esperado: testes aprovados em Go 1.26 + CGO.
- `go test ./pkg/sendMessage/...` — esperado: pacote completo aprovado.
- `git diff --check` — esperado: sem erros de whitespace.

**Manual checks:**
- Recompilar somente o binário do stage local, substituir com backup, enviar uma vez e confirmar a mesma mensagem no celular e Web antes de publicar imagem.

