---
title: 'Adicionar publicacao Docker explicita no Makefile'
type: 'feature'
created: '2026-08-13'
status: 'done'
---

# Objetivo

Adicionar um alvo `docker-publish` que construa, versione e publique a imagem
Docker em `dbadaniel/expertsa`, sem mudar o comportamento de `docker-build`.

# Escopo

- Adicionar variaveis sobrescritiveis para repositorio e tag da imagem.
- Validar versao, repositorio e tag antes de iniciar o build e proteger as
  expansoes contra metacaracteres de shell.
- Derivar por padrao `evolution-go7.2` da versao atual `v0.7.2`.
- Fazer `docker-publish` depender de `docker-build`, executar `docker tag` e
  depois `docker push`.
- Nao executar push durante a implementacao ou validacao.

# Criterios de aceitacao

- Given `VERSION=v0.7.2`, when `make docker-publish` for avaliado, then a imagem
  publicada e `dbadaniel/expertsa:evolution-go7.2`.
- Given que o usuario executa somente `make docker-build`, when o build termina,
  then nenhum `docker push` e realizado.
- Given valores personalizados de `DOCKER_REPOSITORY` ou `DOCKER_TAG`, when o
  alvo e executado, then esses valores substituem os defaults.
- Given versao ausente/malformada ou overrides vazios/invalidos, when a
  publicacao for solicitada, then o alvo falha antes do build sem executar o
  conteudo fornecido como comando de shell.
- Given a publicacao, when os comandos forem executados, then o build ocorre
  antes do tag e o tag ocorre antes do push.

# Revisao adversarial

- A revisao identificou risco de injecao por overrides contendo metacaracteres;
  as expansoes agora usam quoting seguro e valores sao validados antes do build.
- A extracao de `VERSION` foi restringida a SemVer no cabecalho do changelog.
- Overrides vazios, tag invalida, repositorio invalido e versao ausente falham
  antecipadamente.
- `make -n docker-publish` confirmou a ordem validacao, build, tag e push e a
  imagem default `dbadaniel/expertsa:evolution-go7.2`.
- `make -n docker-build` confirmou ausencia de tag e push.
- Casos validos e invalidos de `docker-publish-validate` foram executados no
  WSL sem chamar Docker nem publicar imagens.
