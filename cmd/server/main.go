package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ApoloLopes/clpMonitor/internal/api"
	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/plc"
	"github.com/ApoloLopes/clpMonitor/internal/worker"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// 1. Configuração (Hardcoded por enquanto)
	device := models.Device{
		ID: "PLC_01", Name: "Extrusora", Protocol: "s7", IPAddress: "127.0.0.1", Rack: 0, Slot: 2,
	}

	tags := []models.Tag{
		{ID: "T1", Name: "Temperatura", DBNumber: 1, Offset: 4, DataType: "REAL"},
		{ID: "T2", Name: "Motor_Ligado", DBNumber: 1, Offset: 0, Bit: 0, DataType: "BOOL"},
		{ID: "T3", Name: "Velocidade", DBNumber: 1, Offset: 2, DataType: "INT"},
	}

	// 2. Inicializa o Hub WebSocket
	hub := api.NewWSHub()
	go hub.Run()

	// 3. Inicializa o Driver e Worker do CLP
	var driver plc.Driver = plc.NewS7Driver(device)
	if err := driver.Connect(context.Background()); err != nil {
		log.Printf("Aviso: Não conectou no CLP de imediato: %v\n", err)
	}

	plcWorker := worker.NewPollingWorker(device, driver, tags, 500)
	go plcWorker.Start(context.Background())

	// 4. Goroutine ponte: Escuta o Worker e manda para o Hub WebSocket
	go func() {
		for data := range plcWorker.DataChannel {
			// Repassa as tags lidas para todos os clientes WebSocket conectados
			hub.Broadcast <- data
		}
	}()

	// 5. Configuração do Servidor Web (Fiber)
	app := fiber.New()

	// Middleware para garantir que a rota /ws exija Upgrade para WebSocket
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Rota do WebSocket
	app.Get("/ws", websocket.New(hub.HandleWS))

	// Rota para testarmos rapidamente se o servidor subiu
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Servidor PLC Gateway rodando! Acesse via WebSocket em ws://localhost:8080/ws")
	})
	// Serve todos os arquivos da pasta "web" na raiz do site
	app.Static("/", "./web")
	fmt.Println("Iniciando servidor web na porta 8080...")
	log.Fatal(app.Listen(":8080"))
}
