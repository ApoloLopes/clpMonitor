# 🏭 PLC Gateway - Siemens & Rockwell

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)
![Siemens S7](https://img.shields.io/badge/Protocol-S7_Comm-009999?style=for-the-badge)
![Rockwell CIP](https://img.shields.io/badge/Protocol-CIP_EtherNet/IP-CC0000?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)

Um gateway de comunicação de alta performance desenvolvido em **Go (Golang)** para atuar como ponte entre Controladores Lógicos Programáveis (CLPs) da Siemens e Rockwell e aplicações web. 

O projeto permite a configuração de dispositivos, mapeamento de tags e visualização de dados em tempo real através de um painel rodando no próprio navegador via `localhost`, sem a necessidade de softwares supervisórios pesados.

---

## ✨ Funcionalidades

- **Comunicação Nativa:** Leitura de dados sem necessidade de servidores OPC intermediários.
  - **Siemens:** Suporte a S7-1200, S7-1500, S7-300 via protocolo S7 (biblioteca `gos7`).
  - **Rockwell:** Suporte a ControlLogix e CompactLogix via protocolo CIP (biblioteca `gologix`).
- **Dashboard Web Integrado:** Interface em HTML/JS servida pelo próprio executável em Go para gerenciamento visual.
- **Configuração Dinâmica:** Adição de novos CLPs (IP, Rack, Slot) e Tags (DBs, Merkers, Nomes simbólicos) diretamente pelo browser.
- **Tempo Real (WebSockets):** Atualização instantânea dos valores das variáveis na tela sem *polling* do lado do cliente, garantindo baixa latência no monitoramento.
- **Alta Concorrência:** Cada CLP roda em sua própria *goroutine*, garantindo que falhas de rede ou latência em um equipamento não afetem o *scan* dos demais.

---

## 🛠️ Tecnologias Utilizadas

**Backend:**
- [Go (Golang)](https://go.dev/) - Linguagem principal.
- [Fiber](https://gofiber.io/) - Framework web ultrarrápido para a API REST e servimento de estáticos.
- [Fiber WebSockets](https://docs.gofiber.io/api/middleware/websocket/) - Streaming de dados para o frontend.
- [gos7](https://github.com/robinson/gos7) - Driver S7.
- [gologix](https://github.com/danomagnum/gologix) - Driver EtherNet/IP (CIP).

**Frontend:**
- HTML5, CSS3 e Vanilla JavaScript (API WebSocket nativa).

---

## 📂 Estrutura do Projeto

```text
plc-gateway/
├── cmd/
│   └── server/
│       └── main.go         # Ponto de entrada da aplicação
├── internal/
│   ├── plc/                # Drivers de comunicação (S7, CIP/EtherNet/IP)
│   ├── api/                # Rotas REST para CRUD de CLPs e WebSockets
│   └── models/             # Estruturas de dados (Device, Tag, Config)
├── web/
│   ├── index.html          # Dashboard (Interface do usuário)
│   ├── css/                # Estilos do painel
│   └── js/                 # Lógica WebSocket e integração REST
├── go.mod
└── go.sum