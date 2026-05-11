// ui/config.go
// Le e grava configuracoes no registro do Windows.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/epson-fx80-emulator/storage"
	"golang.org/x/sys/windows/registry"
)

const regKeyPath = `SOFTWARE\EpsonFX80Emulator`

// Config armazena todas as configuracoes da aplicacao.
type Config struct {
	OutputDir   string
	PaperType   int  // 0=branco 1=verde 2=azul
	TractorFeed bool
	Columns     int  // 80 ou 132
}

// loadConfig le as configuracoes do registro. Usa defaults se nao encontrar.
func loadConfig() Config {
	cfg := Config{
		OutputDir:   defaultOutputDir(),
		PaperType:   0,
		TractorFeed: false,
		Columns:     80,
	}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, regKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return cfg
	}
	defer k.Close()

	if v, _, err := k.GetStringValue("OutputDir"); err == nil && v != "" {
		cfg.OutputDir = v
	}
	if v, _, err := k.GetIntegerValue("PaperType"); err == nil {
		cfg.PaperType = int(v)
	}
	if v, _, err := k.GetIntegerValue("TractorFeed"); err == nil {
		cfg.TractorFeed = v != 0
	}
	if v, _, err := k.GetIntegerValue("Columns"); err == nil {
		cfg.Columns = int(v)
		if cfg.Columns != 132 {
			cfg.Columns = 80
		}
	}
	return cfg
}

// saveConfig grava todas as configuracoes no registro via reg.exe.
func saveConfig(cfg Config) error {
	regKey := `HKLM\SOFTWARE\EpsonFX80Emulator`

	tractorVal := "0"
	if cfg.TractorFeed {
		tractorVal = "1"
	}

	cmds := [][]string{
		{"reg", "add", regKey, "/v", "OutputDir", "/t", "REG_SZ", "/d", cfg.OutputDir, "/f"},
		{"reg", "add", regKey, "/v", "PaperType", "/t", "REG_DWORD", "/d", strconv.Itoa(cfg.PaperType), "/f"},
		{"reg", "add", regKey, "/v", "TractorFeed", "/t", "REG_DWORD", "/d", tractorVal, "/f"},
		{"reg", "add", regKey, "/v", "Columns", "/t", "REG_DWORD", "/d", strconv.Itoa(cfg.Columns), "/f"},
	}

	for _, args := range cmds {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return err
		}
	}
	return nil
}

func defaultOutputDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents", "EpsonFX80")
}

// openOutputDir abre a pasta de PDFs no Explorer.
func openOutputDir() {
	cfg := loadConfig()
	os.MkdirAll(cfg.OutputDir, 0755)
	exec.Command("explorer", cfg.OutputDir).Start()
}

// openFile abre um arquivo com o programa padrao do Windows.
func openFile(path string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
}

// openDB abre o banco SQLite na mesma pasta do executavel.
func openDB() (*storage.DB, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(filepath.Dir(exe), "epson_fx80.db")
	return storage.Open(dbPath)
}

// serviceStatusText retorna o status atual do servico Windows.
func serviceStatusText() string {
	out, err := exec.Command("sc", "query", "EpsonFX80Monitor").Output()
	if err != nil {
		return "Nao instalado"
	}
	s := string(out)
	switch {
	case strings.Contains(s, "RUNNING"):
		return "Rodando"
	case strings.Contains(s, "STOPPED"):
		return "Parado"
	case strings.Contains(s, "START_PENDING"):
		return "Iniciando..."
	case strings.Contains(s, "STOP_PENDING"):
		return "Parando..."
	default:
		return "Desconhecido"
	}
}

// restartService reinicia o servico EpsonFX80Monitor.
func restartService() error {
	exec.Command("sc", "stop", "EpsonFX80Monitor").Run()
	return exec.Command("sc", "start", "EpsonFX80Monitor").Run()
}
