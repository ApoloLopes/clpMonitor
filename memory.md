# Memory – clpMonitor (PLC Gateway)

## Visão geral
Gateway em Go que faz a ponte entre CLPs Siemens (S7) e Rockwell (CIP/EtherNet-IP) e um dashboard web em tempo real. Módulo: `github.com/ApoloLopes/clpMonitor`.

## Stack
- Backend: Go, Fiber v2 (REST + estáticos), `gofiber/websocket/v2`
- Drivers: `robinson/gos7` (Siemens), `danomagnum/gologix` (Rockwell)
- Frontend: HTML/CSS/Vanilla JS sem CDN (`web/index.html`, `web/css/style.css`, `web/js/app.js`): páginas Monitoramento, Dispositivos, Tags e Rede (hash routing, diálogos nativos <dialog>)
- Persistência: arquivo JSON (`gateway_config.json`)

## Estrutura
- `cmd/server/main.go` – entrada: storage → WSHub → Manager → Fiber (porta **8080**). Se a config estiver vazia, cria o device `PLC_01` (s7, 127.0.0.1, rack 0, slot 2) e a tag `T1`.
- `internal/models/models.go` – `Tag` (ID, DeviceID, Name, Address, DBNumber, Offset, Bit, DataType, Value, Quality, Timestamp) e `Device` (ID, Name, Protocol "s7"|"cip", IPAddress, Rack, Slot).
- `internal/plc/` – interface `Driver` (`Connect(ctx)`, `Disconnect()`, `ReadTags(*[]Tag)` in-place); `s7_driver.go`, `rockwell_driver.go`.
- `internal/worker/worker.go` – `PollingWorker`: ticker por dispositivo, lê tags, envia cópia ao `DataChannel` de forma não-bloqueante (descarta se ninguém ouvir).
- `internal/worker/manager.go` – `Manager`: `StartAll` (um worker + goroutine ponte por device, intervalo fixo 500 ms), `StopAll` (cancela contexto), `Reload` (hot-reload = Stop + Start).
- `internal/api/routes.go` – `GET /api/config`, `POST /api/devices`, `POST /api/tags`; cada POST grava e chama o callback `onConfigChange` (Manager.Reload).
- `internal/api/ws_hub.go` – `WSHub`: registra/remove clientes, `Broadcast chan []models.Tag` enviado a todos os navegadores; endpoint `/ws`.
- `internal/storage/json_storage.go` – `JSONStorage` (`GetConfig`, `AddDevice`, `AddTag`).

## Fluxo de dados
CLP → Driver.ReadTags → PollingWorker (500 ms) → DataChannel → goroutine ponte → `hub.Broadcast` → WebSocket `/ws` → `app.js`.

## Pontos de atenção / possíveis melhorias
- Intervalo de polling fixo (500 ms) no Manager; não configurável.
- Sem endpoints de update/delete para devices e tags (apenas POST).
- `Reload` não sincroniza com o fim dos workers antigos (`cancelFunc` sem sincronização/mutex).
- Broadcast do hub é global: o payload é um `[]Tag` sem identificar o device de origem além de `Tag.DeviceID`.
- `go.mod` marca todas as dependências como `// indirect` e declara `go 1.27.2` (conferir); README cita Go 1.21+.
- O README ainda usa o nome de pasta `plc-gateway/`.
- Sem autenticação nem testes automatizados.

## Como rodar
```
go run ./cmd/server
```
Abrir `http://localhost:8080`.


## Atualização
- API agora tem PUT/DELETE de devices e tags (com validação, 409 em ID duplicado, exclusão de device apaga suas tags), GET /api/devices/status e /api/network/interfaces (+ state, ipv4, addresses). Pacote internal/netmgr: leitura via net.Interfaces + sysfs; escrita via comando ip (somente Linux, requer root, não persiste após reboot).
- Fiber reaproveita buffers: copiar c.Params()/c.Query() (strings.Clone) antes de guardar.
- Testes: go test ./internal/...
