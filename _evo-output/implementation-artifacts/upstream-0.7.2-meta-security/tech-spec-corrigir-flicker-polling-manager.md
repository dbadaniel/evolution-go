---
title: 'Corrigir flicker no polling da tela de instancias do Manager'
type: 'bugfix'
created: '2026-08-13'
status: 'done'
---

# Objetivo

Reaplicar no bundle distribuido do Manager a correção conhecida do commit
`f79c5c4`, impedindo que o polling periodico substitua a lista de instancias
por skeletons depois da primeira carga.

# Escopo

- Restaurar exatamente o patch de `f79c5c4` no bundle
  `manager/dist/assets/index-LD46dRLh.js`.
- Manter a marcacao de bundles gerados em `.gitattributes` trazida pelo mesmo
  commit.
- Nao alterar `.gitignore`, configuracoes locais ou o bundle de backup.

# Criterios de aceitacao

- Given que a tela de instancias ja concluiu a primeira carga, when o polling
  de cinco segundos consultar a API novamente, then a lista atual permanece
  renderizada enquanto os dados sao atualizados.
- Given que a tela ainda nao carregou dados pela primeira vez, when a consulta
  inicial estiver pendente, then o estado de loading continua podendo exibir os
  skeletons.
- Given o bundle corrigido, when comparado com o artefato do commit `f79c5c4`,
  then seu conteudo e identico.

# Revisao adversarial

- A equivalencia funcional foi confirmada: o blob do bundle corrigido e o de
  `f79c5c4` possuem o mesmo objeto Git (`4db549592cd1b05df1f23ae4feb64624b1b33551`).
- Corrigido um risco encontrado na revisao: como o conteudo havia mudado sem
  mudar o nome hashado, caches poderiam continuar entregando o bundle antigo.
  O arquivo passou a se chamar `index-e340ef6a.js`, e o `index.html` referencia
  esse novo nome.
- `node --check manager/dist/assets/index-e340ef6a.js` passou.
- A ausencia do codigo-fonte do Manager neste repositorio foi registrada como
  trabalho futuro, pois uma recompilacao externa ainda pode reintroduzir a
  regressao.
