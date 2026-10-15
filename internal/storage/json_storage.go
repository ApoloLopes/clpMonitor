package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/ApoloLopes/clpMonitor/internal/models"
)

// Config representa o formato do nosso ficheiro JSON no disco
type Config struct {
	Devices []models.Device `json:"devices"`
	Tags    []models.Tag    `json:"tags"`
}

// JSONStorage gere o acesso ao ficheiro de forma segura contra concorrência
type JSONStorage struct {
	filePath string
	mu       sync.RWMutex
	config   Config
}

// NewJSONStorage inicializa o armazenamento e cria o ficheiro caso não exista
func NewJSONStorage(filePath string) (*JSONStorage, error) {
	storage := &JSONStorage{
		filePath: filePath,
		config: Config{
			Devices: []models.Device{},
			Tags:    []models.Tag{},
		},
	}

	err := storage.load()
	if err != nil {
		return nil, err
	}

	return storage, nil
}

// load lê os dados do disco para a memória
func (s *JSONStorage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verifica se o ficheiro existe, se não, cria um vazio com a estrutura base
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return s.saveToFile()
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("erro ao ler ficheiro de configuração: %v", err)
	}

	if err := json.Unmarshal(data, &s.config); err != nil {
		return fmt.Errorf("ficheiro JSON corrompido: %v", err)
	}

	return nil
}

// saveToFile escreve os dados da memória para o disco (sem lock, pois é chamado por métodos que já têm lock)
func (s *JSONStorage) saveToFile() error {
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

// GetConfig devolve uma cópia segura das configurações atuais
func (s *JSONStorage) GetConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// AddDevice adiciona um novo dispositivo e guarda no disco
func (s *JSONStorage) AddDevice(device models.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.Devices = append(s.config.Devices, device)
	return s.saveToFile()
}

// AddTag adiciona uma nova tag e guarda no disco
func (s *JSONStorage) AddTag(tag models.Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.Tags = append(s.config.Tags, tag)
	return s.saveToFile()
}
