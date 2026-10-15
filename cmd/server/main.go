package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ApoloLopes/clpMonitor/internal/api"
	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/plc"
	"github.com/ApoloLopes/clpMonitor/internal/storage"
	"github.com/ApoloLopes/clpMonitor/internal/worker"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// 1. Inicializa o Armazenamento Persistente
	store, err := storage.NewJSONStorage("gateway_config.json")
	if err != nil {
		log.Fatalf("Erro ao iniciar armazenamento: %v", err)
	}

	cfg := store.GetConfig()

	// Apenas para conveniência no primeiro arranque: se não houver dispositivos, cria um por defeito
	if len(cfg.Devices) == 0 {
		fmt.Println("Ficheiro JSON vazio. A injetar configuração inicial do simulador...")

		device := models.Device{ID: "PLC_01", Name: "Extrusora", Protocol: "s7", IPAddress: "127.0.0.1", Rack: 0, Slot: 2}
		store.AddDevice(device)

		store.AddTag(models.Tag{ID: "T1", Name: "Temperatura", DBNumber: 1, Offset: 4, DataType: "REAL"})
		store.AddTag(models.Tag{ID: "T2", Name: "Motor_Ligado", DBNumber: 1, Offset: 0, Bit: 0, DataType: "BOOL"})
		store.AddTag(models.Tag{ID: "T3", Name: "Velocidade", DBNumber: 1, Offset: 2, DataType: "INT"})

		cfg = store.GetConfig() // Atualiza a variável com a configuração recém-criada
	}

	// 2. Inicializa o Hub WebSocket
	hub := api.NewWSHub()
	go hub.Run()

	// 3. Inicia os Workers para todos os dispositivos configurados no JSON
	// Numa arquitetura real, teríamos um gestor de workers para adicionar/remover dinamicamente sem reiniciar
	for _, dev := range cfg.Devices {
		// Filtra as tags que pertencem apenas a este dispositivo
		// (Para já usamos todas as tags para o PLC único, mas no futuro precisarás de um campo DeviceID na estrutura Tag)

		var driver plc.Driver = plc.NewS7Driver(dev)
		if err := driver.Connect(context.Background()); err != nil {
			log.Printf("Aviso: Não conectou no CLP %s de imediato: %v\n", dev.Name, err)
		}

		plcWorker := worker.NewPollingWorker(dev, driver, cfg.Tags, 500)
		go plcWorker.Start(context.Background())

		// Goroutine ponte para este worker específico
		go func(w *worker.PollingWorker) {
			for data := range w.DataChannel {
				hub.Broadcast <- data
			}
		}(plcWorker)
	}

	// 4. Configuração do Servidor Web
	app := fiber.New()

	// Regista as rotas REST da API
	api.SetupRoutes(app, store)

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(hub.HandleWS))
	app.Static("/", "./web")

	fmt.Println("Iniciando servidor web na porta 8080...")
	log.Fatal(app.Listen(":8080"))
}
