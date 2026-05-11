// fontmgr/fontmgr.go
//
// Gerencia as fontes TTF para os modos de impressao da Epson FX-80.
//
// Estrutura esperada em fonts\ttf\ (relativa ao executavel):
//   fonts\ttf\regular\     -> fontes para modo Regular
//   fonts\ttf\bold\        -> fontes para modo Bold
//   fonts\ttf\italic\      -> fontes para modo Italic
//   fonts\ttf\condensed\   -> fontes para modo Condensed
//   fonts\ttf\expanded\    -> fontes para modo Expanded
//   fonts\ttf\bold-italic\ -> fontes para modo BoldItalic
//   ... etc para cada combinacao
//
// O usuario escolhe qual arquivo .ttf usar em cada modo.
// O mapeamento e salvo no registro do Windows.

package fontmgr

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Mode representa um modo de impressao da Epson FX-80.
type Mode string

const (
	ModeRegular             Mode = "Regular"
	ModeBold                Mode = "Bold"
	ModeItalic              Mode = "Italic"
	ModeCondensed           Mode = "Condensed"
	ModeExpanded            Mode = "Expanded"
	ModeBoldItalic          Mode = "BoldItalic"
	ModeCondensedBold       Mode = "CondensedBold"
	ModeCondensedItalic     Mode = "CondensedItalic"
	ModeExpandedBold        Mode = "ExpandedBold"
	ModeExpandedItalic      Mode = "ExpandedItalic"
	ModeExpandedBoldItalic  Mode = "ExpandedBoldItalic"
	ModeCondensedBoldItalic Mode = "CondensedBoldItalic"
)

// AllModes lista todos os modos na ordem de exibicao na UI.
var AllModes = []Mode{
	ModeRegular,
	ModeBold,
	ModeItalic,
	ModeCondensed,
	ModeExpanded,
	ModeBoldItalic,
	ModeCondensedBold,
	ModeCondensedItalic,
	ModeExpandedBold,
	ModeExpandedItalic,
	ModeExpandedBoldItalic,
	ModeCondensedBoldItalic,
}

// ModeLabel retorna o label legivel de um modo para a UI.
func ModeLabel(m Mode) string {
	labels := map[Mode]string{
		ModeRegular:             "Regular",
		ModeBold:                "Negrito (Bold)",
		ModeItalic:              "Italico (Italic)",
		ModeCondensed:           "Condensado (Condensed)",
		ModeExpanded:            "Expandido (Expanded)",
		ModeBoldItalic:          "Negrito + Italico",
		ModeCondensedBold:       "Condensado + Negrito",
		ModeCondensedItalic:     "Condensado + Italico",
		ModeExpandedBold:        "Expandido + Negrito",
		ModeExpandedItalic:      "Expandido + Italico",
		ModeExpandedBoldItalic:  "Expandido + Negrito + Italico",
		ModeCondensedBoldItalic: "Condensado + Negrito + Italico",
	}
	if l, ok := labels[m]; ok {
		return l
	}
	return string(m)
}

// SubfolderForMode retorna o nome da subpasta esperada para cada modo.
// O usuario organiza os TTFs em subpastas cujo nome corresponde ao modo.
func SubfolderForMode(m Mode) string {
	names := map[Mode]string{
		ModeRegular:             "regular",
		ModeBold:                "bold",
		ModeItalic:              "italic",
		ModeCondensed:           "condensed",
		ModeExpanded:            "expanded",
		ModeBoldItalic:          "bold-italic",
		ModeCondensedBold:       "condensed-bold",
		ModeCondensedItalic:     "condensed-italic",
		ModeExpandedBold:        "expanded-bold",
		ModeExpandedItalic:      "expanded-italic",
		ModeExpandedBoldItalic:  "expanded-bold-italic",
		ModeCondensedBoldItalic: "condensed-bold-italic",
	}
	if n, ok := names[m]; ok {
		return n
	}
	return strings.ToLower(string(m))
}

// FontMap mapeia cada modo para o caminho absoluto do arquivo TTF selecionado.
// Valor vazio = usar fonte padrao (Courier).
type FontMap map[Mode]string

// Manager gerencia a descoberta e configuracao das fontes.
type Manager struct {
	FontsDir string  // caminho absoluto para fonts\ttf\
	Map      FontMap // mapeamento atual modo -> arquivo TTF
}

const regKeyPath = `SOFTWARE\EpsonFX80Emulator\Fonts`

// NewManager cria um Manager com a pasta de fontes ao lado do executavel.
func NewManager(execDir string) *Manager {
	fontsDir := filepath.Join(execDir, "fonts", "ttf")
	m := &Manager{
		FontsDir: fontsDir,
		Map:      make(FontMap),
	}
	m.loadFromRegistry()
	return m
}

// FontsDir returns the fonts directory path for a given base directory.
func FontsDirFor(execDir string) string {
	return filepath.Join(execDir, "fonts", "ttf")
}

// AvailableFonts retorna a lista de arquivos TTF disponíveis para um dado modo.
// Escaneia a subpasta correspondente ao modo dentro de FontsDir.
// Retorna lista de caminhos absolutos, ordenados por nome.
func (m *Manager) AvailableFonts(mode Mode) []string {
	subDir := filepath.Join(m.FontsDir, SubfolderForMode(mode))
	entries, err := os.ReadDir(subDir)
	if err != nil {
		return nil
	}

	var fonts []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".ttf") || strings.HasSuffix(name, ".otf") {
			fonts = append(fonts, filepath.Join(subDir, e.Name()))
		}
	}
	sort.Strings(fonts)
	return fonts
}

// AvailableFontNames retorna apenas os nomes dos arquivos (sem path) para exibicao na UI.
func (m *Manager) AvailableFontNames(mode Mode) []string {
	paths := m.AvailableFonts(mode)
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	return names
}

// SelectedFont retorna o caminho absoluto da fonte selecionada para um modo.
// Se nenhuma fonte foi selecionada, retorna "".
func (m *Manager) SelectedFont(mode Mode) string {
	return m.Map[mode]
}

// SelectedFontName retorna apenas o nome do arquivo da fonte selecionada.
// Retorna "(padrao Courier)" se nenhuma fonte estiver selecionada.
func (m *Manager) SelectedFontName(mode Mode) string {
	p := m.Map[mode]
	if p == "" {
		return "(padrao Courier)"
	}
	return filepath.Base(p)
}

// SetFont define a fonte para um modo. path="" = usar padrao.
func (m *Manager) SetFont(mode Mode, path string) {
	m.Map[mode] = path
}

// SetFontByName define a fonte para um modo pelo nome do arquivo.
// Busca o arquivo na subpasta correspondente ao modo.
func (m *Manager) SetFontByName(mode Mode, name string) {
	if name == "" || name == "(padrao Courier)" {
		m.Map[mode] = ""
		return
	}
	subDir := filepath.Join(m.FontsDir, SubfolderForMode(mode))
	m.Map[mode] = filepath.Join(subDir, name)
}

// Save grava o mapeamento atual no registro do Windows.
func (m *Manager) Save() error {
	regKey := `HKLM\` + regKeyPath
	for _, mode := range AllModes {
		val := m.Map[mode]
		if err := exec.Command("reg", "add", regKey,
			"/v", string(mode),
			"/t", "REG_SZ",
			"/d", val,
			"/f",
		).Run(); err != nil {
			return err
		}
	}
	return nil
}

// loadFromRegistry carrega o mapeamento do registro do Windows.
func (m *Manager) loadFromRegistry() {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, regKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	for _, mode := range AllModes {
		if v, _, err := k.GetStringValue(string(mode)); err == nil {
			m.Map[mode] = v
		}
	}
}

// HasFontsDir verifica se a pasta fonts\ttf\ existe.
func (m *Manager) HasFontsDir() bool {
	_, err := os.Stat(m.FontsDir)
	return err == nil
}

// AllSubfolders retorna quais subpastas de modos existem em FontsDir.
func (m *Manager) AllSubfolders() []string {
	var found []string
	for _, mode := range AllModes {
		sub := filepath.Join(m.FontsDir, SubfolderForMode(mode))
		if _, err := os.Stat(sub); err == nil {
			found = append(found, SubfolderForMode(mode))
		}
	}
	return found
}
