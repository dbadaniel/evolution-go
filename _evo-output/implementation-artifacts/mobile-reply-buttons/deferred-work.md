# Trabalho adiado — botões de resposta móveis

- O caminho legado de botões `reply` com `imageUrl` ou `videoUrl` ignora falhas de download, leitura ou upload e pode enviar a mensagem sem o cabeçalho de mídia. O comportamento já existia antes desta correção e deve ser tratado em uma mudança separada, pois alterar sua política de erro amplia o contrato público de `/send/button`.
