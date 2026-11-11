---
title: 'Identificar tipo da conta conectada'
type: 'feature'
created: '2026-11-11'
status: 'done'
baseline_commit: '687c740b1738965eef1c24d788028c1ab77ab615'
context: []
---

# Identificar tipo da conta conectada

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A API não informa se a instância conectada usa WhatsApp Business ou WhatsApp pessoal, impedindo que consumidores antecipem restrições como a impossibilidade de criar comunidades com contas Business.

**Approach:** Expor `GET /instance/account-type`, autenticado pelo token da instância. Classificar primeiro pelo nome comercial persistido no pareamento e, quando ele estiver vazio, consultar o perfil comercial do próprio JID; distinguir resultado pessoal de falha transitória com os estados `business`, `personal` e `unknown`.

## Boundaries & Constraints

**Always:** Usar a própria conta conectada; remover o device do JID antes da consulta; aplicar timeout; devolver JSON estável com `jid`, `accountType`, `isBusiness` e `businessName`; preservar a autenticação já usada pelas rotas operacionais.

**Ask First:** Qualquer mudança no modelo persistido da instância, no fluxo de pareamento ou na autenticação global.

**Never:** Inferir conta pessoal a partir de timeout/erro de rede; expor dados de outro número; tentar criar uma comunidade como mecanismo de detecção; alterar o comportamento de `/instance/status`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|---------------|----------------------------|----------------|
| Business já identificado | Sessão ativa e `Store.BusinessName` preenchido | `accountType: business`, `isBusiness: true`, nome comercial; sem consulta remota | N/A |
| Business sem nome armazenado | Sessão ativa e perfil comercial disponível | `business`, `true`, nome vazio ou dado disponível | N/A |
| Conta pessoal | Consulta responde sem elemento de perfil comercial | `personal`, `false` | Ausência específica do perfil não vira erro HTTP |
| Consulta inconclusiva | Timeout, desconexão durante IQ ou outro erro remoto | `unknown`, `isBusiness: null` | HTTP 200 para permitir tratamento explícito pelo consumidor |
| Sessão indisponível | Cliente ausente, desconectado, sem login ou JID | Sem classificação | Erro HTTP operacional, sem afirmar `personal` |

</frozen-after-approval>

## Code Map

- `pkg/instance/service/instance_service.go` — contrato, DTO, conexão e classificação da própria conta.
- `pkg/instance/handler/instance_handler.go` — handler HTTP e anotação Swagger.
- `pkg/routes/routes.go` — registro da rota autenticada.
- `pkg/instance/service/instance_service_test.go` — classificação e cenários de erro determinísticos.
- `docs/wiki/guias-api/api-instances.md` — contrato e exemplos de consumo.

## Tasks & Acceptance

**Execution:**
- [ ] `pkg/instance/service/instance_service.go` — adicionar DTO e serviço de detecção com timeout e classificação conservadora.
- [ ] `pkg/instance/handler/instance_handler.go` e `pkg/routes/routes.go` — publicar `GET /instance/account-type` sob `Auth` e mapear respostas.
- [ ] `pkg/instance/service/instance_service_test.go` — cobrir business por cache, business por perfil, personal por ausência específica e unknown por erro transitório.
- [ ] `docs/wiki/guias-api/api-instances.md` — documentar autenticação, resposta e semântica de `unknown`.

**Acceptance Criteria:**
- Given uma instância autenticada e conectada, when `GET /instance/account-type` é chamado, then a resposta contém o JID próprio e exatamente um dos estados `business`, `personal` ou `unknown`.
- Given uma conta Business com nome recebido no pareamento, when a rota é consultada, then nenhuma consulta remota é necessária.
- Given um erro transitório ao consultar o perfil, when a rota responde, then ela retorna `unknown` e nunca `personal`.
- Given token ausente ou inválido, when a rota é chamada, then o middleware existente retorna `401 not authorized`.

## Spec Change Log

## Verification

**Commands:**
- `go test ./pkg/instance/service` — esperado: testes de classificação aprovados.
- `go test ./pkg/instance/... ./pkg/routes/...` — esperado: integração entre contratos compilando e testes aprovados.
- `go test ./...` — esperado: suíte completa aprovada.

