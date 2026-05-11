// ui/main.go
// Interface grafica da impressora Epson FX-80 Emulator.
// Roda como icone na bandeja do sistema (system tray).
// Ao clicar no icone, abre a janela principal com o historico de jobs.
//
// Build: GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o ui.exe ./ui/

package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

func main() {
	setupLog()

	a := app.NewWithID("com.epson.fx80.emulator")
	a.SetIcon(printerIcon())

	w := newMainWindow(a)

	// System tray: so disponivel em desktop
	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(printerIcon())
		desk.SetSystemTrayMenu(buildTrayMenu(a, w))
	} else {
		// Fallback: sem suporte a tray, mostra janela direto
		w.Show()
	}

	// Nao mostra a janela ao iniciar - fica so na bandeja
	a.Run()
}

// buildTrayMenu cria o menu do system tray.
func buildTrayMenu(a fyne.App, w fyne.Window) *fyne.Menu {
	showItem := fyne.NewMenuItem("Abrir gerenciador", func() {
		w.Show()
		w.RequestFocus()
	})

	openDirItem := fyne.NewMenuItem("Abrir pasta de PDFs", func() {
		openOutputDir()
	})

	sep := fyne.NewMenuItemSeparator()

	quitItem := fyne.NewMenuItem("Encerrar", func() {
		a.Quit()
	})

	return fyne.NewMenu("Epson FX-80 Emulator", showItem, openDirItem, sep, quitItem)
}

func setupLog() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	logPath := filepath.Join(filepath.Dir(exe), "ui.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
