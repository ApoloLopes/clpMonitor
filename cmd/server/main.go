package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ApoloLopes/clpMonitor/internal/models"
	"github.com/ApoloLopes/clpMonitor/internal/plc"
	"github.com/ApoloLopes/clpMonitor/internal/worker"
)

func main() {
	// 1. Configuração do Dispositivo e Tags
	device := models.Device{
		ID: "PLC_01", Name: "Extrusora", Protocol: "s7", IPAddress: "127.0.0.1", Rack: 0, Slot: 2,
	}

	tags := []models.Tag{
		{ID: "T1", Name: "Temperatura", DBNumber: 1, Offset: 4, DataType: "REAL"},
		{ID: "T2", Name: "Motor_Ligado", DBNumber: 1, Offset: 0, Bit: 0, DataType: "BOOL"},
		{ID: "T3", Name: "Velocidade", DBNumber: 1, Offset: 2, DataType: "INT"},
	}

	// 2. Inicialização do Driver
	var driver plc.Driver = plc.NewS7Driver(device)

	// Contexto base para conectar
	connCtx, cancelConn := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelConn()

	if err := driver.Connect(connCtx); err != nil {
		log.Fatalf("Erro ao conectar no CLP: %v", err)
	}

	// 3. Criação e disparo do Worker
	// Configurado para ler a cada 500 milissegundos
	plcWorker := worker.NewPollingWorker(device, driver, tags, 3000)

	// Contexto para controlar o ciclo de vida do Worker (Graceful Shutdown)
	ctx, cancelAllWorkers := context.WithCancel(context.Background())

	// DISPARA O WORKER EM CONCORRÊNCIA! A thread principal continua executando.
	go plcWorker.Start(ctx)

	// 4. Lógica para escutar os dados e escutar interrupção do teclado (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Sistema rodando. Pressione Ctrl+C para sair.")

	for {
		select {
		case data := <-plcWorker.DataChannel:
			// Recebeu dados frescos do Worker
			fmt.Printf("\n--- Leitura %s ---\n", time.Now().Format("15:04:05.000"))
			for _, t := range data {
				if t.Quality {
					fmt.Printf("%s: %v\n", t.Name, t.Value)
				} else {
					fmt.Printf("%s: BAD QUALITY\n", t.Name)
				}
			}

		case <-sigChan:
			// Usuário apertou Ctrl+C
			fmt.Println("\nDesligando sistema de forma segura...")
			cancelAllWorkers()          // Avisa todos os workers para pararem
			time.Sleep(1 * time.Second) // Dá um tempo para os workers fecharem as conexões
			fmt.Println("Fui!")
			return
		}
	}
}
