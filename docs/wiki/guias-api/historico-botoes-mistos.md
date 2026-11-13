# Histórico de testes: botões mistos e carrossel

Atualizado em 13/11/2026. Este documento reúne os resultados dos testes do EvoGo na branch `fix/mixed-url-reply-buttons` e no Docker de teste local (`http://localhost:4000`). A alteração não foi publicada no registry nem integrada à `main-expertsa`. O servidor estável de produção permaneceu separado.

## Resultado atual

- **Mensagem avulsa com botão URL e botão reply:** o celular recebe e exibe os botões; no WhatsApp Web do navegador a mensagem aparece como “Não foi possível carregar a mensagem. Use seu celular para acessá-la.” Nenhum formato abaixo resolveu a exibição Web.
- **Carrossel com ações mistas:** o usuário validou que funciona no celular e no WhatsApp Web. É o único formato misto confirmado pelos testes desta investigação.
- **Imagem no carrossel:** foi observado espaço vazio acima da imagem no celular e recorte/escala diferente no Web. O construtor do carrossel não incluía largura/altura no `ImageMessage`; aplicamos `populateImageDimensions` usando os bytes da imagem original. Após esse ajuste, o usuário ainda considerou a apresentação insatisfatória e suspeita da própria imagem de origem. A causa visual não foi confirmada.

## Tentativas de mensagem avulsa

| Tentativa | Formato | Celular | WhatsApp Web |
|---|---|---|---|
| Native Flow misto | `quick_reply` e CTA URL no mesmo `NativeFlowMessage`, com metadados de negócio | Recebeu | Incompatível/indisponível |
| Template hidratado em `DocumentWithCaptionMessage` para dispositivos vinculados | Mudança somente no envelope dos dispositivos vinculados | Recebeu | Incompatível/indisponível |
| Template hidratado direto para dispositivos vinculados | `TemplateMessage`/hydrated template | Recebeu | Não apareceu ou ficou indisponível |
| `ButtonsMessage` com `NativeFlowInfo` por botão | Reply legado e CTA Native Flow no payload dos dispositivos vinculados | Recebeu | Incompatível/indisponível |
| Envelope simples observado no Wuzapi | `InteractiveMessage` Native Flow e nó `biz/interactive/native_flow` | Recebeu | Incompatível/indisponível no nosso ambiente |
| Conversão de dispositivo vinculado ativada por `messageParamsJSON` | Principal recebe Native Flow; dispositivos vinculados recebem `ButtonsMessage` com `NativeFlowInfo` | Recebeu | Incompatível/indisponível |
| Envelope HSM para template misto | Marcador HSM experimental na camada de transporte | A conta recebeu erro 403 e desconectou | Não validado |

Nos envios das 16:04 e 16:10, o servidor retornou sucesso e registrou recibos. Isso confirma transporte/recepção pelo WhatsApp, mas não que o cliente Web conseguiu decodificar a mensagem. No envio das 16:10, os logs registraram o formato `buttons_native_flow_info` para os dispositivos vinculados; ainda assim, o Web mostrou a mensagem indisponível.

## Risco de desconexão da conta

A tentativa HSM foi seguida por erro 403 e desconexão/logout da conta conectada. A solicitação de HSM não foi aceita como método válido naquele envio. O código dessa tentativa foi removido. **Não repetir nem reintroduzir o transporte HSM sem evidência nova e um plano de recuperação da sessão.**

O estado “success” do endpoint ou um recibo de transporte não demonstra compatibilidade visual. As mensagens de teste foram enviadas para uma conta real e a desconexão ocorreu durante a tentativa HSM; por isso esse formato permanece retirado.

## Referências externas verificadas

- [Wuzapi, versão `v1.1.0` citada no issue](https://github.com/asternic/wuzapi/blob/49bf3dd/handlers.go): constrói reply e CTA URL no mesmo Native Flow. A issue [#409](https://github.com/asternic/wuzapi/issues/409) relata teste em Web, mas o formato correspondente não reproduziu o resultado no nosso ambiente.
- [Quepasa](https://github.com/OfficialMoAdel/quepasa/blob/main/src/whatsmeow/whatsmeow_extensions.go): o construtor encontrado gera somente botões `ButtonsMessage_Button_RESPONSE`; não contém exemplo de URL misturada com reply.
- [WhatsApp Web parser harness](../../../_evo-output/implementation-artifacts/mixed-url-reply-buttons/verify-public-web-parser.cjs): análise offline do parser público. O resultado do parser não comprova entrega/aceitação do servidor nem substitui testes reais nos clientes.

## Imagem do carrossel

A rotina de dimensões já era usada nos envios comuns de imagem, mas não no `ImageMessage` aninhado no cabeçalho do cartão. Ela agora é aplicada à imagem original no construtor do carrossel; o thumbnail continua sendo redimensionado separadamente. O usuário ainda observou layout ruim depois da atualização. Falta validar com uma imagem de origem conhecida e conferir dimensões/formato reais do arquivo para diferenciar metadados ausentes de proporção/corte do asset.
