// pdfgen/testpage.go
// Gera o PDF da pagina de teste com controle direto de fonte por modo.
// Cada modo usa seu proprio TTF registrado independentemente no fpdf,
// garantindo que Bold, Italic, Condensed etc sejam visualmente distintos.

package pdfgen

import (
	"fmt"
	"strings"

	"github.com/epson-fx80-emulator/fontmgr"
	"github.com/go-pdf/fpdf"
)

// FontEntry descreve uma fonte a ser usada na pagina de teste.
type FontEntry struct {
	Mode     fontmgr.Mode
	Label    string // ex: "Negrito (Bold)"
	FontFile string // caminho absoluto do TTF, ou "" para Courier
	FontName string // nome do arquivo sem path, para exibicao
}

// GenerateTestPage gera o PDF da pagina de teste em pdfPath.
// Cada modo tem seu TTF registrado com familia unica, garantindo
// que o fpdf use o arquivo correto sem conflito de estilos.
func GenerateTestPage(pdfPath string, entries []FontEntry, versionLine string) (int, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, marginBottom)

	// Registra cada TTF com nome de familia unico: "TTF_Regular", "TTF_Bold", etc.
	// Isso evita o problema de o fpdf mapear Bold/Italic para estilos da mesma familia.
	type fontInfo struct {
		family string
		size   float64
		lineH  float64
	}
	fonts := make(map[fontmgr.Mode]fontInfo)

	for _, e := range entries {
		family := "Courier"
		size := 10.0
		if e.FontFile != "" {
			family = "TTF_" + strings.ReplaceAll(string(e.Mode), " ", "_")
			pdf.AddUTF8Font(family, "", e.FontFile)
		}
		// Condensed usa fonte menor para simular compressao
		switch e.Mode {
		case fontmgr.ModeCondensed, fontmgr.ModeCondensedBold,
			fontmgr.ModeCondensedItalic, fontmgr.ModeCondensedBoldItalic:
			size = 8.0
		// Expanded usa fonte maior
		case fontmgr.ModeExpanded, fontmgr.ModeExpandedBold,
			fontmgr.ModeExpandedItalic, fontmgr.ModeExpandedBoldItalic:
			size = 13.0
		}
		fonts[e.Mode] = fontInfo{family: family, size: size, lineH: size * 0.55}
	}

	const mL, mR, mT = 15.0, 15.0, 15.0
	textW := 210.0 - mL - mR
	pageCount := 0

	newPage := func() {
		pdf.AddPage()
		pageCount++
		pdf.SetFillColor(255, 255, 255)
		pdf.Rect(0, 0, 210, 297, "F")
		pdf.SetTextColor(30, 30, 30)
		pdf.SetXY(mL, mT)
	}

	// useFont ativa a fonte do modo no fpdf
	useFont := func(mode fontmgr.Mode) float64 {
		fi, ok := fonts[mode]
		if !ok {
			pdf.SetFont("Courier", "", 10)
			return 5.5
		}
		pdf.SetFont(fi.family, "", fi.size)
		return fi.lineH
	}

	// cell escreve uma linha com a fonte do modo e retorna a altura usada
	cell := func(text string, mode fontmgr.Mode) {
		lh := useFont(mode)
		pdf.MultiCell(textW, lh, text, "", "L", false)
	}

	// hdr escreve em Courier bold (para cabecalhos fixos)
	hdr := func(text string, size float64, align string) {
		pdf.SetFont("Courier", "B", size)
		pdf.MultiCell(textW, size*0.55, text, "", align, false)
	}
	mono := func(text string, size float64) {
		pdf.SetFont("Courier", "", size)
		pdf.MultiCell(textW, size*0.5, text, "", "L", false)
	}

	// -- Pagina 1: cabecalho + regua + ASCII ------------------------------
	newPage()

	hdr("EPSON FX-80 EMULATOR - PAGINA DE TESTE", 12, "C")
	mono(versionLine, 9)
	mono(strings.Repeat("=", 80), 8)
	pdf.Ln(2)

	// Regua de 80 colunas
	hdr("REGUA DE 80 COLUNAS:", 9, "L")
	pdf.Ln(1)

	var top, bot, marks strings.Builder
	for i := 1; i <= 80; i++ {
		top.WriteByte(byte('0' + (i/10)%10))
		bot.WriteByte(byte('0' + i%10))
		switch {
		case i%10 == 0:
			marks.WriteByte('|')
		case i%5 == 0:
			marks.WriteByte('+')
		default:
			marks.WriteByte('-')
		}
	}
	mono(top.String(), 8)
	mono(bot.String(), 8)
	mono(marks.String(), 8)
	pdf.Ln(3)

	// Caracteres ASCII
	hdr("CARACTERES ASCII IMPRIMIVEIS (32-126):", 9, "L")
	pdf.Ln(1)
	var ascii strings.Builder
	for c := 32; c <= 126; c++ {
		ascii.WriteByte(byte(c))
	}
	str := ascii.String()
	for i := 0; i < len(str); i += 64 {
		end := i + 64
		if end > len(str) {
			end = len(str)
		}
		mono("  "+str[i:end], 8)
	}
	pdf.Ln(3)

	// -- Bloco de teste por fonte -----------------------------------------
	mono(strings.Repeat("=", 80), 8)
	hdr("TESTE DE FONTES POR MODO DE IMPRESSAO:", 10, "L")
	mono(strings.Repeat("-", 80), 8)
	pdf.Ln(2)

	const phrase = "Epson FX-80 Emulator Driver (Go)"
	const digits = "1234567890  !@#$%^&*()  ABCDEFGHIJKLMNOPQRSTUVWXYZ  abcdefghijklmnopqrstuvwxyz"

	for _, e := range entries {
		// Nova pagina se estiver perto do fim
		if pdf.GetY() > 255 {
			newPage()
		}

		// Cabecalho do bloco em Courier pequeno
		header := fmt.Sprintf("[%s]", e.Label)
		if e.FontFile != "" {
			header += "  arquivo: " + e.FontName
		} else {
			header += "  Courier (padrao)"
		}
		mono(header, 7)

		// Frase e digitos na fonte real do modo
		cell(phrase, e.Mode)
		cell(digits, e.Mode)

		// Separador pontilhado em Courier
		mono(strings.Repeat(".", 80), 7)
		pdf.Ln(1)
	}

	// Rodape
	if pdf.GetY() > 268 {
		newPage()
	}
	mono(strings.Repeat("=", 80), 8)
	hdr("FIM DA PAGINA DE TESTE", 9, "C")
	mono("Epson FX-80 Emulator  github.com/epson-fx80-emulator", 8)

	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		return 0, fmt.Errorf("erro ao salvar PDF de teste: %w", err)
	}
	return pageCount, nil
}
