// pdfgen/options.go
// Le as opcoes de papel e fontes do registro do Windows.

package pdfgen

import (
	"os"
	"path/filepath"

	"github.com/epson-fx80-emulator/fontmgr"
	"golang.org/x/sys/windows/registry"
)

const regKey = `SOFTWARE\EpsonFX80Emulator`

// LoadOptions le as opcoes de papel e fontes do registro do Windows.
// Retorna DefaultOptions() se a chave nao existir.
func LoadOptions() Options {
	opts := DefaultOptions()

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, regKey, registry.QUERY_VALUE)
	if err != nil {
		return opts
	}
	defer k.Close()

	if v, _, err := k.GetIntegerValue("PaperType"); err == nil {
		opts.Paper = PaperType(v)
	}
	if v, _, err := k.GetIntegerValue("TractorFeed"); err == nil {
		opts.TractorFeed = v != 0
	}
	if v, _, err := k.GetIntegerValue("Columns"); err == nil {
		if v == 132 {
			opts.Cols = Columns132
		} else {
			opts.Cols = Columns80
		}
	}

	// Carrega mapeamento de fontes usando o fontmgr
	execDir := executableDir()
	mgr := fontmgr.NewManager(execDir)
	opts.Fonts = mgr.Map

	return opts
}

// executableDir retorna o diretorio do executavel atual.
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
