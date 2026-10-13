package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/plc" // Ajuste para o nome do seu módulo
)

func main() {
	fmt.Println("Iniciando Gateway PLC...")

	// Instancia o driver (IP, Rack, Slot)
	// Para S7-1200/1500 o padrão é Rack 0, Slot 1. Para S7-300 geralmente é Rack 0, Slot 2.
	s7 := plc.NewS7Driver("192.168.0.10", 0, 1)

	err := s7.Connect()
	if err != nil {
		log.Fatalf("Erro fatal: %v", err)
	}
	defer s7.Disconnect()

	fmt.Println("Conectado ao CLP Siemens com sucesso!")

	// Loop de varredura (Scan)
	for {
		// Exemplo: Lendo a variável tipo REAL no offset 4 da DB 1 (DB1.DBD4)
		value, err := s7.ReadDBReal(1, 4)
		if err != nil {
			log.Printf("Falha na leitura: %v", err)
		} else {
			fmt.Printf("Valor atual da DB1.DBD4: %.2f\n", value)
		}

		// Pausa de 500ms entre as leituras para não sobrecarregar a rede
		time.Sleep(500 * time.Millisecond)
	}
}
