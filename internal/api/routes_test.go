package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ApoloLopes/clpMonitor/internal/storage"
	"github.com/gofiber/fiber/v2"
)

func setup(t *testing.T) (*fiber.App, *storage.JSONStorage, *int) {
	t.Helper()
	store, err := storage.NewJSONStorage(filepath.Join(t.TempDir(), "cfg.json"))
	if err != nil {
		t.Fatal(err)
	}
	reloads := 0
	app := fiber.New(fiber.Config{UnescapePath: true})
	SetupRoutes(app, store, func() { reloads++ }, func() map[string]DeviceStatus { return map[string]DeviceStatus{} })
	return app, store, &reloads
}

func call(t *testing.T, app *fiber.App, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

func TestDevicesAndTagsCRUD(t *testing.T) {
	app, store, reloads := setup(t)

	steps := []struct {
		name, method, path, body string
		want                     int
	}{
		{"device proto inválido", "POST", "/api/devices", `{"id":"D1","name":"x","protocol":"foo","ip_address":"1.2.3.4"}`, 400},
		{"device ip inválido", "POST", "/api/devices", `{"id":"D1","name":"x","protocol":"s7","ip_address":"abc"}`, 400},
		{"device s7", "POST", "/api/devices", `{"id":"D1","name":"Extrusora","protocol":"s7","ip_address":"127.0.0.1","rack":0,"slot":1}`, 201},
		{"device cip", "POST", "/api/devices", `{"id":"D2","name":"Forno","protocol":"cip","ip_address":"127.0.0.2"}`, 201},
		{"device duplicado", "POST", "/api/devices", `{"id":"D1","name":"x","protocol":"s7","ip_address":"127.0.0.1"}`, 409},
		{"device update", "PUT", "/api/devices/D1", `{"name":"Extrusora 2","protocol":"s7","ip_address":"127.0.0.1","rack":0,"slot":2}`, 200},
		{"device update 404", "PUT", "/api/devices/NOPE", `{"name":"x","protocol":"s7","ip_address":"127.0.0.1"}`, 404},
		{"tag sem device", "POST", "/api/tags", `{"id":"T1","name":"a","device_id":"NOPE","data_type":"REAL"}`, 400},
		{"tag tipo inválido", "POST", "/api/tags", `{"id":"T1","name":"a","device_id":"D1","data_type":"STR"}`, 400},
		{"tag bit inválido", "POST", "/api/tags", `{"id":"T1","name":"a","device_id":"D1","data_type":"BOOL","bit":9}`, 400},
		{"tag cip sem address", "POST", "/api/tags", `{"id":"T1","name":"a","device_id":"D2","data_type":"REAL"}`, 400},
		{"tag real", "POST", "/api/tags", `{"id":"T1","name":"Temp","device_id":"D1","data_type":"REAL","db_number":1,"offset":4}`, 201},
		{"tag bool s7", "POST", "/api/tags", `{"id":"T2","name":"Motor","device_id":"D1","data_type":"BOOL","db_number":1,"offset":0,"bit":3}`, 201},
		{"tag bool cip", "POST", "/api/tags", `{"id":"T3","name":"Valv","device_id":"D2","data_type":"BOOL","address":"Valvula"}`, 201},
		{"tag duplicada", "POST", "/api/tags", `{"id":"T1","name":"a","device_id":"D1","data_type":"REAL"}`, 409},
		{"tag update", "PUT", "/api/tags/T2", `{"name":"Motor Lig","device_id":"D1","data_type":"BOOL","db_number":1,"offset":0,"bit":4}`, 200},
		{"tag delete", "DELETE", "/api/tags/T3", ``, 200},
		{"tag delete 404", "DELETE", "/api/tags/T3", ``, 404},
	}
	for _, s := range steps {
		if code, body := call(t, app, s.method, s.path, s.body); code != s.want {
			t.Errorf("%s: esperado %d, obtido %d (%s)", s.name, s.want, code, body)
		}
	}

	cfg := store.GetConfig()
	if len(cfg.Devices) != 2 || len(cfg.Tags) != 2 {
		t.Fatalf("estado inesperado: %d devices, %d tags: %+v", len(cfg.Devices), len(cfg.Tags), cfg.Tags)
	}
	for _, tg := range cfg.Tags {
		if tg.ID == "T2" && (tg.Bit != 4 || tg.Name != "Motor Lig") {
			t.Errorf("update de tag não persistiu: %+v", tg)
		}
	}

	// Excluir device remove as tags em cascata
	if code, _ := call(t, app, "DELETE", "/api/devices/D1", ``); code != 200 {
		t.Errorf("delete device: %d", code)
	}
	cfg = store.GetConfig()
	if len(cfg.Devices) != 1 || len(cfg.Tags) != 0 {
		t.Errorf("cascata falhou: %d devices, %d tags", len(cfg.Devices), len(cfg.Tags))
	}
	if *reloads == 0 {
		t.Error("hot-reload nunca foi disparado")
	}

	code, body := call(t, app, "GET", "/api/config", ``)
	var out map[string]any
	if code != 200 || json.Unmarshal([]byte(body), &out) != nil {
		t.Errorf("GET /api/config: %d %s", code, body)
	}
}

func TestNetworkEndpoints(t *testing.T) {
	app, _, _ := setup(t)
	code, body := call(t, app, "GET", "/api/network/interfaces", ``)
	if code != 200 {
		t.Fatalf("list: %d %s", code, body)
	}
	var info struct {
		Writable   bool `json:"writable"`
		Interfaces []struct {
			Name string `json:"name"`
		} `json:"interfaces"`
	}
	if err := json.Unmarshal([]byte(body), &info); err != nil || len(info.Interfaces) == 0 {
		t.Fatalf("resposta inválida: %v %s", err, body)
	}

	// Interface inexistente nunca deve chegar ao comando `ip`
	if code, _ := call(t, app, "POST", "/api/network/interfaces/nao-existe0/state", `{"up":true}`); code != 400 {
		t.Errorf("interface inexistente: esperado 400, obtido %d", code)
	}
	name := url.PathEscape(info.Interfaces[0].Name)
	if code, _ := call(t, app, "POST", "/api/network/interfaces/"+name+"/addresses", `{"address":"abc"}`); code != 400 {
		t.Errorf("CIDR inválido: esperado 400, obtido %d", code)
	}
	if !info.Writable {
		if code, _ := call(t, app, "POST", "/api/network/interfaces/"+name+"/state", `{"up":true}`); code != 501 {
			t.Errorf("fora do Linux: esperado 501, obtido %d", code)
		}
	}
}
