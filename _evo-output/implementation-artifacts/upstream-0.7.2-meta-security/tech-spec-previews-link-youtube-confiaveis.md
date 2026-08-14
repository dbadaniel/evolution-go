---
title: 'Tornar previews de link e YouTube confiaveis'
type: 'bugfix'
created: '2026-08-14'
status: 'done'
baseline_commit: 'b8851f9fbd7dd7de490b3baf992ab36a4f1ee71d'
context: []
---

# Tornar previews de link e YouTube confiaveis

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** O endpoint `/send/link` pode montar `MatchedText` com a propriedade `url` sem que essa URL esteja literalmente presente em `Text`, condicao que impede a previa segundo exemplos validados pela comunidade whatsmeow. Links do YouTube tambem podem retornar a pagina generica e localizada do site, produzindo titulo incorreto, descricao em outro idioma e nenhuma thumbnail.

**Approach:** Normalizar texto e URL antes de montar a mensagem, acrescentando a URL explicita ao texto somente quando ele nao contiver nenhuma correspondencia adequada. Para hosts YouTube reconhecidos com seguranca, obter titulo, autor e thumbnail pelo endpoint publico oEmbed e manter o extrator HTML atual como fallback.

## Boundaries & Constraints

**Always:** Manter `MatchedText` literalmente contido no `Text` enviado; preservar a URL ja presente sem duplica-la; preservar metadata fornecida pelo consumidor; manter timeout, retry, logs, fallback generico, thumbnail JPEG inline unica de ate 200 px e o payload simples validado no commit `b8851f9`.

**Ask First:** Alterar o contrato JSON do endpoint, adicionar dependencia externa ao `go.mod`, usar upload de thumbnail para servidores WhatsApp ou mudar outros endpoints de mensagem.

**Never:** Restaurar `ExternalAdReply`; usar `ThumbnailDirectPath`, `MediaKey` ou hashes de upload; fabricar preview grande ou player incorporado; aceitar dominios como `youtube.com.evil.example` como YouTube; depender da rede real nos testes automatizados.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| URL ja no texto | `text` contem a mesma `url` explicita | Texto permanece identico e `MatchedText` usa essa URL | N/A |
| URL somente no campo | `url` valida ausente de `text` | Acrescentar a URL uma unica vez em nova linha e usa-la como `MatchedText` | N/A |
| URL somente no texto | `url` vazia e `text` contem link | Usar o primeiro link encontrado sem modificar o texto | N/A |
| YouTube valido | Host `youtube.com`, subdominio seguro ou `youtu.be` | Usar titulo, autor e thumbnail retornados pelo oEmbed | Resposta invalida, parcial ou nao-2xx cai no extrator HTML |
| Site comum | URL nao-YouTube | Usar apenas o extrator HTML atual | Manter comportamento atual |
| Host semelhante malicioso | `youtube.com.evil.example` | Nao chamar oEmbed do YouTube | Tratar como site comum |

</frozen-after-approval>

## Code Map

- `pkg/sendMessage/service/send_service.go` -- resolve texto/MatchedText, busca metadata e monta o payload simples.
- `pkg/sendMessage/service/send_service_test.go` -- cobre invariantes do payload, resolucao da URL e metadata YouTube sem rede real.

## Tasks & Acceptance

**Execution:**
- [x] `pkg/sendMessage/service/send_service.go` -- adicionar helper puro para resolver texto e URL; detectar hosts YouTube; consultar oEmbed com cliente HTTP injetavel e limite de resposta; aplicar fallback ao scraper HTML existente sem mudar a precedencia dos campos informados.
- [x] `pkg/sendMessage/service/send_service_test.go` -- testar URL presente/ausente, classificacao segura de hosts, oEmbed bem-sucedido, fallback em erro/JSON parcial, site comum e preservacao do payload simples.

**Acceptance Criteria:**
- Given qualquer mensagem produzida por `/send/link` com `MatchedText` nao vazio, when o protobuf e montado, then `Text` contem literalmente `MatchedText`.
- Given uma live publica do YouTube com oEmbed disponivel, when a metadata e buscada, then titulo, autor e thumbnail especificos substituem a pagina generica, sem adicionar campos remotos ao protobuf.
- Given falha no oEmbed, when a pagina possui OpenGraph valido, then o extrator generico fornece a metadata e o envio continua.
- Given o payload final, when serializado, then nao existe `ExternalAdReply` e a thumbnail inline nao e duplicada.

## Spec Change Log

## Design Notes

O oEmbed fornece `title`, `author_name` e `thumbnail_url`, mas nao uma descricao editorial. `author_name` sera usado como descricao curta. A URL explicita sera anexada somente quando estiver ausente do texto, preservando compatibilidade com clientes que enviam `text` e `url` separadamente.

## Verification

**Commands:**
- `gofmt -w pkg/sendMessage/service/send_service.go pkg/sendMessage/service/send_service_test.go` -- expected: formatacao Go valida.
- `docker build --target build --build-arg VERSION=test -t evolution-go:link-metadata-test .` -- expected: compilacao Linux/CGO aprovada.
- `docker run --rm --entrypoint go -w /build evolution-go:link-metadata-test test ./pkg/sendMessage/service` -- expected: suite aprovada sem acessar YouTube real.
- `git diff --check` -- expected: sem erros de whitespace.

**Manual checks:**
- Apos deploy em teste, enviar a mesma URL de live ao grupo e confirmar titulo/thumbnail corretos e recibos de todos os destinatarios.
