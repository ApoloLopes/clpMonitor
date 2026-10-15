package worker

import (
	"context"
	"log"

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
}

// NewManager cria uma nova instância do gestor
func NewManager(store *storage.JSONStorage, hub *api.WSHub) *Manager {
	return &Manager{
		store: store,
		hub:   hub,
	}
}

// StartAll lê as configurações e liga os workers
func (m *Manager) StartAll() {
	cfg := m.store.GetConfig()

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
			driver = plc.NewRockwellDriver(dev) // <- Novo Driver Rockwell adicionado!
		default:
			log.Printf("[Manager] Aviso: Protocolo desconhecido '%s' no dispositivo %s\n", dev.Protocol, dev.Name)
			continue
		}

		// Tenta conectar...
		if err := driver.Connect(ctx); err != nil {
			log.Printf("[Manager] Aviso: Falha ao conectar no CLP %s: %v\n", dev.Name, err)
		}

		var deviceTags []models.Tag
		for _, t := range cfg.Tags {
			if t.DeviceID == dev.ID {
				deviceTags = append(deviceTags, t)
			}
		}

		// ATENÇÃO: Passa a lista filtrada (deviceTags) em vez da lista global (cfg.Tags)
		w := NewPollingWorker(dev, driver, deviceTags, 500)
		go w.Start(ctx)

		// Goroutine ponte: lê do Worker e envia para o WebSocket
		go func(w *PollingWorker) {
			for {
				select {
				case <-ctx.Done():
					// Se o contexto for cancelado, esta goroutine morre (evita memory leaks)
					return
				case data := <-w.DataChannel:
					m.hub.Broadcast <- data
				}
			}
		}(w)
	}
	log.Println("[Manager] Todos os workers iniciados com sucesso.")
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
	log.Println("[Manager] HOT-RELOAD DISPARADO! A atualizar configurações...")
	m.StopAll()
	m.StartAll()
}
