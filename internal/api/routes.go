package api

import (
	"errors"
	"net"
	"strings"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/netmgr"
	"github.com/ApoloLopes/clpMonitor/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// DeviceStatus descreve o estado de conexão de um dispositivo
type DeviceStatus struct {
	DeviceID   string    `json:"device_id"`
	State      string    `json:"state"` // "online", "offline", "connecting", "unknown"
	Message    string    `json:"message"`
	LastUpdate time.Time `json:"last_update"`
}

func erro(c *fiber.Ctx, code int, msg string) error {
	return c.Status(code).JSON(fiber.Map{"erro": msg})
}

// param devolve uma cópia do parâmetro de rota. O Fiber reutiliza os buffers da
// requisição, então valores guardados (ex.: no storage) precisam ser copiados.
func param(c *fiber.Ctx, key string) string {
	return strings.Clone(c.Params(key))
}

func validateDevice(d *models.Device) error {
	d.ID = strings.TrimSpace(d.ID)
	d.Name = strings.TrimSpace(d.Name)
	d.IPAddress = strings.TrimSpace(d.IPAddress)
	if d.ID == "" || d.Name == "" || d.IPAddress == "" {
		return errors.New("ID, nome e endereço IP são obrigatórios")
	}
	if d.Protocol != "s7" && d.Protocol != "cip" {
		return errors.New("protocolo deve ser 's7' ou 'cip'")
	}
	if net.ParseIP(d.IPAddress) == nil {
		return errors.New("endereço IP inválido")
	}
	return nil
}

func validateTag(t *models.Tag, store *storage.JSONStorage) error {
	t.ID = strings.TrimSpace(t.ID)
	t.Name = strings.TrimSpace(t.Name)
	t.Address = strings.TrimSpace(t.Address)
	if t.ID == "" || t.Name == "" {
		return errors.New("ID e nome são obrigatórios")
	}
	var dev *models.Device
	for _, d := range store.GetConfig().Devices {
		if d.ID == t.DeviceID {
			d := d
			dev = &d
			break
		}
	}
	if dev == nil {
		return errors.New("dispositivo não encontrado")
	}
	switch t.DataType {
	case "BOOL", "INT", "REAL":
	default:
		return errors.New("tipo de dado deve ser BOOL, INT ou REAL")
	}
	if t.Bit < 0 || t.Bit > 7 {
		return errors.New("bit deve estar entre 0 e 7")
	}
	if dev.Protocol == "cip" && t.Address == "" {
		return errors.New("o endereço (nome simbólico) é obrigatório para dispositivos Rockwell")
	}
	if dev.Protocol == "s7" && (t.DBNumber < 0 || t.Offset < 0) {
		return errors.New("DB e offset devem ser positivos")
	}
	return nil
}

// SetupRoutes registra a API REST. onConfigChange é chamado após cada alteração de configuração.
func SetupRoutes(app *fiber.App, store *storage.JSONStorage, onConfigChange func(), getStatuses func() map[string]DeviceStatus) {
	api := app.Group("/api")

	api.Get("/config", func(c *fiber.Ctx) error {
		return c.JSON(store.GetConfig())
	})

	// ---------- Dispositivos ----------
	api.Get("/devices/status", func(c *fiber.Ctx) error {
		return c.JSON(getStatuses())
	})

	api.Post("/devices", func(c *fiber.Ctx) error {
		var device models.Device
		if err := c.BodyParser(&device); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		if err := validateDevice(&device); err != nil {
			return erro(c, 400, err.Error())
		}
		for _, d := range store.GetConfig().Devices {
			if d.ID == device.ID {
				return erro(c, 409, "Já existe um dispositivo com este ID")
			}
		}
		if err := store.AddDevice(device); err != nil {
			return erro(c, 500, "Falha ao gravar")
		}
		onConfigChange()
		return c.Status(201).JSON(device)
	})

	api.Put("/devices/:id", func(c *fiber.Ctx) error {
		var device models.Device
		if err := c.BodyParser(&device); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		device.ID = param(c, "id")
		if err := validateDevice(&device); err != nil {
			return erro(c, 400, err.Error())
		}
		if err := store.UpdateDevice(device); err != nil {
			return erro(c, 404, err.Error())
		}
		onConfigChange()
		return c.JSON(device)
	})

	api.Delete("/devices/:id", func(c *fiber.Ctx) error {
		id := param(c, "id")
		if err := store.DeleteDevice(id); err != nil {
			return erro(c, 404, err.Error())
		}
		onConfigChange()
		return c.JSON(fiber.Map{"mensagem": "Dispositivo e tags associadas excluídos", "id": id})
	})

	// ---------- Tags ----------
	api.Post("/tags", func(c *fiber.Ctx) error {
		var tag models.Tag
		if err := c.BodyParser(&tag); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		if err := validateTag(&tag, store); err != nil {
			return erro(c, 400, err.Error())
		}
		for _, t := range store.GetConfig().Tags {
			if t.ID == tag.ID {
				return erro(c, 409, "Já existe uma tag com este ID")
			}
		}
		if err := store.AddTag(tag); err != nil {
			return erro(c, 500, "Falha ao gravar")
		}
		onConfigChange()
		return c.Status(201).JSON(tag)
	})

	api.Put("/tags/:id", func(c *fiber.Ctx) error {
		var tag models.Tag
		if err := c.BodyParser(&tag); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		tag.ID = param(c, "id")
		if err := validateTag(&tag, store); err != nil {
			return erro(c, 400, err.Error())
		}
		if err := store.UpdateTag(tag); err != nil {
			return erro(c, 404, err.Error())
		}
		onConfigChange()
		return c.JSON(tag)
	})

	api.Delete("/tags/:id", func(c *fiber.Ctx) error {
		id := param(c, "id")
		if err := store.DeleteTag(id); err != nil {
			return erro(c, 404, err.Error())
		}
		onConfigChange()
		return c.JSON(fiber.Map{"mensagem": "Tag excluída com sucesso", "id": id})
	})

	// ---------- Rede (interfaces do host) ----------
	netErr := func(c *fiber.Ctx, err error) error {
		if errors.Is(err, netmgr.ErrUnsupported) {
			return erro(c, 501, err.Error())
		}
		return erro(c, 400, err.Error())
	}

	api.Get("/network/interfaces", func(c *fiber.Ctx) error {
		info, err := netmgr.List()
		if err != nil {
			return erro(c, 500, err.Error())
		}
		return c.JSON(info)
	})

	api.Post("/network/interfaces/:name/state", func(c *fiber.Ctx) error {
		var body struct {
			Up bool `json:"up"`
		}
		if err := c.BodyParser(&body); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		if err := netmgr.SetState(param(c, "name"), body.Up); err != nil {
			return netErr(c, err)
		}
		return c.JSON(fiber.Map{"ok": true})
	})

	// Define um IPv4 estático único (substitui os atuais)
	api.Put("/network/interfaces/:name/ipv4", func(c *fiber.Ctx) error {
		var body struct {
			Address string `json:"address"`
			Gateway string `json:"gateway"`
		}
		if err := c.BodyParser(&body); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		if err := netmgr.SetStatic(param(c, "name"), body.Address, body.Gateway); err != nil {
			return netErr(c, err)
		}
		return c.JSON(fiber.Map{"ok": true})
	})

	// Adiciona um IP secundário
	api.Post("/network/interfaces/:name/addresses", func(c *fiber.Ctx) error {
		var body struct {
			Address string `json:"address"`
			Gateway string `json:"gateway"`
		}
		if err := c.BodyParser(&body); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		if err := netmgr.AddAddress(param(c, "name"), body.Address, body.Gateway); err != nil {
			return netErr(c, err)
		}
		return c.Status(201).JSON(fiber.Map{"ok": true})
	})

	api.Delete("/network/interfaces/:name/addresses", func(c *fiber.Ctx) error {
		if err := netmgr.RemoveAddress(param(c, "name"), strings.Clone(c.Query("address"))); err != nil {
			return netErr(c, err)
		}
		return c.JSON(fiber.Map{"ok": true})
	})

	// Edita um endereço IP existente (em tempo real)
	api.Put("/network/interfaces/:name/addresses", func(c *fiber.Ctx) error {
		var body struct {
			OldAddress string `json:"old_address"`
			NewAddress string `json:"new_address"`
			Gateway    string `json:"gateway"`
		}
		if err := c.BodyParser(&body); err != nil {
			return erro(c, 400, "Formato inválido")
		}
		if err := netmgr.EditAddress(param(c, "name"), body.OldAddress, body.NewAddress, body.Gateway); err != nil {
			return netErr(c, err)
		}
		return c.JSON(fiber.Map{"ok": true})
	})
}
