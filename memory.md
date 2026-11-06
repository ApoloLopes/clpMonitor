# Memory – clpMonitor (PLC Gateway)

## Visão Geral
Gateway industrial em Go de alta performance que conecta CLPs Siemens (S7) e Rockwell (CIP/EtherNet-IP) a uma aplicação web moderna em tempo real, sem dependência de middlewares pesados ou servidores OPC intermediários.
Módulo Go: `github.com/ApoloLopes/clpMonitor`.

## Stack Tecnológica
- **Backend:** Go (1.21+), Fiber v2 (REST e arquivos estáticos), `gofiber/websocket/v2`.
- **Drivers de Comunicação:** `robinson/gos7` (Siemens S7Comm), `danomagnum/gologix` (Rockwell CIP).
- **Rede do Host:** Pacote próprio `internal/netmgr` utilizando `net.Interfaces`, `sysfs` e comandos de rede Linux (`ip`) para configuração em tempo real.
- **Frontend:** SPA desenvolvida em HTML5 / CSS3 / Vanilla JS puro, offline-first (sem dependência de CDNs ou bibliotecas externas), utilizando elementos nativos `<dialog>` para modais e hash routing (`#dashboard`, `#devices`, `#tags`, `#network`).
- **Persistência:** Ficheiro JSON com bloqueio seguro via `sync.RWMutex` (`gateway_config.json`).

## Arquitetura e Estrutura
- `cmd/server/main.go`: Entrada principal. Inicializa o storage JSON, o hub de WebSockets (`WSHub`), o orquestrador de workers (`Manager`), registra rotas da API com suporte a hot-reload e levanta o servidor Fiber na porta `8080`.
- `internal/models/models.go`:
  - `Device`: ID, Name, Protocol ("s7" ou "cip"), IPAddress, Rack, Slot.
  - `Tag`: ID, DeviceID, Name, Address, DBNumber, Offset, Bit, DataType ("BOOL", "INT", "REAL"), Value, Quality, Timestamp.
- `internal/plc/`:
  - Interface `Driver`: métodos `Connect(ctx)`, `Disconnect()`, `ReadTags(*[]Tag)`.
  - `s7_driver.go`: Suporte a leitura Siemens S7 (`gos7`), incluindo leitura bit a bit de DBs (`GetBoolAt`).
  - `rockwell_driver.go`: Suporte a leitura CIP (`gologix`) por tag simbólica.
- `internal/worker/`:
  - `worker.go` (`PollingWorker`): Ciclo periódico de leitura por CLP em goroutine dedicada; notifica alterações de tags no canal de dados e loga no console apenas quando houver transição de valor ou estado de qualidade.
  - `manager.go` (`Manager`): Gerencia ciclo de vida dos workers com controle de concorrência (`statusMu`, `reloadMu`), monitora estados de conexão ("online", "offline", "connecting", "unknown") e executa Hot-Reload (`Reload`) de forma atômica e graciosa.
- `internal/netmgr/netmgr.go`: Leitura de portas de rede (MAC, MTU, Carrier/Link, Velocidade, Duplex, Bytes RX/TX, Gateway) e comandos de alteração de estado (`SetState`), endereço estático (`SetStatic`), inclusão (`AddAddress`), edição em tempo real (`EditAddress`) e exclusão (`RemoveAddress`).
- `internal/api/`:
  - `routes.go`: Endpoints REST com validações rigorosas (409 para duplicados, 400 para parâmetros inválidos) e clonagem de buffers de memória do Fiber (`param()` / `strings.Clone()`).
  - `ws_hub.go`: Hub WebSocket distribuindo broadcasts de leitura em tempo real para os clientes conectados.
  - `routes_test.go`: Testes unitários cobrindo o ciclo de vida completo de dispositivos, tags e rotas de rede.
- `web/`:
  - `index.html`: Layout da interface, barra lateral de navegação e diálogos modais nativos.
  - `css/style.css`: Estilização escura moderna e responsiva.
  - `js/app.js`: Roteamento por hash, conexão WebSocket com auto-reconnect, sincronização dinâmica de formulários e gerenciamento de chamadas REST.
- `build_arm.bat`: Script de compilação automatizada com flags de redução de tamanho (`-ldflags="-s -w"`) visando arquitetura Linux ARMv7 (`GOOS=linux GOARCH=arm GOARM=7`).

## Rotas REST
- `GET /api/config`: Retorna a configuração completa (dispositivos e tags).
- `GET /api/devices/status`: Retorna o estado de comunicação em tempo real de cada CLP.
- `POST /api/devices`: Cria novo CLP.
- `PUT /api/devices/:id`: Atualiza parâmetros do CLP e dispara hot-reload.
- `DELETE /api/devices/:id`: Exclui CLP e cascateia a remoção de todas as suas tags associadas.
- `POST /api/tags`: Cria nova tag vinculada a um CLP.
- `PUT /api/tags/:id`: Atualiza configuração da tag.
- `DELETE /api/tags/:id`: Exclui a tag.
- `GET /api/network/interfaces`: Lista detalhes e estatísticas das interfaces de rede da máquina.
- `POST /api/network/interfaces/:name/state`: Liga ou desliga interface no Linux.
- `PUT /api/network/interfaces/:name/ipv4`: Define IPv4 estático único.
- `POST /api/network/interfaces/:name/addresses`: Adiciona IP adicional.
- `PUT /api/network/interfaces/:name/addresses`: Edita IP existente em tempo real.
- `DELETE /api/network/interfaces/:name/addresses`: Remove IP especificado da interface.

## Procedimentos de Teste e Compilação
- Testes unitários do backend:
  ```bash
  go test ./internal/...
  ```
- Execução local para testes:
  ```bash
  go run ./cmd/server
  ```
- Build para Linux ARM:
  Executar `build_arm.bat` ou via PowerShell com variáveis de ambiente dedicadas.
