package plc

import (
	"context"
	"fmt"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/danomagnum/gologix"
)

// Garante em tempo de compilação que RockwellDriver implementa plc.Driver
var _ Driver = (*RockwellDriver)(nil)

type RockwellDriver struct {
	device models.Device
	client *gologix.Client
}

func NewRockwellDriver(device models.Device) *RockwellDriver {
	return &RockwellDriver{
		device: device,
	}
}

func (r *RockwellDriver) Connect(ctx context.Context) error {
	// A biblioteca gologix geralmente assume comunicação com a CPU no Slot 0 (padrão CompactLogix).
	// Para ControlLogix em slots diferentes, o IP pode ser formatado com o caminho CIP.
	r.client = gologix.NewClient(r.device.IPAddress)

	// Canais para gerir o timeout através do Context
	errChan := make(chan error, 1)

	go func() {
		errChan <- r.client.Connect()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout ou cancelamento ao conectar no Rockwell %s", r.device.IPAddress)
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("falha de conexão CIP: %w", err)
		}
	}

	return nil
}

func (r *RockwellDriver) Disconnect() error {
	if r.client != nil {
		return r.client.Disconnect()
	}
	return nil
}

func (r *RockwellDriver) ReadTags(tags *[]models.Tag) error {
	for i := range *tags {
		tag := &(*tags)[i]
		tag.Timestamp = time.Now()

		var err error

		// O protocolo CIP necessita que passemos a variável com o tipo correto
		// para que ele possa fazer a desserialização dos bytes do pacote de resposta
		switch tag.DataType {
		case "BOOL":
			var val bool
			err = r.client.Read(tag.Address, &val)
			tag.Value = val
		case "INT":
			var val int16
			err = r.client.Read(tag.Address, &val)
			tag.Value = val
		case "REAL":
			var val float32
			err = r.client.Read(tag.Address, &val)
			tag.Value = val
		default:
			tag.Quality = false
			continue
		}

		if err != nil {
			tag.Quality = false
		} else {
			tag.Quality = true
		}
	}

	return nil
}
