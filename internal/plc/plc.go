package plc

import (
	"context"

	"github.com/ApoloLopes/clpMonitor/internal/models"
)

// Driver é a interface que todos os protocolos (S7, Rockwell) devem implementar
type Driver interface {
	Connect(ctx context.Context) error
	Disconnect() error
	// ReadTags recebe um ponteiro para um slice de tags e atualiza seus valores in-place
	ReadTags(tags *[]models.Tag) error
}
