// pdfgen/options.go
// Le e salva as opcoes de papel no registro do Windows.
// Chave: HKLM\SOFTWARE\EpsonFX80Emulator
//   PaperType    REG_DWORD  0=branco 1=verde 2=azul
//   TractorFeed  REG_DWORD  0=nao 1=sim
//   Columns      REG_DWORD  80 ou 132

package pdfgen

import (
	"golang.org/x/sys/windows/registry"
)

const regKey = `SOFTWARE\EpsonFX80Emulator`

// LoadOptions le as opcoes de papel do registro do Windows.
// Retorna DefaultOptions() se a chave nao existir.
func LoadOptions() Options {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, regKey, registry.QUERY_VALUE)
	if err != nil {
		return DefaultOptions()
	}
	defer k.Close()

	opts := DefaultOptions()

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
	return opts
}
