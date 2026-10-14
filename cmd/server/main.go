package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/plc"
)

func main() {
	device := models.Device{
		ID:        "PLC_01",
		Name:      "Extrusora",
		Protocol:  "s7",
		IPAddress: "127.0.0.1",
		Rack:      0,
		Slot:      2,
	}

	// Criamos a lista de variáveis que queremos monitorar
	tags := []models.Tag{
		{ID: "T1", Name: "Temperatura", DBNumber: 1, Offset: 4, DataType: "REAL"},
		{ID: "T2", Name: "Motor_Ligado", DBNumber: 1, Offset: 0, Bit: 0, DataType: "BOOL"},
		{ID: "T3", Name: "Velocidade", DBNumber: 1, Offset: 2, DataType: "INT"},
	}

	// Injeção de dependência: instanciamos o S7 mas o tratamos como um Driver genérico
	var driver plc.Driver = plc.NewS7Driver(device)

	// Contexto de 3 segundos para abortar a conexão se o cabo estiver desconectado
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	fmt.Println("Conectando ao CLP...")
	if err := driver.Connect(ctx); err != nil {
		log.Fatalf("Erro fatal de conexão: %v", err)
	}
	defer driver.Disconnect()

	fmt.Println("Conectado! Iniciando leitura...")

	// Loop de varredura
	for {
		err := driver.ReadTags(&tags)
		if err != nil {
			log.Printf("Erro geral de leitura: %v", err)
		}

		// Imprimimos o resultado de cada tag
		for _, tag := range tags {
			if tag.Quality {
				fmt.Printf("[%s] %s: %v\n", tag.Timestamp.Format("15:04:05"), tag.Name, tag.Value)
			} else {
				fmt.Printf("[%s] %s: BAD QUALITY (Falha na leitura)\n", tag.Timestamp.Format("15:04:05"), tag.Name)
			}
		}

		fmt.Println("----------------------")
		time.Sleep(1 * time.Second)
	}
}
