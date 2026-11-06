package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/api"
	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/plc"
	"github.com/ApoloLopes/clpMonitor/internal/storage"
)

// Manager orquestra o ciclo de vida de todos os workers de polling
type Manager struct {
	store      *storage.JSONStorage
	hub        *api.WSHub
	cancelFunc context.CancelFunc // Função para cancelar os workers em execução

	statusMu sync.RWMutex
	reloadMu sync.Mutex
	statuses map[string]api.DeviceStatus
}

// NewManager cria uma nova instância do gestor
func NewManager(store *storage.JSONStorage, hub *api.WSHub) *Manager {
	return &Manager{
		store:    store,
		hub:      hub,
		statuses: map[string]api.DeviceStatus{},
	}
}

func (m *Manager) setStatus(deviceID, state, msg string) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.statuses[deviceID] = api.DeviceStatus{
		DeviceID:   deviceID,
		State:      state,
		Message:    msg,
		LastUpdate: time.Now(),
	}
}

// setStatusIfCurrent só grava se o contexto (geração de workers) ainda estiver ativo,
// evitando que goroutines antigas sobrescrevam o estado após um hot-reload.
func (m *Manager) setStatusIfCurrent(ctx context.Context, deviceID, state, msg string) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	if ctx.Err() != nil {
		return
	}
	m.statuses[deviceID] = api.DeviceStatus{
		DeviceID:   deviceID,
		State:      state,
		Message:    msg,
		LastUpdate: time.Now(),
	}
}

// GetStatuses devolve o estado de conexão atual de todos os dispositivos
func (m *Manager) GetStatuses() map[string]api.DeviceStatus {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	out := make(map[string]api.DeviceStatus, len(m.statuses))
	for k, v := range m.statuses {
		out[k] = v
	}
	return out
}

// StartAll lê as configurações e liga os workers
func (m *Manager) StartAll() {
	cfg := m.store.GetConfig()

	// Reinicia a tabela de estados (dispositivos removidos desaparecem)
	m.statusMu.Lock()
	m.statuses = map[string]api.DeviceStatus{}
	m.statusMu.Unlock()

	// Cria um novo contexto derivado para esta "geração" de workers
	// Quando chamarmos cancel(), todas as goroutines que usam este ctx vão parar
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	for _, dev := range cfg.Devices {
		// Instancia o driver correto consoante o protocolo
		var driver plc.Driver

		switch dev.Protocol {
		case "s7":
			driver = plc.NewS7Driver(dev)
		case "cip":
			driver = plc.NewRockwellDriver(dev)
		default:
			log.Printf("[Manager] Aviso: Protocolo desconhecido '%s' no dispositivo %s\n", dev.Protocol, dev.Name)
			m.setStatus(dev.ID, "offline", "Protocolo desconhecido: "+dev.Protocol)
			continue
		}

		m.setStatus(dev.ID, "connecting", "A conectar...")

		var deviceTags []models.Tag
		for _, t := range cfg.Tags {
			if t.DeviceID == dev.ID {
				deviceTags = append(deviceTags, t)
			}
		}

		w := NewPollingWorker(dev, driver, deviceTags, 500)

		// Conecta e inicia o polling numa goroutine própria: um CLP lento/offline
		// não bloqueia a API nem os demais dispositivos.
		go func(dev models.Device, driver plc.Driver, w *PollingWorker) {
			cctx, ccancel := context.WithTimeout(ctx, 5*time.Second)
			err := driver.Connect(cctx)
			ccancel()
			if err != nil {
				log.Printf("[Manager] Aviso: Falha ao conectar no CLP %s: %v\n", dev.Name, err)
				m.setStatusIfCurrent(ctx, dev.ID, "offline", err.Error())
			}
			if ctx.Err() != nil {
				driver.Disconnect()
				return
			}
			w.Start(ctx)
		}(dev, driver, w)

		// Goroutine ponte: lê do Worker, atualiza o estado e envia para o WebSocket
		go func(w *PollingWorker) {
			for {
				select {
				case <-ctx.Done():
					return
				case data := <-w.DataChannel:
					m.updateStatusFromData(ctx, w.Device, data)
					m.hub.Broadcast <- data
				}
			}
		}(w)
	}
	log.Println("[Manager] Todos os workers iniciados com sucesso.")
}

// updateStatusFromData deduz o estado de conexão a partir da qualidade das leituras
func (m *Manager) updateStatusFromData(ctx context.Context, dev models.Device, data []models.Tag) {
	if len(data) == 0 {
		m.setStatusIfCurrent(ctx, dev.ID, "unknown", "Sem tags configuradas: estado não verificável")
		return
	}
	good := 0
	for _, t := range data {
		if t.Quality {
			good++
		}
	}
	switch {
	case good == 0:
		m.setStatusIfCurrent(ctx, dev.ID, "offline", "Nenhuma tag lida com sucesso")
	case good < len(data):
		m.setStatusIfCurrent(ctx, dev.ID, "online", "Online (algumas tags com falha de leitura)")
	default:
		m.setStatusIfCurrent(ctx, dev.ID, "online", "Comunicação OK")
	}
}

// StopAll cancela o contexto atual, parando todos os workers e conexões
func (m *Manager) StopAll() {
	if m.cancelFunc != nil {
		log.Println("[Manager] A enviar sinal de paragem para os workers antigos...")
		m.cancelFunc()
	}
}

// Reload executa o Hot-Reload (Para tudo e inicia novamente com dados atualizados)
func (m *Manager) Reload() {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()
	log.Println("[Manager] HOT-RELOAD DISPARADO! A atualizar configurações...")
	m.StopAll()
	m.StartAll()
}
