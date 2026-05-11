// portmonitor/processor.go
// Le config do registro, converte texto em PDF e salva no SQLite.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/epson-fx80-emulator/pdfgen"
	"github.com/epson-fx80-emulator/storage"
	"golang.org/x/sys/windows/registry"
)

func processJob(jobID int, baseName string, data []byte) error {
	outputDir := readOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("nao foi possivel criar diretorio: %w", err)
	}

	text := bytesToText(data)
	if strings.TrimSpace(text) == "" {
		log.Printf("[job #%d] Conteudo vazio apos limpeza, ignorando", jobID)
		return nil
	}

	pdfPath := filepath.Join(outputDir, baseName+".pdf")
	log.Printf("[job #%d] Gerando PDF: %s", jobID, pdfPath)

	opts := pdfgen.LoadOptions()
	// Injeta o mapa de fontes pre-carregado na inicializacao do servico
	if fontManager != nil {
		opts.Fonts = fontManager.Map
	}
	pages, err := pdfgen.Generate(pdfPath, text, opts)
	if err != nil {
		return fmt.Errorf("erro ao gerar PDF: %w", err)
	}
	log.Printf("[job #%d] PDF gerado com %d pagina(s)", jobID, pages)

	db, err := storage.Open(dbPath())
	if err != nil {
		log.Printf("[job #%d] Aviso: banco indisponivel: %v", jobID, err)
		return nil
	}
	defer db.Close()

	return db.InsertJob(storage.Job{
		Name:      baseName,
		PDFPath:   pdfPath,
		Pages:     pages,
		ByteSize:  len(data),
		CreatedAt: time.Now(),
	})
}

func readOutputDir() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\EpsonFX80Emulator`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if v, _, err := k.GetStringValue("OutputDir"); err == nil && v != "" {
			return v
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents", "EpsonFX80")
}

func dbPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "epson_fx80.db"
	}
	return filepath.Join(filepath.Dir(exe), "epson_fx80.db")
}

func bytesToText(data []byte) string {
	var s string
	if utf8.Valid(data) {
		s = string(data)
	} else {
		var sb strings.Builder
		for _, b := range data {
			if b < 128 {
				sb.WriteByte(b)
			} else {
				sb.WriteRune(rune(b))
			}
		}
		s = sb.String()
	}
	return cleanControlChars(s)
}

func cleanControlChars(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		r := runes[i]
		switch {
		case r == 0x1B:
			i = skipEscSeq(runes, i)
			continue
		case r == '\n' || r == '\r' || r == '\t' || r == '\f':
			sb.WriteRune(r)
		case r < 0x20 || r == 0x7F:
			// ignora outros controles
		default:
			sb.WriteRune(r)
		}
		i++
	}
	return sb.String()
}

func skipEscSeq(runes []rune, i int) int {
	if i+1 >= len(runes) {
		return i + 1
	}
	fixed := map[rune]int{
		'@': 0, 'E': 0, 'F': 0, 'G': 0, 'H': 0, '4': 0, '5': 0,
		'6': 0, '7': 0, '8': 0, '9': 0, 'M': 0, 'P': 0,
		'W': 1, 'A': 1, 'J': 1, 'N': 1, 'Q': 1, 'R': 1, 'S': 1,
		'U': 1, '3': 1,
	}
	if n, ok := fixed[runes[i+1]]; ok {
		return i + 2 + n
	}
	return i + 2
}
