package main

import (
	"fmt"
	"log"

	"github.com/ApoloLopes/clpMonitor/internal/api"
	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/storage"
	"github.com/ApoloLopes/clpMonitor/internal/worker"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// 1. Armazenamento
	store, err := storage.NewJSONStorage("gateway_config.json")
	if err != nil {
		log.Fatalf("Erro ao iniciar armazenamento: %v", err)
	}

	cfg := store.GetConfig()
	if len(cfg.Devices) == 0 {
		device := models.Device{ID: "PLC_01", Name: "Extrusora", Protocol: "s7", IPAddress: "127.0.0.1", Rack: 0, Slot: 2}
		store.AddDevice(device)
		store.AddTag(models.Tag{ID: "T1", Name: "Temperatura", DBNumber: 1, Offset: 4, DataType: "REAL"})
	}

	// 2. Hub WebSockets
	hub := api.NewWSHub()
	go hub.Run()

	// 3. Orquestrador de Workers
	// Passamos o storage e o hub para ele saber de onde ler e para onde enviar
	manager := worker.NewManager(store, hub)

	// Ligamos os motores pela primeira vez!
	manager.StartAll()

	// 4. Servidor Web
	app := fiber.New()

	// Passamos o manager.Reload como callback para a API
	api.SetupRoutes(app, store, manager.Reload)

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(hub.HandleWS))
	app.Static("/", "./web")

	fmt.Println("🚀 Servidor PLC Gateway a correr na porta 8080...")
	log.Fatal(app.Listen(":8080"))
}
