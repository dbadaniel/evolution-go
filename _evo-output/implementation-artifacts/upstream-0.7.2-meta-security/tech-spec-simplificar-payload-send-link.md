---
title: 'Simplificar payload do send link para entrega compativel em grupos'
type: 'bugfix'
created: '2026-08-14'
status: 'done'
baseline_commit: 'd20a50df8be09085f1d2cd509a31271927929a6b'
context: []
---

# Simplificar payload do send link para entrega compativel em grupos

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Mensagens comuns chegam a todos os participantes do grupo, mas mensagens produzidas por `/send/link` apresentam entrega parcial. O payload atual duplica a miniatura em `JPEGThumbnail` e `ExternalAdReply.Thumbnail`, usa contexto de anuncio para uma previa comum e permite miniaturas de ate 600 px.

**Approach:** Aproximar a mensagem do formato simples do projeto original: manter os campos nativos de `ExtendedTextMessage`, limitar e normalizar a miniatura, remover `ExternalAdReply` e preservar os tratamentos locais de timeout, metadata, retry e diagnostico.

## Boundaries & Constraints

**Always:** Preservar a API HTTP existente e os campos aceitos por `LinkStruct`; manter retry de conexao, timeout de download, fallback sem miniatura e logs operacionais; usar somente um campo binario para a miniatura.

**Ask First:** Qualquer mudanca no contrato do endpoint, na biblioteca whatsmeow ou no comportamento de outros tipos de mensagem.

**Never:** Enviar `ExternalAdReply` em uma previa comum; duplicar os bytes da miniatura; alterar arquivos locais alheios ja presentes na arvore de trabalho; realizar envio real ao WhatsApp sem autorizacao e credenciais proprias para o teste.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Link com imagem | URL, texto e metadata com imagem valida | `ExtendedTextMessage` com URL, metadata e uma unica miniatura JPEG limitada | N/A |
| Link sem imagem | URL valida sem imagem ou download/conversao falha | Mensagem de link sem miniatura e sem contexto de anuncio | Registrar aviso e continuar o envio |
| Metadata parcial | Titulo, descricao ou imagem fornecidos pelo cliente | Preservar campos fornecidos e buscar apenas os ausentes | Falha de busca nao deve impedir o envio do texto |

</frozen-after-approval>

## Code Map

- `pkg/sendMessage/service/send_service.go` -- baixa e normaliza a miniatura e monta o `ExtendedTextMessage` usado por `/send/link`.
- `pkg/sendMessage/service/send_service_test.go` -- novos testes unitarios focados na composicao do payload de link.

## Tasks & Acceptance

**Execution:**
- [ ] `pkg/sendMessage/service/send_service.go` -- extrair uma composicao testavel do payload, reduzir a miniatura para no maximo 200 px e remover `ExternalAdReply`, `ThumbnailURL` e `OriginalImageURL`.
- [ ] `pkg/sendMessage/service/send_service_test.go` -- cobrir payload com e sem miniatura e garantir que os bytes nao sejam duplicados em contexto de anuncio.

**Acceptance Criteria:**
- Given uma imagem valida, when `/send/link` monta a mensagem, then o protobuf contem `JPEGThumbnail` uma unica vez, dimensoes coerentes, `PreviewType_IMAGE`, a URL em `MatchedText` e nenhum `ExternalAdReply`.
- Given ausencia ou falha de imagem, when a mensagem e montada, then ela continua sendo um `ExtendedTextMessage` valido sem thumbnail e sem contexto de anuncio.
- Given o codigo existente de conexao e metadata, when a correcao e aplicada, then retry, timeout, fallback e logs permanecem ativos.

## Spec Change Log

- 2026-08-14 — A auditoria constatou que `waE2E.ExtendedTextMessage` da versao fixada do whatsmeow nao possui `CanonicalURL`. O criterio foi corrigido para exigir `MatchedText`, evitando codigo que nao compila. KEEP: payload simples, thumbnail unica de ate 200 px, dimensoes minimas validas, ausencia de `ExternalAdReply`, retry/timeout/logs e testes de normalizacao/fallback.

## Design Notes

O limite de 200 px e um compromisso conservador: reduz substancialmente o payload observado em producao sem eliminar a previa. A composicao em helper puro permite validar o protobuf sem conexao real ao WhatsApp.

## Verification

**Commands:**
- `gofmt -w pkg/sendMessage/service/send_service.go pkg/sendMessage/service/send_service_test.go` -- expected: arquivos formatados.
- `go test ./pkg/sendMessage/service` -- expected: testes do servico aprovados; se o ambiente Windows bloquear dependencias CGO, registrar a limitacao e executar ao menos verificacoes estaticas focadas.
- `git diff --check` -- expected: nenhuma falha de whitespace.

**Manual checks:**
- Publicar a imagem em ambiente de teste e repetir envio com link ao mesmo grupo; confirmar recibos para todos os quatro destinatarios.
