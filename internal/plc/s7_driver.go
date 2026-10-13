package plc

import (
	"fmt"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models" // Ajuste para o nome do seu módulo
	"github.com/robinson/gos7"
)

type S7Driver struct {
	Device  models.Device
	Client  gos7.Client
	Handler *gos7.TCPClientHandler
}

// NewS7Driver inicializa a estrutura do driver
func NewS7Driver(ip string, rack, slot int) *S7Driver {
	return &S7Driver{
		Device: models.Device{
			IPAddress: ip,
			Rack:      rack,
			Slot:      slot,
		},
	}
}

// Connect abre o socket TCP com o CLP
func (s *S7Driver) Connect() error {
	s.Handler = gos7.NewTCPClientHandler(s.Device.IPAddress, s.Device.Rack, s.Device.Slot)
	s.Handler.Timeout = 2 * time.Second
	s.Handler.IdleTimeout = 2 * time.Second

	if err := s.Handler.Connect(); err != nil {
		return fmt.Errorf("falha ao conectar no S7 em %s: %v", s.Device.IPAddress, err)
	}

	s.Client = gos7.NewClient(s.Handler)
	return nil
}

// Disconnect fecha a conexão de forma segura
func (s *S7Driver) Disconnect() {
	if s.Handler != nil {
		s.Handler.Close()
	}
}

// ReadDBReal lê 4 bytes de uma DB em um offset específico e converte para Float32 (Real no CLP)
func (s *S7Driver) ReadDBReal(dbNumber int, offset int) (float32, error) {
	buffer := make([]byte, 4) // Variável REAL ocupa 4 bytes

	// AGReadDB(numeroDaDB, offsetInicial, tamanho, bufferDestino)
	err := s.Client.AGReadDB(dbNumber, offset, 4, buffer)
	if err != nil {
		return 0, fmt.Errorf("erro ao ler DB%d.DBD%d: %v", dbNumber, offset, err)
	}

	// A biblioteca gos7 possui um Helper para extrair os dados do byte array já convertidos
	var helper gos7.Helper
	var value float32
	helper.GetValueAt(buffer, 0, &value)

	return value, nil
}
