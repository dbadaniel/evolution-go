# Trabalho adiado

- Sincronizar o acesso ao mapa global `clientPointer` em todos os serviços e no ciclo de vida do whatsmeow. A revisão deste endpoint confirmou leituras e escritas concorrentes preexistentes em diversos pacotes; corrigir apenas esta rota não elimina a corrida e uma solução coerente exige um proprietário único ou `RWMutex` compartilhado.
- Disponibilizar no ambiente de desenvolvimento um compilador C compatível com o pacote `github.com/chai2010/webp`, permitindo executar `go test ./...` com CGO habilitado.
