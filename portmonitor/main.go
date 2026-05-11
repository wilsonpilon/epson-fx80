// portmonitor/main.go
// Servico Windows que escuta o named pipe e converte jobs de impressao em PDF.
// Uso: portmonitor.exe          -> roda como servico Windows
//      portmonitor.exe -debug   -> roda no terminal com logs ao vivo

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/svc"
)

const (
	ServiceName = "EpsonFX80Monitor"
	PipeName    = `\\.\pipe\epson_fx80_emulator`
)

func main() {
	debug := flag.Bool("debug", false, "Roda no terminal em vez de como servico Windows")
	flag.Parse()

	setupLog()

	if *debug {
		log.Println("Modo debug: rodando no terminal. Ctrl+C para encerrar.")
		runMonitor()
		return
	}

	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Erro ao detectar contexto: %v", err)
	}

	if isSvc {
		if err := svc.Run(ServiceName, &epsonService{}); err != nil {
			log.Fatalf("Servico encerrado com erro: %v", err)
		}
	} else {
		fmt.Println("Epson FX-80 Port Monitor")
		fmt.Println("  -debug   : roda no terminal")
		fmt.Println("  sc start : inicia como servico Windows")
	}
}

func setupLog() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	logPath := filepath.Join(filepath.Dir(exe), "portmonitor.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
