# Estratégia de Versionamento e Gestão de Fork (EvolutionAPI)

Este documento descreve a arquitetura de versionamento Git adotada para o projeto \`evolution-go\`. O objetivo principal é permitir a aplicação de correções locais (`hotfixes` ou `features` customizadas) rapidamente, mantendo o repositório 100% sincronizado com as atualizações oficias do projeto original (upstream).

## A Arquitetura do "Rio Paralelo"

A estratégia baseia-se em três pilares principais de branches:

1.  **\`main\` (A Base Espelho):**
    Esta branch deve ser um espelho exato da branch `main` do repositório original (EvolutionAPI).
    *   **Regra de Ouro:** NUNCA faça commits diretos na `main`. Ela serve apenas para buscar as novidades do `upstream`.

2.  **\`fix/*\` (As Ilhas de Correção):**
    Para cada ajuste, correção de bug ou feature customizada, uma nova branch deve nascer a partir da `main` limpa.
    *   Exemplo: `fix/auth-grupos`, `fix/preview`.
    *   *Ciclo de Vida:* Quando a equipe oficial do EvolutionAPI resolver o problema e a correção chegar na `main` através do `upstream`, a branch `fix/` correspondente deve ser deletada no repositório local.

3.  **\`custom-main\` (O Ponto de Encontro/Produção):**
    É a branch utilizada para testes finais, deploy e execução.
    *   *Como é formada:* Ela é criada descartando a versão anterior, nascendo novamente a partir da `main` atualizada, e então recebendo o merge das branches `fix/*` ativas.

---

## O Fluxo de Trabalho Automático (Makefile)

Para evitar erros manuais e garantir a sanidade da árvore do Git, o fluxo foi automatizado no `Makefile`. 

### Comandos Disponíveis

*   **`make setup-upstream`**: Configura o repositório remoto oficial (caso ainda não exista).
*   **`make sync-main`**: Baixa as últimas atualizações do repositório oficial e atualiza a branch `main` local.
*   **`make build-custom`**: É o motor principal. Ele sincroniza a `main`, deleta a `custom-main` antiga, cria uma nova a partir do oficial atualizado, e aplica as correções (`fix/*`) por cima.

### Exemplo de Trecho para o \`Makefile\`

\`\`\`makefile
# Define o endereço do projeto pai (Upstream)
UPSTREAM_URL := https://github.com/EvolutionAPI/evolution-go.git

setup-upstream:
	@git remote get-url upstream >/dev/null 2>&1 || git remote add upstream $(UPSTREAM_URL)
	@echo "Upstream configurado com sucesso."

sync-main: setup-upstream
	@echo "Sincronizando main local com o upstream oficial..."
	git fetch upstream
	git checkout main
	git pull upstream main --rebase

build-custom: sync-main
	@echo "Recriando a branch custom-main limpa..."
	git checkout main
	git branch -D custom-main || true
	git checkout -b custom-main
	@echo "Aplicando as correções locais (fix/*)..."
	-git merge fix/auth-grupos --no-edit
	-git merge fix/preview --no-edit
	@echo "Branch custom-main gerada com sucesso e pronta para uso!"
\`\`\`

## Instruções para os Agentes de IA

Sempre que a IA (seja o analista, o arquiteto ou o desenvolvedor) precisar atuar em uma mudança de código neste projeto:
1. Verifique se a mudança é uma correção que deveria estar no projeto original.
2. Se sim, crie uma branch `fix/nome-da-correcao` partindo da `main`.
3. Aplique e teste a correção nessa branch.
4. Adicione essa nova branch ao comando `build-custom` do Makefile para que ela entre no ciclo de build oficial.
5. Lembre o desenvolvedor/usuário de deletar a branch quando a EvolutionAPI integrar a funcionalidade nas releases futuras.
