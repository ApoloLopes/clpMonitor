# 🏭 PLC Gateway - Siemens & Rockwell

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)
![Siemens S7](https://img.shields.io/badge/Protocol-S7_Comm-009999?style=for-the-badge)
![Rockwell CIP](https://img.shields.io/badge/Protocol-CIP_EtherNet/IP-CC0000?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)

Gateway industrial de comunicação de alta performance desenvolvido em **Go (Golang)** para atuar como ponte entre Controladores Lógicos Programáveis (CLPs) da Siemens e Rockwell e aplicações web/sistemas de supervisão.

O projeto inclui um painel web completo, responsivo e 100% autônomo (sem dependência de CDN/internet), permitindo monitoramento em tempo real, gerenciamento de equipamentos, mapeamento de variáveis e configuração das portas de rede da plataforma host (Linux/Windows).

---

## ✨ Funcionalidades

- **Comunicação Nativa com CLPs:**
  - **Siemens S7:** Suporte a S7-1200, S7-1500, S7-300 via protocolo S7 (biblioteca `gos7`). Leitura de tipos `BOOL` (DBX bit a bit), `INT` (16 bits) e `REAL` (Float 32 bits).
  - **Rockwell CIP / EtherNet/IP:** Suporte a CompactLogix e ControlLogix via protocolo CIP nativo (biblioteca `gologix`) por endereçamento simbólico (`BOOL`, `INT`, `REAL`).
- **Dashboard Web Integrado (Offline-first / Sem CDN):**
  - **Monitoramento em Tempo Real:** Streaming contínuo via WebSockets (`/ws`) com latência mínima, exibindo valores, qualidade da leitura e estado dos CLPs com indicadores visuais.
  - **Gerenciador de Dispositivos:** CRUD completo (criação, edição e exclusão com remoção de tags em cascata) e verificação do estado de conexão ao vivo (Online, Offline, A conectar, Desconhecido).
  - **Gerenciador de Tags:** CRUD completo com busca textual, filtro por CLP e formulários contextuais inteligentes (campos específicos para Siemens DB/Offset/Bit ou Rockwell Tag Name).
  - **Gerenciador de Portas Ethernet do Host:** Visualização das interfaces de rede físicas e virtuais da plataforma (MAC, MTU, Carrier/Link, RX/TX, velocidades e gateways) com suporte a ligar/desligar interface, definir IP estático e adicionar/editar/remover IPs em tempo real no Linux.
- **Alta Concorrência e Tolerância a Falhas:**
  - Cada CLP opera em sua própria goroutine/worker, com conexão isolada para evitar que falhas ou timeouts travem outros dispositivos ou a API.
  - Reconexão automática em caso de queda de rede ou CLP desligado.
  - Hot-reload dinâmico do sistema após alterações de configuração sem necessidade de reiniciar o processo.
  - Logs no terminal otimizados, emitindo atualizações apenas quando há mudança de valor ou alteração de qualidade.

---

## 🛠️ Tecnologias Utilizadas

**Backend:**
- [Go (Golang)](https://go.dev/)
- [Fiber v2](https://gofiber.io/) - Servidor HTTP de alta performance e estáticos.
- [Fiber WebSockets](https://docs.gofiber.io/api/middleware/websocket/) - Streaming de tags em tempo real.
- [gos7](https://github.com/robinson/gos7) - Driver de comunicação Siemens S7.
- [gologix](https://github.com/danomagnum/gologix) - Driver de comunicação Rockwell CIP.

**Frontend:**
- HTML5 semântico com diálogos nativos (`<dialog>`), evitando sobreposição ou bugs de backdrop.
- CSS3 moderno com tema escuro industrial, layout flexbox/grid e sem frameworks externos.
- Vanilla JavaScript com roteamento por hash (`#dashboard`, `#devices`, `#tags`, `#network`) e consumo nativo de WebSocket e APIs REST.

---

## 📂 Estrutura do Projeto

```text
clpMonitor/
├── build_arm.bat           # Script para compilação cruzada para Linux ARMv7
├── cmd/
│   └── server/
│       └── main.go         # Inicialização do storage, workers e servidor Fiber
├── gateway_config.json     # Armazenamento persistente das configurações
├── internal/
│   ├── api/                # Rotas REST (/api), Hub WebSocket e testes unitários
│   ├── models/             # Modelos de dados (Device, Tag)
│   ├── netmgr/             # Gerenciamento de interfaces de rede (sysfs/ip)
│   ├── plc/                # Drivers de comunicação (S7Driver, RockwellDriver)
│   ├── storage/            # Persistência thread-safe em JSON
│   └── worker/             # PollingWorkers individuais e orquestrador Manager
├── web/                    # Front-end estático servido pelo Fiber
│   ├── css/
│   │   └── style.css       # Estilos da interface
│   ├── js/
│   │   └── app.js          # Lógica de interface, WebSocket e chamadas REST
│   └── index.html          # SPA unificada
├── go.mod
└── go.sum
```

---

## 🚀 Como Executar

### 1. Executando Localmente (Windows/Linux)
Certifique-se de ter o Go instalado (1.21+):

```bash
go run ./cmd/server
```

Acesse no navegador:
```text
http://localhost:8080
```

### 2. Compilação para Embarcados (Linux ARMv7)
Para compilar para plataformas Linux embarcadas (ex: Raspberry Pi, Toradex, BeagleBone, etc.):

- **Via Script:** Dê dois cliques em `build_arm.bat`.
- **Ou via Terminal:**
  ```powershell
  $env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="arm"; $env:GOARM="7"
  go build -ldflags="-s -w" -o clpMonitor cmd/server/main.go
  ```

Copie o executável `clpMonitor` e a pasta `web/` para o dispositivo de destino e execute:
```bash
chmod +x clpMonitor
sudo ./clpMonitor
```
*(O uso de `sudo` é necessário caso deseje utilizar o gerenciamento de interfaces de rede no Linux).*
