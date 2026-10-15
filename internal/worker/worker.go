package worker

import (
	"context"
	"log"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/plc"
)

// PollingWorker gerencia o ciclo de leitura de um dispositivo específico
type PollingWorker struct {
	Device   models.Device
	Driver   plc.Driver
	Tags     []models.Tag
	Interval time.Duration
	// DataChannel é o canal por onde enviaremos os dados lidos para o resto do sistema
	DataChannel chan []models.Tag
}

func NewPollingWorker(device models.Device, driver plc.Driver, tags []models.Tag, intervalMs int) *PollingWorker {
	return &PollingWorker{
		Device:      device,
		Driver:      driver,
		Tags:        tags,
		Interval:    time.Duration(intervalMs) * time.Millisecond,
		DataChannel: make(chan []models.Tag),
	}
}

// Start inicia o loop infinito do worker. Deve ser chamado com a palavra-chave 'go'
func (w *PollingWorker) Start(ctx context.Context) {
	log.Printf("[Worker %s] Iniciando polling a cada %v", w.Device.Name, w.Interval)

	// Ticker é um relógio que "apita" a cada intervalo definido
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Se o contexto for cancelado (ex: o usuário fechou o programa), o worker para graciosamente
			log.Printf("[Worker %s] Parando execução...", w.Device.Name)
			w.Driver.Disconnect()
			return

		case <-ticker.C:
			// A cada "apito" do relógio, fazemos a leitura
			err := w.Driver.ReadTags(&w.Tags)
			if err != nil {
				log.Printf("[Worker %s] Erro de leitura: %v", w.Device.Name, err)
			}

			// Cria uma cópia das tags para enviar pelo canal (evita condição de corrida)
			tagsCopy := make([]models.Tag, len(w.Tags))
			copy(tagsCopy, w.Tags)

			// Envia os dados lidos pelo canal de forma não-bloqueante
			select {
			case w.DataChannel <- tagsCopy:
				// Dados enviados com sucesso para quem estiver escutando
			default:
				// Se ninguém estiver escutando o canal, descarta a leitura para não travar o worker
			}
		}
	}
}
