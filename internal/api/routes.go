package api

import (
	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes agora recebe uma função de callback (onConfigChange)
func SetupRoutes(app *fiber.App, store *storage.JSONStorage, onConfigChange func()) {
	api := app.Group("/api")

	api.Get("/config", func(c *fiber.Ctx) error {
		return c.JSON(store.GetConfig())
	})

	api.Post("/devices", func(c *fiber.Ctx) error {
		var device models.Device
		if err := c.BodyParser(&device); err != nil {
			return c.Status(400).JSON(fiber.Map{"erro": "Formato inválido"})
		}
		if err := store.AddDevice(device); err != nil {
			return c.Status(500).JSON(fiber.Map{"erro": "Falha ao gravar"})
		}

		// Dispara o Hot-Reload!
		onConfigChange()

		return c.Status(201).JSON(device)
	})

	api.Post("/tags", func(c *fiber.Ctx) error {
		var tag models.Tag
		if err := c.BodyParser(&tag); err != nil {
			return c.Status(400).JSON(fiber.Map{"erro": "Formato inválido"})
		}
		if err := store.AddTag(tag); err != nil {
			return c.Status(500).JSON(fiber.Map{"erro": "Falha ao gravar"})
		}

		// Dispara o Hot-Reload!
		onConfigChange()

		return c.Status(201).JSON(tag)
	})
}
