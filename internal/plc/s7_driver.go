package plc

import (
	"context"
	"fmt"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/robinson/gos7"
)

// Garantir em tempo de compilação que S7Driver implementa plc.Driver
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

	// Canais para gerenciar o timeout da conexão usando o Context
	errChan := make(chan error, 1)

	go func() {
		errChan <- s.handler.Connect()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout ou cancelamento ao conectar no S7 %s", s.device.IPAddress)
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
	}
	return nil
}

func (s *S7Driver) ReadTags(tags *[]models.Tag) error {
	var helper gos7.Helper

	// Itera sobre as tags por referência para atualizar seus valores originais
	for i := range *tags {
		tag := &(*tags)[i]
		tag.Timestamp = time.Now()

		// Determina o tamanho do buffer baseado no tipo de dado
		var size int
		switch tag.DataType {
		case "BOOL":
			size = 1
		case "INT":
			size = 2 // Int no S7 tem 16 bits (2 bytes)
		case "REAL":
			size = 4 // Real no S7 tem 32 bits (4 bytes)
		default:
			tag.Quality = false
			continue
		}

		buffer := make([]byte, size)
		err := s.client.AGReadDB(tag.DBNumber, tag.Offset, size, buffer)

		if err != nil {
			tag.Quality = false
			continue
		}

		tag.Quality = true

		// Conversão dinâmica
		switch tag.DataType {
		case "BOOL":
			// Converte tag.Bit para uint e recebe o retorno diretamente
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
