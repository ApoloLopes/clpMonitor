package plc

import (
	"context"
	"fmt"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/robinson/gos7"
)

var _ Driver = (*S7Driver)(nil)

type S7Driver struct {
	device  models.Device
	client  gos7.Client
	handler *gos7.TCPClientHandler
}

func NewS7Driver(device models.Device) *S7Driver {
	return &S7Driver{
		device: device,
	}
}

func (s *S7Driver) Connect(ctx context.Context) error {
	s.handler = gos7.NewTCPClientHandler(s.device.IPAddress, s.device.Rack, s.device.Slot)
	s.handler.Timeout = 5 * time.Second
	s.handler.IdleTimeout = 30 * time.Second

	errChan := make(chan error, 1)

	go func() {
		errChan <- s.handler.Connect()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout ao conectar no S7 %s", s.device.IPAddress)
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("falha ao conectar: %w", err)
		}
	}

	s.client = gos7.NewClient(s.handler)
	return nil
}

func (s *S7Driver) Disconnect() error {
	if s.handler != nil {
		s.handler.Close()
		s.handler = nil
	}
	// FUNDAMENTAL: Avisar a memória que o cliente morreu
	s.client = nil
	return nil
}

func (s *S7Driver) ReadTags(tags *[]models.Tag) error {
	// 1. AUTO-RECONNECT: Se o cliente for nulo (iniciou offline ou a rede caiu antes)
	if s.client == nil {
		// Dá apenas 2 segundos para tentar reconectar e não bloquear os outros Workers
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := s.Connect(ctx); err != nil {
			// Continua offline: marca todas as tags como má qualidade e aborta a leitura
			for i := range *tags {
				(*tags)[i].Quality = false
				(*tags)[i].Timestamp = time.Now()
			}
			return fmt.Errorf("clp offline, a aguardar reconexão na próxima varredura")
		}
	}

	var helper gos7.Helper
	var connectionDropped bool

	for i := range *tags {
		tag := &(*tags)[i]
		tag.Timestamp = time.Now()

		// Se a conexão caiu a meio do loop, já nem tenta ler as restantes
		if connectionDropped || s.client == nil {
			tag.Quality = false
			continue
		}

		var size int
		switch tag.DataType {
		case "BOOL":
			size = 1
		case "INT":
			size = 2
		case "REAL":
			size = 4
		default:
			tag.Quality = false
			continue
		}

		buffer := make([]byte, size)

		// 2. LÊ DO CLP
		err := s.client.AGReadDB(tag.DBNumber, tag.Offset, size, buffer)

		// 3. DETETA QUEDA DE REDE
		if err != nil {
			tag.Quality = false
			connectionDropped = true

			// Destrói o cliente imediatamente. Isto obriga o código a passar pelo
			// passo de Auto-Reconnect no início da próxima vez que esta função for chamada.
			s.Disconnect()
			continue
		}

		tag.Quality = true

		switch tag.DataType {
		case "BOOL":
			value := helper.GetBoolAt(buffer[0], uint(tag.Bit))
			tag.Value = value
		case "INT":
			var value int16
			helper.GetValueAt(buffer, 0, &value)
			tag.Value = value
		case "REAL":
			var value float32
			helper.GetValueAt(buffer, 0, &value)
			tag.Value = value
		}
	}

	return nil
}
