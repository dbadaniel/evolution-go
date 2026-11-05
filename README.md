<div align="center">
  <img src="./Exsa-azul.png" alt="Expertsa" width="480"/>
  <h1>Evolution Go — Expertsa Edition</h1>
  <p><strong>Fork de produção focado em solidez, estabilidade de sessão e entrega contínua de eventos para o ecossistema Evolution Go</strong></p>

[![Docker Image](https://img.shields.io/badge/Docker-dbadaniel%2Fevogo:0.7.3-2496ED?logo=docker&logoColor=white)](https://hub.docker.com/r/dbadaniel/evogo)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![whatsmeow](https://img.shields.io/badge/whatsmeow-Nov%2F2026%20Official-25D366?logo=whatsapp&logoColor=white)](https://github.com/tulir/whatsmeow)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](./LICENSE)
[![Upstream](https://img.shields.io/badge/Upstream-EvolutionAPI%2Fevolution--go-orange)](https://github.com/EvolutionAPI/evolution-go)
</div>

---

## Sobre o Projeto

O **Evolution Go** é uma API de WhatsApp de alta performance construída em Go, desenvolvida originalmente pela **[Evolution Foundation](https://evolutionfoundation.com.br/)** ([EvolutionAPI/evolution-go](https://github.com/EvolutionAPI/evolution-go)) utilizando a biblioteca [whatsmeow](https://github.com/tulir/whatsmeow).

Esta distribuição — mantida pela **[Expertsa](https://expertsa.com.br)** na imagem Docker **`dbadaniel/evogo`** — nasce com um objetivo claro: **entregar solidez, previsibilidade e confiabilidade de nível produção no melhor fork da Evolution Go**. Em vez de apenas replicar código upstream ou manter submódulos congelados no tempo, a Expertsa realiza curadoria técnica rigorosa, testes reais de protocolo e correções profundas de concorrência para garantir que chatbots, integrações com n8n, CRMs e operações críticas não parem de funcionar.

---

## 🚀 Por que este fork é diferente? (Qualidade Expertsa)

Ambientes de produção reais sofrem com oscilações de rede, mudanças silenciosas no protocolo da Meta/WhatsApp e restarts de containers. A imagem **`dbadaniel/evogo`** resolve na raiz os gargalos mais críticos da operação diária:

### 1. Protocolo Oficial `whatsmeow` Atualizado (Nov/2026 + Go 1.26)
- **Fim do erro `405 Client outdated`:** Substituímos o antigo submódulo estático pela biblioteca oficial `go.mau.fi/whatsmeow` atualizada (`v0.0.0-20261102152715-54038813c47f`, compilada em **Go 1.26**).
- **Handshake resiliente:** Inclui as correções oficiais de protocolo de final de outubro/novembro de 2026 (interrupção imediata quando o socket fecha durante o handshake, auto-reconexão quando o handshake não recebe resposta e protobufs atualizados `v1052556849`), além de suporte a override de versão via variáveis de ambiente (`WHATSAPP_VERSION_MAJOR`, `WHATSAPP_VERSION_MINOR`, `WHATSAPP_VERSION_PATCH`) e cache inteligente da revisão do WhatsApp Web.

### 2. Entrega Contínua do Evento `Message` no Webhook e Auto-Reconnect Nativo
- **Zero perda de mensagens após minutos/horas de operação:** Corrigimos a condição de corrida entre chamadas concorrentes de `POST /instance/connect` e `GET /instance/qr` que instanciavam múltiplos clientes/rotinas de QR para a mesma sessão e derrubavam a conexão recém-autenticada.
- **Recuperação nativa de KeepAlive:** Ativamos o `EnableAutoReconnect` e `InitialAutoReconnect` nativos do `whatsmeow` e eliminamos o recriamento destrutivo do cliente em timeouts transitórios de rede, mantendo os handlers de eventos e a sessão Signal íntegros.
- **Canais de encerramento determinísticos:** Substituímos o antigo `killChannel` global sem buffer por canais dedicados por cliente (`stopCh` e `qrStopCh`), evitando que rotinas de presença (`schedulePresenceUpdates`) consumissem sinais de encerramento por engano.
- **Preservação de Webhook e Eventos:** Reconexões e chamadas ao `/instance/connect` sem payload não apagam mais a URL de webhook nem a lista de eventos assinados pela instância.

### 3. Restauração Automática de Sessões no Restart do Container
- **Persistência real de autenticação:** Quando `POSTGRES_AUTH_DB` não é declarado separadamente no `.env`, a aplicação faz fallback automático para `POSTGRES_USERS_DB` (ou `POSTGRES_HOST`/`POSTGRES_DB`), garantindo que as chaves criptográficas do `whatsmeow` sejam persistidas no PostgreSQL.
- **`CONNECT_ON_STARTUP` inteligente:** Por padrão (`CONNECT_ON_STARTUP=true`), ao reiniciar o container todas as instâncias pareadas (`jid != ''`) que não foram desconectadas manualmente via API reconectam sozinhas sem necessidade de ler o QR Code novamente — respeitando integralmente `CONNECT_ON_STARTUP=false` caso você deseje desativar a reconexão no boot.

### 4. Mensagens Interativas Multiplataforma (Android, iOS e WhatsApp Web)
- **Botões (`POST /send/button`), Listas (`POST /send/list`) e Pix:** Reconciliação cuidadosa dos nós de transporte (`biz`, `native_flow`, `product_list`) e `DocumentWithCaptionMessage` para máxima compatibilidade.
- **Correção do erro `473` em botões CTA (`url`, `call`, `copy`):** Payload enxuto sem nós `bot` incompatíveis com contas padrão vinculadas.
- **Carrossel (`POST /send/carousel`):** Envio de cards ricos com imagens, textos, rodapés, botões interativos e geração automática de thumbnails JPEG para carregamento instantâneo.

### 5. Pareamento com Passkey (WebAuthn), Mídia Avançada e Regra de 9º Dígito BR
- **Suporte a Passkey (WebAuthn):** Compatível com contas que o servidor do WhatsApp protege com cerimônia de Passkey durante a vinculação de dispositivo.
- **Thumbnails de PDF e Mídia em Base64:** Documentos PDF enviados em `/send/media` ganham miniatura automática da primeira página via `pdftoppm` (`poppler-utils` incluso na imagem), e o campo `url` também aceita conteúdo em Base64.
- **Publicação de Status e Fixação de Mensagens:** Endpoints para publicar Status de texto/mídia (`/send/status/text`, `/send/status/media`) e fixar/desafixar mensagens (`/message/pin`, `/message/unpin`).
- **Números móveis brasileiros:** Preservação correta do 9º dígito em celulares do Brasil e deduplicação de recibos de leitura/entrega por participante em grupos.

---

## 📦 Instalação e Quick Start

### 1. Usando a Imagem Oficial da Expertsa no Docker Hub (Recomendado)

A imagem pronta para produção está publicada em [`dbadaniel/evogo`](https://hub.docker.com/r/dbadaniel/evogo):

```bash
docker pull dbadaniel/evogo:0.7.3
# ou
docker pull dbadaniel/evogo:latest
```

Exemplo de `docker-compose.yml`:

```yaml
services:
  evolution-go:
    image: dbadaniel/evogo:0.7.3
    container_name: evolution-go
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - CLIENT_NAME=evolution
      - GLOBAL_API_KEY=sua-chave-api-segura
      - POSTGRES_USERS_DB=postgresql://postgres:password@postgres:5432/evogo?sslmode=disable
      - POSTGRES_AUTH_DB=postgresql://postgres:password@postgres:5432/evogo?sslmode=disable
      - CONNECT_ON_STARTUP=true
      - DATABASE_SAVE_MESSAGES=false
      - WADEBUG=INFO
      - LOGTYPE=console
```

### 2. Compilando a partir do Código-Fonte

```bash
git clone https://github.com/dbadaniel/evolution-go.git
cd evolution-go

# Build e execução via Docker
make docker-build
make docker-run
```

Para desenvolvimento local (requer **Go 1.26+**):

```bash
make setup
cp .env.example .env
make dev
```

> Execute `make help` para ver todos os comandos disponíveis ou consulte [COMMANDS.md](./COMMANDS.md).

---

## ⚙️ Configuração (`.env`)

```env
# Servidor
SERVER_PORT=8080
CLIENT_NAME=evolution

# Segurança
GLOBAL_API_KEY=your-secure-api-key-here

# Banco de Dados (PostgreSQL)
POSTGRES_AUTH_DB=postgresql://postgres:password@localhost:5432/evogo_auth?sslmode=disable
POSTGRES_USERS_DB=postgresql://postgres:password@localhost:5432/evogo_users?sslmode=disable
DATABASE_SAVE_MESSAGES=false

# Comportamento de Inicialização
CONNECT_ON_STARTUP=true

# Versão do Cliente WhatsApp (Opcional - busca dinâmica automática por padrão)
# WHATSAPP_VERSION_MAJOR=2
# WHATSAPP_VERSION_MINOR=3000
# WHATSAPP_VERSION_PATCH=1052878631

# Logs
WADEBUG=INFO
LOGTYPE=console

# Opcionais (Filas, Webhook Global e Storage S3/MinIO)
# AMQP_URL=amqp://guest:guest@localhost:5672/
# NATS_URL=nats://localhost:4222
# WEBHOOK_URL=https://seu-webhook.com/webhook
# MINIO_ENABLED=true
# MINIO_ENDPOINT=localhost:9000
# MINIO_ACCESS_KEY=minioadmin
# MINIO_SECRET_KEY=minioadmin
```

| Variável                 | Descrição                                                                                                         | Padrão                          |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------- | ------------------------------- |
| `SERVER_PORT`            | Porta HTTP do servidor                                                                                            | `8080`                          |
| `CLIENT_NAME`            | Identificador do cliente exibido no dispositivo conectado                                                         | `evolution`                     |
| `GLOBAL_API_KEY`         | Chave global de autenticação da API                                                                               | **Obrigatório**                 |
| `POSTGRES_USERS_DB`      | URI PostgreSQL para instâncias, configurações e mensagens                                                         | **Obrigatório**                 |
| `POSTGRES_AUTH_DB`       | URI PostgreSQL para chaves de sessão do `whatsmeow` (faz fallback automático para `POSTGRES_USERS_DB` se omitido) | Fallback p/ `POSTGRES_USERS_DB` |
| `CONNECT_ON_STARTUP`     | Reconecta automaticamente todas as sessões pareadas ao iniciar o container (`true` / `false`)                     | `true`                          |
| `DATABASE_SAVE_MESSAGES` | Ativa a persistência de mensagens recebidas/enviadas no banco                                                     | `false`                         |
| `PROXY_PROTOCOL`         | Protocolo padrão de proxy (`http`, `https`, `socks5`)                                                             | Inferido por porta              |
| `WADEBUG`                | Nível de log do cliente WhatsApp (`DEBUG`, `INFO`, `WARN`, `ERROR`)                                               | `INFO`                          |

---

## 📚 Documentação da API e Endpoints Principais

Com o servidor em execução, a interface Swagger UI fica disponível em:

```text
http://localhost:8080/swagger/index.html
```

E o painel **Evolution Manager** em:

```text
http://localhost:8080/manager/login
```

### Principais Endpoints

| Método   | Endpoint               | Descrição                                                                               |
| -------- | ---------------------- | --------------------------------------------------------------------------------------- |
| `POST`   | `/instance/create`     | Cria uma nova instância WhatsApp                                                        |
| `POST`   | `/instance/connect`    | Inicia a conexão e configura Webhook / Eventos assinados                                |
| `GET`    | `/instance/qr`         | Obtém o QR Code atual para pareamento                                                   |
| `POST`   | `/instance/pair`       | Pareamento via código numérico / Passkey                                                |
| `GET`    | `/instance/status`     | Consulta o status de conexão da instância                                               |
| `POST`   | `/instance/disconnect` | Desconecta a sessão mantendo as chaves salvas                                           |
| `DELETE` | `/instance/logout`     | Faz logout do aparelho e limpa a sessão                                                 |
| `POST`   | `/send/text`           | Envia mensagem de texto (suporta menções, quote e `forwardingScore`)                    |
| `POST`   | `/send/media`          | Envia mídia via URL, upload multipart ou Base64 (gera thumbnail de PDF automaticamente) |
| `POST`   | `/send/button`         | Envia mensagens com botões interativos (Reply, CTA URL/Call/Copy, Pix)                  |
| `POST`   | `/send/list`           | Envia mensagens de lista interativa multiplataforma                                     |
| `POST`   | `/send/carousel`       | Envia carrossel de cards interativos com imagens e botões                               |
| `POST`   | `/send/status/text`    | Publica Status em texto (`status@broadcast`)                                            |
| `POST`   | `/send/status/media`   | Publica Status em imagem ou vídeo                                                       |
| `POST`   | `/message/pin`         | Fixa uma mensagem na conversa                                                           |
| `POST`   | `/message/unpin`       | Desafixa uma mensagem na conversa                                                       |

---

## 🛠️ Stack Tecnológica

| Componente                  | Tecnologia                                                          |
| --------------------------- | ------------------------------------------------------------------- |
| Linguagem                   | Go 1.26+                                                            |
| Framework HTTP              | Gin                                                                 |
| Motor WhatsApp              | [go.mau.fi/whatsmeow](https://github.com/tulir/whatsmeow) (Oficial) |
| Banco de Dados              | PostgreSQL + GORM                                                   |
| Mensageria & Eventos        | Webhook, WebSocket, RabbitMQ (AMQP), NATS                           |
| Object Storage              | MinIO / Amazon S3                                                   |
| Processamento de PDF/Imagem | `poppler-utils` (`pdftoppm`), WebP/JPEG nativo                      |
| Container                   | Docker (Alpine Linux 3.21)                                          |

---

## 💡 Conheça também: Expertsa Groups

Precisa escalar suas campanhas e lançamentos no WhatsApp sem perder leads quando os grupos enchem?

Conheça o **[Expertsa Groups](https://expertsa.com.br)** — nossa plataforma inteligente de gerenciamento e redirecionamento para grupos de WhatsApp. Crie um único link de convite que distribui automaticamente os participantes para o próximo grupo disponível, monitora a lotação em tempo real e oferece métricas completas de cliques e conversão para suas automações.

👉 Saiba mais em **[expertsa.com.br](https://expertsa.com.br)**

---

## 💖 Apoie o Projeto (Expertsa)

Manter este fork atualizado com as mudanças constantes de protocolo do WhatsApp, corrigir bugs complexos de sessão e disponibilizar imagens Docker públicas exige dedicação contínua. Se a imagem **`dbadaniel/evogo`** ajuda a sua operação ou empresa no dia a dia, considere apoiar nosso trabalho via **Pix**:

- **Chave Pix (E-mail):** `expertsa.oficial@gmail.com`

Toda contribuição incentiva a continuidade das atualizações e novas melhorias para a comunidade!

---

## 📜 Créditos, Atribuição Oficial e Licença

Este projeto é um fork mantido pela **[Expertsa](https://expertsa.com.br)** construído sobre o trabalho original do **Evolution Go**, desenvolvido pela **[Evolution Foundation](https://evolutionfoundation.com.br/)** ([EvolutionAPI/evolution-go](https://github.com/EvolutionAPI/evolution-go)).

Deixamos registrado nosso reconhecimento e respeito aos criadores e mantenedores originais do ecossistema Evolution:

- **Projeto Original (Upstream):** [EvolutionAPI/evolution-go](https://github.com/EvolutionAPI/evolution-go) — desenvolvido pela [Evolution Team / Evolution Foundation](https://evolutionfoundation.com.br/)
- **Documentação Oficial Evolution:** [docs.evolutionfoundation.com.br](https://docs.evolutionfoundation.com.br/)
- **Comunidade Evolution:** [evolutionfoundation.com.br/community](https://evolutionfoundation.com.br/community)
- **Biblioteca WhatsApp:** [whatsmeow](https://github.com/tulir/whatsmeow) por [tulir (Tulir Asokan)](https://github.com/tulir)

### Licença (Apache License 2.0 + Condições Adicionais da Evolution)

O **Evolution Go** é licenciado sob a **Apache License 2.0**, com as seguintes condições adicionais estabelecidas pelo desenvolvedor original ([LICENSE](./LICENSE)):

1. **Logotipo e direitos autorais (Logo and copyright):** Você não pode remover ou modificar o logotipo ou as informações de direitos autorais no console Evolution ou nos aplicativos ao utilizar componentes de frontend.
2. **Notificação de uso (Usage notification):** Se o Evolution Go for utilizado como parte de qualquer projeto (incluindo código fechado), uma notificação clara de que o Evolution Go está sendo utilizado deve estar visível para os administradores do sistema.

Para dúvidas de licenciamento comercial junto à detentora original da marca Evolution, entre em contato pelo e-mail `contato@evolution-api.com`. Detalhes completos da licença Apache 2.0 em [apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0).

---

<div align="center">

**Evolution Go** — Criado originalmente por [Evolution Foundation](https://evolutionfoundation.com.br/) © Evolution Foundation  
**Expertsa Edition (`dbadaniel/evogo`)** — Evoluído e mantido com foco em solidez por [Expertsa](https://expertsa.com.br)

</div>
