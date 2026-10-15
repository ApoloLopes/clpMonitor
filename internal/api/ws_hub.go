package api

import (
	"log"
	"sync"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/gofiber/websocket/v2"
)

// WSHub gerencia as conexões WebSocket ativas e distribui os dados
type WSHub struct {
	clients    map[*websocket.Conn]bool // Mapa de clientes conectados
	Broadcast  chan []models.Tag        // Canal que recebe dados do Worker
	register   chan *websocket.Conn     // Canal para registrar novas conexões
	unregister chan *websocket.Conn     // Canal para remover conexões caídas
	mu         sync.Mutex               // Protege o mapa de clientes contra concorrência
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*websocket.Conn]bool),
		Broadcast:  make(chan []models.Tag),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run inicia o loop principal do Hub (deve rodar em uma goroutine)
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Println("Novo cliente Web conectado!")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				log.Println("Cliente Web desconectado.")
			}
			h.mu.Unlock()

		case data := <-h.Broadcast:
			// Quando chegam dados do CLP, envia para todos os navegadores
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteJSON(data)
				if err != nil {
					// Se der erro (ex: fechou a aba), remove o cliente
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// HandleWS é a rota do Fiber que faz o upgrade de HTTP para WebSocket
func (h *WSHub) HandleWS(c *websocket.Conn) {
	h.register <- c
	defer func() {
		h.unregister <- c
	}()

	// Loop infinito lendo mensagens do cliente para manter a conexão viva
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}
