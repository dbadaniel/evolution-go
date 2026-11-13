# Guia de utilização de botões, Pix e listas — Evogo ExpertSA

Referência conferida no código da versão `main-expertsa`, em 13/11/2026. Documento destinado a quem integra a API. Os exemplos são modelos; este documento não dispara mensagens.

## 1. Endpoints e autenticação

| Operação | Método e caminho |
|---|---|
| Respostas rápidas, URL, ligação, cópia e Pix | `POST /send/button` |
| Menu com seções e opções | `POST /send/list` |
| Cards com botões | `POST /send/carousel` |

Base pública usada na stage: `https://devogo.expertsa.com.br`. Acesso local: `http://localhost:4000`. O túnel público encaminha para o serviço local; o contrato JSON é o mesmo.

Todos os endpoints recebem JSON com estes headers:

```http
Content-Type: application/json
apikey: SUA_CHAVE_DA_INSTANCIA
```

A chave seleciona a instância remetente. `number` é o destinatário. Não é necessário incluir o nome da instância na URL ou no JSON. A instância precisa estar pareada e conectada.

Os nomes dos campos diferenciam maiúsculas de minúsculas nos exemplos de integração: use exatamente os nomes apresentados. Para `/send/button`, os valores de `type` são comparados literalmente; use minúsculas.

### Como ler as regras

- **Obrigatório validado:** a API rejeita sua ausência ou valor vazio na validação indicada.
- **Necessário para uso:** preencha para a função ter sentido, mesmo quando o servidor não valida esse campo individualmente.
- **Opcional:** pode ser omitido. Um campo aceito pelo JSON não garante que seja utilizado em todos os formatos.

Não use campos vazios ou tipos desconhecidos para testar se a mensagem funciona. Parte das validações ocorre somente depois da verificação de conexão.

## 2. Campos comuns de `/send/button`

| Campo | Tipo JSON | Regra | Descrição |
|---|---|---|---|
| `number` | string | Obrigatório validado | DDI + DDD + número, sem espaços, `+` ou pontuação; exemplo `5511982081887`. Para grupo, usar o JID completo, como `120363000000000000@g.us`. |
| `title` | string | Obrigatório validado | Título da mensagem. |
| `description` | string | Obrigatório validado | Texto principal da mensagem. |
| `footer` | string | Obrigatório validado | Rodapé; não pode ser vazio nem omitido. |
| `buttons` | array de objetos | Necessário para uso | Envie ao menos um botão. A implementação atual não rejeita explicitamente o array vazio. |
| `id` | string | Opcional | ID da **mensagem inteira**. Se omitido, o servidor gera um ID. Prefira omitir; se usar, gere um ID novo a cada envio. Não é o ID da resposta do botão. |
| `delay` | inteiro de 32 bits | Opcional | Espera em milissegundos antes do envio. Valores positivos enviam presença de digitação e aguardam; omitido equivale a zero. Não agenda uma campanha. |
| `quoted` | objeto | Opcional | Referência a uma mensagem anterior; campos `messageId` e `participant`. |
| `formatJid` | boolean | Opcional | Controla a normalização do destinatário; omitido segue o comportamento padrão de formatação. Para uso normal, omita. |
| `mentionedJid` | array de strings | Opcional | JIDs a mencionar. Não é uma string única. Aplicação varia conforme o formato da mensagem. |
| `mentionAll` | boolean | Opcional | Solicita menção a participantes de grupo; omitido equivale a `false`. Aplicação varia conforme o formato. |
| `imageUrl` | string | Opcional, com limitação | Em `reply`, seleciona o caminho legado com imagem. Esse caminho não é o formato textual validado no celular/Web. |
| `videoUrl` | string | Opcional, com limitação | Em `reply`, seleciona o caminho legado com vídeo. Se ambos os campos de mídia forem preenchidos, a imagem tem prioridade. |

Nos builders atuais de CTA e Pix, `imageUrl` e `videoUrl` não são utilizados para anexar mídia. Para o cenário validado de respostas rápidas, omita ambos. Não considere as menções em listas ou no caminho legado como funcionalidade validada: o código comum trata menções por tipo de mensagem e não possui tratamento equivalente para todos eles.

Exemplo de referência a uma mensagem:

```json
{
  "quoted": {
    "messageId": "ID_DA_MENSAGEM_ANTERIOR",
    "participant": "5511999999999@s.whatsapp.net"
  }
}
```

`participant` identifica o autor citado, especialmente relevante em grupos. Esse objeto é adicional aos demais campos obrigatórios do envio.

## 3. Tipos disponíveis em `/send/button`

| `type` | Ação | Campos necessários para uso | Opcionais específicos |
|---|---|---|---|
| `reply` | Enviar uma resposta selecionada pelo usuário | `displayText`, `id` | Nenhum |
| `url` | Abrir um endereço | `displayText`, `url` | `id` não é utilizado por esse botão |
| `call` | Acionar ligação telefônica | `displayText`, `phoneNumber` | `id` não é utilizado por esse botão |
| `copy` | Copiar texto | `displayText` e `copyCode` ou `id` | `id` pode ser gerado automaticamente |
| `pix` | Exibir dados de uma chave Pix estática | `name`, `keyType`, `key` | Rótulo do botão é controlado pelo WhatsApp |

O servidor não valida individualmente todos os campos dessa tabela. Por exemplo, URL vazia ou `displayText` ausente pode passar pelo handler e resultar em mensagem inválida ou não exibida. Valide esses campos na integração.

### Combinações e quantidades

- `reply`: de 1 a **3** botões; use somente `reply` no mesmo envio.
- `pix`: exatamente **1** botão, sem combinar com outro tipo.
- `url`, `call` e `copy`: podem ser combinados pelo código, mas as combinações e quantidades não foram todas validadas em entrega. Para começar, envie um CTA por mensagem.
- Não existe um limite geral de caracteres ou de quantidade de CTAs validado pelo código atual. Isso não significa quantidade ou tamanho ilimitado no WhatsApp.
- Misturar `reply` com CTA é rejeitado. Misturar `reply` com Pix também é rejeitado pela regra de exclusividade do Pix.

### 3.1. Resposta rápida: Sim e Não

```json
{
  "number": "5511982081887",
  "title": "testado",
  "description": "Você confirma esta opção?",
  "footer": "Selecione uma resposta",
  "buttons": [
    {"type": "reply", "id": "sim", "displayText": "confirmado"},
    {"type": "reply", "id": "nao", "displayText": "recusar serviço"}
  ]
}
```

`displayText` é o texto visível. `buttons[].id` é o valor que sua aplicação usa para identificar a escolha. Use IDs distintos e estáveis dentro de cada menu; o servidor não verifica duplicidade de IDs em botões de resposta.

Não confunda `buttons[].id` com o `id` na raiz. O ID da raiz é opcional e identifica o envio inteiro. Reutilizá-lo não é uma forma documentada de garantir idempotência.

### 3.2. CTA de URL

```json
{
  "number": "5511982081887",
  "title": "Acesse nosso site",
  "description": "Toque abaixo para abrir o site.",
  "footer": "Atendimento",
  "buttons": [
    {
      "type": "url",
      "displayText": "Abrir site",
      "url": "https://oconsultoriojunguiano.com.br"
    }
  ]
}
```

Use uma URL completa, preferencialmente HTTPS. O botão abre o endereço; não envia automaticamente uma resposta de texto ao remetente.

### 3.3. CTA de ligação

```json
{
  "number": "5511982081887",
  "title": "Fale conosco",
  "description": "Toque para ligar para o atendimento.",
  "footer": "Ligação telefônica",
  "buttons": [
    {"type": "call", "displayText": "Ligar", "phoneNumber": "+5511999999999"}
  ]
}
```

`phoneNumber` é o número da ligação, não o destinatário da mensagem. Preencha em formato internacional. Não substitua por `id`, que não é usado para a ligação neste endpoint. O comportamento depende dos recursos do dispositivo do destinatário.

### 3.4. CTA de cópia

```json
{
  "number": "5511982081887",
  "title": "Seu código",
  "description": "Copie o código abaixo para utilizar no atendimento.",
  "footer": "Código de referência",
  "buttons": [
    {"type": "copy", "displayText": "Copiar código", "copyCode": "ATENDIMENTO123"}
  ]
}
```

Regras implementadas:

- O texto copiado vem de `copyCode`.
- Se `copyCode` estiver vazio, o servidor usa `id` como texto a copiar.
- Se `id` estiver vazio, o servidor gera o identificador do botão; isso não cria um texto útil para cópia quando ambos os campos originais estão vazios.
- Recomenda-se sempre informar `copyCode` explicitamente.

### 3.5. Pix

```json
{
  "number": "5511982081887",
  "title": "Pagamento Pix",
  "description": "Toque para consultar os dados do Pix.",
  "footer": "Confira o recebedor antes de pagar",
  "buttons": [
    {
      "type": "pix",
      "name": "NOME DO RECEBEDOR",
      "keyType": "random",
      "key": "SUA_CHAVE_ALEATORIA_CADASTRADA"
    }
  ]
}
```

Substitua os valores de exemplo por dados reais antes de enviar. O contrato atual envia **chave estática sem valor de cobrança**. Não gera cobrança bancária dinâmica, não confirma pagamento e não oferece parâmetro público para valor ou vencimento.

| Campo do botão Pix | Regra |
|---|---|
| `type` | Use `pix`. |
| `name` | Nome do recebedor. Necessário para uso; não há validação individual de nome vazio. |
| `keyType` | Tipo da chave, conforme tabela abaixo. Necessário para uso; não há lista fechada validada pelo handler. |
| `key` | Obrigatório validado: chave não pode ser vazia após remoção de espaços nas extremidades. |
| `displayText` | Não é usado na montagem do botão Pix. |
| `currency` | Campo aceito na estrutura, mas ignorado neste envio: o builder fixa `BRL`. |
| `id` | O ID dentro do botão não é usado como chave ou referência do pagamento. A referência interna é gerada pelo servidor. |

| Chave | `keyType` recomendado | Conteúdo de `key` | Tipo interno |
|---|---|---|---|
| CPF | `cpf` | CPF cadastrado; aceita pontuação e normaliza para dígitos | `CPF` |
| CNPJ | `cnpj` | CNPJ cadastrado; prefira somente dígitos | `CNPJ` |
| E-mail | `email` | E-mail cadastrado | `EMAIL` |
| Telefone | `phone` | Telefone cadastrado com DDI, como `+5511999999999` | `PHONE` |
| Aleatória | `random` | Chave aleatória cadastrada, completa | `EVP` |

Use os valores minúsculos da tabela para garantir o mapeamento. A implementação repassa tipos desconhecidos; isso não os torna suportados.

O CPF tem validação local de tamanho, dígitos repetidos e dígitos verificadores. CPF inválido retorna HTTP 400. Para os demais tipos, o código atual apenas remove espaços nas extremidades e exige chave não vazia: não normaliza CNPJ/telefone e não valida o formato ou a existência da chave. Mesmo um CPF matematicamente válido pode não estar cadastrado no Pix. A API não consulta o cadastro bancário.

Não envie `amount`, `value`, `expiration` ou um código Pix copia e cola no lugar da chave esperando gerar cobrança: esses usos não fazem parte deste contrato. O exemplo de chave aleatória demonstra a montagem; não comprova entrega para todos os tipos de chave.

## 4. Listas: `/send/list`

Uma lista abre um menu com seções e linhas. Ela não é um botão com `type: "list"` dentro de `/send/button`.

| Campo | Tipo | Regra |
|---|---|---|
| `number` | string | Obrigatório validado. |
| `title` | string | Obrigatório validado. |
| `description` | string | Obrigatório validado. |
| `footerText` | string | Obrigatório validado. Atenção: **não é `footer`**. |
| `buttonText` | string | Obrigatório validado; texto que abre o menu. |
| `sections` | array | Obrigatório validado; entre 1 e 10 seções. |
| `sections[].title` | string | Opcional; recomenda-se preencher para organizar o menu. |
| `sections[].rows` | array | Cada seção precisa ter ao menos uma linha. Máximo de **10 linhas somando todas as seções**. |
| `sections[].rows[].title` | string | Necessário para uso; o código permite vazio e o substitui por espaço. |
| `sections[].rows[].description` | string | Opcional. |
| `sections[].rows[].rowId` | string | Opcional; se omitido, é gerado. IDs explícitos devem ser únicos em toda a lista. Prefira informar para identificar escolhas no backend. |
| `id`, `delay`, `quoted`, `formatJid` | mesmos tipos anteriores | Opcionais, com a mesma finalidade geral. |
| `mentionedJid`, `mentionAll` | array de strings, boolean | Aceitos na estrutura; menções em listas não devem ser consideradas validadas. |

```json
{
  "number": "5511982081887",
  "title": "Menu de atendimento",
  "description": "Selecione uma opção:",
  "footerText": "Atendimento ExpertSA",
  "buttonText": "Ver opções",
  "sections": [
    {
      "title": "Serviços",
      "rows": [
        {"title": "Agendar", "description": "Consultar horários", "rowId": "agendar"},
        {"title": "Suporte", "description": "Falar com o atendimento", "rowId": "suporte"}
      ]
    }
  ]
}
```

Não há seleção múltipla nesse contrato. A resposta identifica uma linha escolhida. Mesmo que o erro diga `footer is required`, o campo correto desse endpoint é `footerText`.

## 5. Carrossel: `/send/carousel`

O carrossel possui um contrato próprio. Não reutilize os objetos de `/send/button` sem adaptar os campos.

| Campo | Regra atual |
|---|---|
| `number` | Obrigatório validado. |
| `cards` | Obrigatório validado; pelo menos um card. Para um carrossel de vários cards, use pelo menos dois; não há validação local de máximo. |
| `body`, `footer` | Textos gerais opcionais. Aqui o rodapé pode ser omitido. |
| `id`, `delay`, `quoted` | Opcionais. |
| `formatJid` | Aceito e usado na análise inicial do destinatário, mas não repassado ao envio comum; não dependa de `false` para contornar a validação final. |
| `cards[].header.title`, `cards[].header.subtitle` | Textos do cabeçalho; não são exigidos individualmente pelo handler. |
| `cards[].header.imageUrl`, `cards[].header.videoUrl` | Mídia opcional na estrutura; imagem tem prioridade. Use mídia válida para os cenários de cards ricos; compatibilidade precisa ser testada. |
| `cards[].body.text` | Texto do card; necessário para um card informativo, sem validação individual de vazio. |
| `cards[].footer` | Rodapé opcional do card. |
| `cards[].buttons` | Botões opcionais do card. |

Mapeamento dos botões **dentro de um card**:

| `type` | Campos utilizados |
|---|---|
| `reply` | `displayText` e `id` como identificador de resposta |
| `url` | `displayText` e **`id` como URL**; não usa campo `url` |
| `call` | `displayText` e **`id` como telefone**; não usa campo `phoneNumber` |
| `copy` | `displayText` e `copyCode`; sem o fallback de cópia de `/send/button` |

O carrossel converte `type` para maiúsculas internamente. Tipo vazio ou desconhecido cai em resposta rápida; isso não significa suporte ao tipo desconhecido. **Pix não é implementado nos botões do carrossel.**

Exemplo estrutural, ainda sujeito a teste de entrega:

```json
{
  "number": "5511982081887",
  "body": "Conheça os serviços",
  "cards": [
    {
      "header": {"title": "Consultoria", "imageUrl": "https://SEU_DOMINIO/consultoria.jpg"},
      "body": {"text": "Veja os detalhes da consultoria."},
      "buttons": [{"type": "url", "displayText": "Saiba mais", "id": "https://SEU_DOMINIO/consultoria"}]
    },
    {
      "header": {"title": "Suporte", "imageUrl": "https://SEU_DOMINIO/suporte.jpg"},
      "body": {"text": "Converse com nossa equipe."},
      "buttons": [{"type": "reply", "displayText": "Quero suporte", "id": "suporte"}]
    }
  ]
}
```

Substitua as URLs. O builder de botões do carrossel interpola textos em JSON; evite aspas duplas e quebras de linha nos campos dos botões sem validar o resultado. Esse caminho não tem as mesmas garantias dos builders de botão avulso.

## 6. cURL para WSL / Bash

Salve um dos payloads completos acima em `payload.json`. Ajuste o endpoint conforme o exemplo escolhido:

```bash
BASE_URL='https://devogo.expertsa.com.br'
read -rsp 'Chave da instância: ' API_KEY; echo
curl --request POST "$BASE_URL/send/button" \
  --header 'Content-Type: application/json' \
  --header "apikey: $API_KEY" \
  --data-binary @payload.json
unset API_KEY
```

Para lista, use `"$BASE_URL/send/list"`; para carrossel, `"$BASE_URL/send/carousel"`. O arquivo deve conter JSON válido em UTF-8, sem comentários. Não use a sintaxe de continuação do PowerShell (crase) dentro do WSL; no Bash a continuação é `\`.

## 7. Receber a escolha pelo webhook

Habilite `BUTTON_CLICK` ou `MESSAGE` nos eventos da instância e configure o webhook. Respostas interativas recebidas podem gerar `ButtonClick`.

Exemplo simplificado de resposta a um botão, com apenas parte dos campos:

```json
{
  "event": "ButtonClick",
  "instanceId": "UUID_DA_INSTANCIA",
  "instanceName": "NOME_DA_INSTANCIA",
  "data": {
    "buttonId": "sim",
    "buttonText": "confirmado",
    "type": "native_flow_response",
    "messageId": "ID_DA_RESPOSTA_DO_USUARIO"
  }
}
```

- Resposta rápida moderna: `type: "native_flow_response"`; a identificação vem dos parâmetros recebidos do cliente.
- Resposta rápida legada: `type: "buttons_response"`.
- Lista: `type: "list_response"`; `buttonId` corresponde ao `rowId` selecionado.
- Também existe tratamento de `template_button_reply`.
- `data.messageId` identifica a **mensagem de resposta**, não necessariamente o envio original. Use IDs de opções adequados ao seu fluxo e guarde o ID do envio retornado pela API.
- `instanceId` identifica a instância que recebeu o evento. Não use somente o número para separar instâncias.
- Abrir URL, copiar texto, iniciar ligação ou abrir dados Pix **não garante** uma resposta interativa recebida. Não trate `ButtonClick` como rastreamento universal de todos os CTAs ou como confirmação de pagamento.

## 8. Retorno, entrega e erros

Formato geral de sucesso:

```json
{
  "message": "success",
  "data": {
    "Info": {
      "ID": "ID_DA_MENSAGEM",
      "Chat": "JID_DO_DESTINATARIO"
    }
  }
}
```

Esse exemplo omite outros campos reais. **HTTP 200 e `success` não comprovam entrega, renderização nem leitura.** Guarde `data.Info.ID`, acompanhe recibos e confirme a apresentação nos clientes usados pelo destinatário. Envio Pix também não comprova pagamento.

| Erro / situação | Interpretação e ação |
|---|---|
| `phone number is required` | Preencher `number`. |
| `title is required` | Preencher `title`, mesmo que prefira uma mensagem curta. |
| `description is required` | Preencher `description`. |
| `footer is required` | Em botão, preencher `footer`; em lista, preencher `footerText`. |
| `button text is required` | Lista sem `buttonText`. |
| Chave Pix vazia ou CPF inválido | Corrigir `key`; validação local retorna HTTP 400. |
| Seções/linhas fora dos limites ou `rowId` duplicado | Corrigir a lista; retorna HTTP 400. |
| Mais de 3 replies, tipos incompatíveis ou tipo desconhecido | Erros do serviço de botões; atualmente retornados como HTTP 500, mesmo sendo problema de payload. |
| `server returned error 405` / `473` | Rejeição do WhatsApp. Não permite concluir sozinho qual campo está errado; verificar payload, versão e logs. |
| `websocket disconnected before message send returned response` | Envio sem resposta final por perda da conexão; verificar sessão e recibos antes de repetir e arriscar duplicidade. |
| `success`, mas nenhuma mensagem visível | Investigar recibos e compatibilidade; não presumir entrega pelo retorno. |

Não envie o corpo JSON `null`: o handler de lista o rejeita, mas os handlers atuais de botão/carrossel não têm a mesma proteção explícita.

## 9. O que já foi validado nesta versão

| Cenário | Evidência disponível nesta sessão |
|---|---|
| Reply textual, sem imagem/vídeo | Confirmado no celular e no cliente Web/Windows. |
| URL avulsa | Entrega funcional confirmada pelo usuário após os ajustes de CTA. |
| Pix | Confirmado no celular e Web após os ajustes; não equivale a teste de todos os tipos de chave. |
| Lista | Mensagem aparece e menu abre no celular e Web. |
| Call, copy, CTAs combinados e carrossel | Implementados; não houve confirmação específica de teste nesta sessão. |
| Reply com mídia, grupos e iOS | Não houve validação específica suficiente nesta sessão. |

O formato público continua sendo o JSON deste guia. Não é necessário enviar `native_flow`, `payment_info`, metadados de dispositivo ou envelopes internos do WhatsApp: o Evogo os monta.

## 10. Referências no repositório

- Contratos, montagem e regras: `pkg/sendMessage/service/send_service.go` (`Button`, `ButtonStruct`, `ListStruct`, `CarouselStruct`, `SendButton`, `SendList`, `SendCarousel`).
- Validação HTTP: `pkg/sendMessage/handler/send_handler.go` (`SendButton`, `SendList`, `SendCarousel`).
- Eventos de resposta: `pkg/whatsmeow/service/whatsmeow.go` (`ButtonClick`).
- [Referência geral de mensagens interativas](api-interactive.md).

As regras deste guia foram conferidas na implementação local. Limites adicionais e apresentação final podem depender do WhatsApp e da versão do cliente receptor.
