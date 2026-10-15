package api

import (
	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configura todos os endpoints REST da aplicação
func SetupRoutes(app *fiber.App, store *storage.JSONStorage) {
	api := app.Group("/api")

	// Devolve toda a configuração (Dispositivos e Tags)
	api.Get("/config", func(c *fiber.Ctx) error {
		return c.JSON(store.GetConfig())
	})

	// Regista um novo CLP
	api.Post("/devices", func(c *fiber.Ctx) error {
		var device models.Device
		if err := c.BodyParser(&device); err != nil {
			return c.Status(400).JSON(fiber.Map{"erro": "Formato de dispositivo inválido"})
		}

		if err := store.AddDevice(device); err != nil {
			return c.Status(500).JSON(fiber.Map{"erro": "Falha ao gravar dispositivo"})
		}

		return c.Status(201).JSON(device)
	})

	// Regista uma nova Tag
	api.Post("/tags", func(c *fiber.Ctx) error {
		var tag models.Tag
		if err := c.BodyParser(&tag); err != nil {
			return c.Status(400).JSON(fiber.Map{"erro": "Formato de tag inválido"})
		}

		if err := store.AddTag(tag); err != nil {
			return c.Status(500).JSON(fiber.Map{"erro": "Falha ao gravar tag"})
		}

		return c.Status(201).JSON(tag)
	})
}
