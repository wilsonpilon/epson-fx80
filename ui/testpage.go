// ui/testpage.go
// Gera a pagina de teste chamando pdfgen.GenerateTestPage com uma FontEntry
// por modo, cada uma apontando diretamente para o arquivo TTF configurado.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/epson-fx80-emulator/fontmgr"
	"github.com/epson-fx80-emulator/pdfgen"
	"github.com/epson-fx80-emulator/storage"
)

// generateTestPage cria o PDF de teste e retorna o caminho do arquivo.
func generateTestPage() (string, error) {
	cfg := loadConfig()
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("nao foi possivel criar diretorio: %w", err)
	}

	entries := buildFontEntries()
	versionLine := fmt.Sprintf("Versao %s  build %s", Version, BuildStamp)

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_testpage.pdf", timestamp)
	pdfPath := filepath.Join(cfg.OutputDir, filename)

	pages, err := pdfgen.GenerateTestPage(pdfPath, entries, versionLine)
	if err != nil {
		return "", err
	}

	db, err := openDB()
	if err == nil {
		defer db.Close()
		db.InsertJob(storage.Job{
			Name:      strings.TrimSuffix(filename, ".pdf"),
			PDFPath:   pdfPath,
			Pages:     pages,
			ByteSize:  len(entries) * 100,
			CreatedAt: time.Now(),
		})
	}

	return pdfPath, nil
}

// buildFontEntries monta a lista de FontEntry lendo o fontManager do registro.
// Cada entrada tem o caminho real do TTF configurado para o modo -- a fonte
// sera registrada com familia unica no fpdf, garantindo distincao visual.
func buildFontEntries() []pdfgen.FontEntry {
	exeDir := executableDir()
	mgr := fontmgr.NewManager(exeDir)

	var entries []pdfgen.FontEntry
	for _, mode := range fontmgr.AllModes {
		entries = append(entries, pdfgen.FontEntry{
			Mode:     mode,
			Label:    fontmgr.ModeLabel(mode),
			FontFile: mgr.SelectedFont(mode),
			FontName: mgr.SelectedFontName(mode),
		})
	}
	return entries
}
