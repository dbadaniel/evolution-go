---
title: 'Corrigir listas no celular e Web com transporte explícito whatsmeow'
type: 'bugfix'
created: '2026-11-12'
status: 'done'
baseline_commit: '40898632c41b34ec73fd717de8500d391df2bd0e'
context: []
---

# Corrigir listas no celular e Web

## Intent

`/send/list` retornava ACK200 sem entrega. A variante ListMessage top-level com os metadados automáticos da biblioteca era rejeitada com 405. O usuário confirmou reply, URL e Pix funcionando e autorizou corrigir a lista na branch local `fix/list-message-delivery`, com testes na stage Docker.

O formato validado é ListMessage SINGLE_SELECT top-level, MessageSecret/metadados de dispositivo v2, transporte text e exatamente um biz/list product_list com metadados quality_control e nenhum bot. O patch de biblioteca restringe-se à classificação da lista e à supressão do biz automático quando o app passa biz/list explícito. Manter uma cópia da mesma revisão oficial já fixada, com os dois ajustes, torna o resultado reproduzível em builds locais e Docker.

## Boundaries

- Preservar literalmente os builders/transportes de reply, CTA e Pix, SendButton e SendMessage. Preservar as alterações autorizadas já existentes na árvore suja.
- Preservar contrato JSON público: number, title, description, footerText, buttonText, sections/rows/rowId, id, delay, mentions, quoted.
- Não atualizar a revisão do whatsmeow, alterar pareamento/conexões, criar fork remoto, fazer merge ou push nesta etapa.
- Manter licença e origem da cópia em third_party/whatsmeow. Nenhuma modificação global no cache Go do host.
- Diferenciar ACK, receipt e confirmação visual. Não declarar iOS validado.

## Code Map

- `pkg/sendMessage/service/send_service.go`: normalização, validação, builder e envio de lista; mesmo helper de metadados de reply, com filho list/product_list.
- `pkg/sendMessage/handler/send_handler.go`: erros estruturais/limites de lista devolvem HTTP400; campos exigidos existentes mantidos.
- `pkg/sendMessage/service/send_service_test.go`: payload, Unicode, IDs e limites; testes de reply/CTA/Pix preservados.
- `third_party/whatsmeow/send.go`: ListMessage é text; suprimir auto-biz apenas para lista top-level que já tenha biz/list explícito.
- `third_party/whatsmeow/expertsa_list_transport_test.go`: um único biz explícito e outros tipos inalterados.
- `go.mod`, `Dockerfile`: seleção explícita da cópia local fixada, incluindo-a antes de go mod download.
- `patches/whatsmeow-list-transport.patch`, `third_party/whatsmeow/EXPERTSA_PATCH.md`: delta auditável e origem exata.
- README e guias de mensagens: formato realmente validado, limites e contrato preservado.

## Acceptance

Given um payload válido de lista, when enviado, then há ListMessage SINGLE_SELECT top-level, textos/IDs preservados, nenhum envelope ou bot e só um biz/list product_list.

Given opções fora de 1–10 seções/10 opções totais, seção vazia ou ID explícito duplicado, when processadas, then HTTP400 ocorre antes do envio. IDs ausentes são globais e não colidem com IDs explícitos.

Given uma mensagem de outro tipo, when enviada com a mesma biblioteca, then o patch não altera sua classificação ou nós.

Given build local ou Docker a partir da branch, when compilado, then a revisão oficial fixada com os dois ajustes é usada, sem depender de alterações temporárias no container/cache.

Given o teste real identificado, when aberto no celular e Web, then o menu aparece e abre as duas opções. Este critério foi confirmado pelo usuário.

## Validation Evidence

- Três revisões independentes do caminho de lista não encontraram bloqueadores. Comparação literal preservou 11 funções protegidas de reply/CTA/Pix/despacho.
- Native-flow single_select com bot: ACK200 sem entrega. Native-flow com transporte QC sem bot: receipts de entrega e confirmação de menu no celular, mas mensagem incompatível no Web.
- Experimento legado com patch de biblioteca: `3EB0062394260D407B3AD9`, ACK200 às 16:48:15; receipts dos dispositivos do destinatário às 16:48:16; usuário confirmou menu funcionando nos dois clientes.
- Teste da biblioteca executado com descoberta do arquivo confirmada (sem aceitar saída “no tests to run”). Payload legado, validação e regressões de reply/CTA/Pix passaram no builder Linux.
- Reprodução final concluída: imagem `evolution-go:list-web-mobile-local` construída pelo Dockerfile atualizado. Testes de serviço/handler e `TestExpertsaExplicitListTransport` passaram, com seleção de módulo comprovada em `/build/third_party/whatsmeow`.
- A cópia local mantém todos os 256 arquivos oficiais da revisão fixada, somente send.go alterado e dois arquivos adicionados (documentação do patch e teste). As três revisões finais não encontraram bloqueadores; o serviço e o send.go local são idênticos às cópias validadas no experimento.
- Stage atualizada com o binário extraído dessa imagem, versão `list-web-mobile-final`, às 16:57 de 12/11/2026. Containers temporários removidos; binários de rollback preservados fora do repositório. Sem merge ou push.

## Sources

- https://github.com/tulir/whatsmeow/pull/1221 — trata respostas a opções, não o menu.
- https://github.com/tulir/whatsmeow/pull/1146 — transporte explícito e metadados.
- https://github.com/wppconnect-team/wa-js/blob/main/src/chat/functions/sendListMessage.ts — classificação text.
- https://github.com/evolution-foundation/evolution-api/releases — ListMessage/product_list para Web em lugar do single_select moderno.
