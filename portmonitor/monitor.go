// portmonitor/monitor.go
// Loop principal: cria o named pipe e despacha jobs recebidos do spooler.

package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Microsoft/go-winio"
)

func runMonitor() {
	stop := make(chan struct{})
	runMonitorWithStop(stop)
}

func runMonitorWithStop(stop <-chan struct{}) {
	log.Printf("[monitor] Criando pipe: %s", PipeName)

	cfg := &winio.PipeConfig{
		// Permite leitura/escrita pelo SYSTEM (spooler) e administradores
		SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;WD)",
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	listener, err := winio.ListenPipe(PipeName, cfg)
	if err != nil {
		log.Fatalf("[monitor] Falha ao criar pipe: %v", err)
	}
	defer listener.Close()

	log.Println("[monitor] Aguardando jobs do spooler...")

	go func() {
		<-stop
		listener.Close()
	}()

	jobCounter := 0
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-stop:
				return
			default:
				log.Printf("[monitor] Erro ao aceitar conexao: %v - tentando em 1s", err)
				time.Sleep(time.Second)
				continue
			}
		}
		jobCounter++
		id := jobCounter
		log.Printf("[monitor] Job #%d recebido", id)
		go handleJob(id, conn)
	}
}

func handleJob(id int, r io.ReadCloser) {
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		log.Printf("[job #%d] Erro lendo pipe: %v", id, err)
		return
	}
	if len(data) == 0 {
		log.Printf("[job #%d] Job vazio, ignorando", id)
		return
	}
	log.Printf("[job #%d] %d bytes recebidos", id, len(data))

	timestamp := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("%s_job%04d", timestamp, id)

	if err := processJob(id, baseName, data); err != nil {
		log.Printf("[job #%d] Erro: %v", id, err)
	}
}
