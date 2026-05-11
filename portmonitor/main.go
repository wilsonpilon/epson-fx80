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

	"github.com/epson-fx80-emulator/fontmgr"
	"golang.org/x/sys/windows/svc"
)

const (
	ServiceName = "EpsonFX80Monitor"
	PipeName    = `\\.\pipe\epson_fx80_emulator`
)

// fontManager e o gerenciador de fontes pre-carregado na inicializacao.
// Usado pelo processor para saber quais TTFs aplicar em cada job.
var fontManager *fontmgr.Manager

func main() {
	debug := flag.Bool("debug", false, "Roda no terminal em vez de como servico Windows")
	flag.Parse()

	setupLog()
	preloadFonts()

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

// preloadFonts inicializa o fontManager e loga as fontes encontradas.
func preloadFonts() {
	exe, err := os.Executable()
	if err != nil {
		log.Println("[fonts] Nao foi possivel determinar o diretorio do executavel")
		return
	}
	exeDir := filepath.Dir(exe)
	fontManager = fontmgr.NewManager(exeDir)

	if !fontManager.HasFontsDir() {
		log.Printf("[fonts] Pasta de fontes nao encontrada: %s", fontManager.FontsDir)
		log.Println("[fonts] Usando fonte padrao Courier para todos os modos")
		return
	}

	log.Printf("[fonts] Pasta de fontes: %s", fontManager.FontsDir)
	for _, mode := range fontmgr.AllModes {
		if path := fontManager.SelectedFont(mode); path != "" {
			log.Printf("[fonts] %-30s -> %s", fontmgr.ModeLabel(mode), filepath.Base(path))
		}
	}
	log.Println("[fonts] Pre-carga concluida")
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
