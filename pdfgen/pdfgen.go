// pdfgen/pdfgen.go
// Converte texto simples em PDF com opcoes de papel matricial da epoca:
// - Tipo de papel: branco, zebrado verde, zebrado azul
// - Furos de trator: faixa cinza lateral com circulos
// - Colunas: 80 ou 132

package pdfgen

import (
	"fmt"
	"math"
	"strings"

	"github.com/epson-fx80-emulator/fontmgr"
	"github.com/go-pdf/fpdf"
)

// PaperType define o tipo de papel.
type PaperType int

const (
	PaperWhite      PaperType = iota // Papel branco simples
	PaperGreenZebra                  // Papel zebrado verde claro
	PaperBlueZebra                   // Papel zebrado azul claro
)

// Columns define a largura em colunas.
type Columns int

const (
	Columns80  Columns = 80
	Columns132 Columns = 132
)

// Options contem as configuracoes de papel e fontes.
type Options struct {
	Paper       PaperType
	Cols        Columns
	TractorFeed bool            // exibe faixa lateral com furos de trator
	Fonts       fontmgr.FontMap // mapeamento modo -> arquivo TTF (nil = usar Courier)
}

// DefaultOptions retorna as opcoes padrao (papel branco, 80 col, sem trator).
func DefaultOptions() Options {
	return Options{
		Paper:       PaperWhite,
		Cols:        Columns80,
		TractorFeed: false,
		Fonts:       make(fontmgr.FontMap),
	}
}

// dimensoes da pagina em mm
const (
	pageW        = 210.0
	pageH        = 297.0
	marginTop    = 12.0
	marginBottom = 12.0

	tractorW     = 12.0 // largura da faixa de trator em mm
	tractorHoleR = 2.5  // raio dos furos em mm
	holeSpacing  = 12.7 // espacamento entre furos (0.5 pol = espacamento real)

	defaultFont = "Courier" // fallback quando nenhuma TTF estiver configurada
)

// Generate cria um PDF em pdfPath a partir de text com as opcoes dadas.
// Retorna o numero de paginas geradas.
func Generate(pdfPath, text string, opts Options) (int, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0) // controlamos manualmente

	// Registra fontes TTF configuradas e determina a familia a usar
	fontFamily := registerFonts(pdf, opts.Fonts)

	// Calcula margens baseado em colunas e presenca de trator
	marginLeft, marginRight, fontSize, lineH := calcLayout(opts)

	pdf.SetMargins(marginLeft, marginTop, marginRight)
	pdf.SetFont(fontFamily, "", fontSize)

	logicalPages := strings.Split(text, "\f")
	pageCount := 0

	for _, page := range logicalPages {
		if strings.TrimSpace(page) == "" {
			continue
		}
		pdf.AddPage()
		pageCount++

		// Desenha fundo e decoracoes da pagina
		drawPageDecorations(pdf, opts, lineH)

		// Posiciona cursor para escrita do texto
		pdf.SetXY(marginLeft, marginTop)
		pdf.SetFont(fontFamily, "", fontSize)
		pdf.SetTextColor(30, 30, 30)

		lines := strings.Split(page, "\n")
		for i, line := range lines {
			line = strings.TrimRight(line, "\r")

			y := marginTop + float64(i)*lineH
			if y+lineH > pageH-marginBottom {
				break // nao estoura a pagina
			}

			// Fundo zebrado: linha par = cor, linha impar = branco
			if opts.Paper != PaperWhite {
				if i%2 == 0 {
					r, g, b := zebraColor(opts.Paper)
					pdf.SetFillColor(r, g, b)
					textW := pageW - marginLeft - marginRight
					pdf.Rect(marginLeft, y, textW, lineH, "F")
				}
			}

			// Escreve o texto
			pdf.SetXY(marginLeft, y)
			pdf.SetFont(fontFamily, "", fontSize)
			pdf.SetTextColor(30, 30, 30)

			// Trunca a linha conforme o numero de colunas
			maxCols := int(opts.Cols)
			runes := []rune(line)
			if len(runes) > maxCols {
				runes = runes[:maxCols]
			}
			pdf.CellFormat(pageW-marginLeft-marginRight, lineH, string(runes), "", 0, "L", false, 0, "")
		}
	}

	if pageCount == 0 {
		pdf.AddPage()
		pageCount = 1
		drawPageDecorations(pdf, opts, lineH)
		pdf.SetXY(marginLeft, marginTop)
		pdf.SetFont(fontFamily, "I", fontSize)
		pdf.SetTextColor(150, 150, 150)
		pdf.Cell(0, lineH, "(pagina em branco)")
	}

	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		return 0, fmt.Errorf("erro ao salvar PDF: %w", err)
	}
	return pageCount, nil
}

// registerFonts registra as fontes TTF no fpdf e retorna o nome da familia a usar.
// Se nenhuma fonte TTF estiver configurada, retorna "Courier" (built-in).
// A familia registrada usa o nome "EpsonTTF" e mapeia cada estilo para o TTF correto.
func registerFonts(pdf *fpdf.Fpdf, fonts fontmgr.FontMap) string {
	if len(fonts) == 0 {
		return defaultFont
	}

	// Verifica se ha pelo menos a fonte Regular configurada
	regularPath, hasRegular := fonts[fontmgr.ModeRegular]
	if !hasRegular || regularPath == "" {
		return defaultFont
	}

	const familyName = "EpsonTTF"

	// Registra Regular (obrigatorio)
	pdf.AddUTF8Font(familyName, "", regularPath)

	// Registra estilos opcionais se configurados
	styleMap := map[fontmgr.Mode]string{
		fontmgr.ModeBold:       "B",
		fontmgr.ModeItalic:     "I",
		fontmgr.ModeBoldItalic: "BI",
	}
	for mode, style := range styleMap {
		if path, ok := fonts[mode]; ok && path != "" {
			pdf.AddUTF8Font(familyName, style, path)
		}
	}

	return familyName
}

// calcLayout retorna margem esquerda, direita, tamanho de fonte e altura de linha
// com base nas opcoes de papel.
func calcLayout(opts Options) (mL, mR, fSize, lineH float64) {
	tractor := 0.0
	if opts.TractorFeed {
		tractor = tractorW
	}

	// Largura de texto disponivel apos as faixas de trator
	textW := pageW - 2*tractor

	// Numero de colunas define o tamanho da fonte Courier
	// Courier 10pt = 2.12mm por caractere em 72dpi
	// Calculamos a fonte para caber exatamente o numero de colunas
	cols := float64(opts.Cols)
	// largura de um caractere Courier em mm = textW / cols
	charW := textW / cols
	// Em fpdf, Courier tem proporcao fixa: 1pt = 0.353mm, char width = fontSize * 0.6 * 0.353
	// charW = fontSize * 0.6 * 0.353 => fontSize = charW / (0.6 * 0.353)
	fSize = charW / (0.6 * 0.353)
	// Limita para leitura razoavel
	if fSize > 11 {
		fSize = 11
	}
	if fSize < 5 {
		fSize = 5
	}

	lineH = fSize * 0.353 * 1.8 // altura de linha = 1.8x a altura da fonte em mm

	mL = tractor + 4
	mR = tractor + 4
	return
}

// drawPageDecorations desenha o fundo da pagina e as faixas de trator.
func drawPageDecorations(pdf *fpdf.Fpdf, opts Options, lineH float64) {
	// Fundo base da pagina
	switch opts.Paper {
	case PaperGreenZebra:
		pdf.SetFillColor(240, 253, 244) // verde muito claro
	case PaperBlueZebra:
		pdf.SetFillColor(239, 246, 255) // azul muito claro
	default:
		pdf.SetFillColor(255, 255, 255)
	}
	pdf.Rect(0, 0, pageW, pageH, "F")

	// Faixas de trator
	if opts.TractorFeed {
		drawTractorStrip(pdf, 0, lineH)              // esquerda
		drawTractorStrip(pdf, pageW-tractorW, lineH) // direita
	}
}

// drawTractorStrip desenha a faixa cinza lateral com furos.
func drawTractorStrip(pdf *fpdf.Fpdf, x, lineH float64) {
	// Fundo cinza da faixa
	pdf.SetFillColor(229, 231, 235) // gray-200
	pdf.Rect(x, 0, tractorW, pageH, "F")

	// Linha divisoria interna (borda do picote)
	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.2)
	if x < pageW/2 {
		pdf.Line(x+tractorW, 0, x+tractorW, pageH)
	} else {
		pdf.Line(x, 0, x, pageH)
	}

	// Furos de trator
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.3)

	cx := x + tractorW/2
	numHoles := int(math.Ceil(pageH / holeSpacing))
	for i := 0; i <= numHoles; i++ {
		cy := float64(i)*holeSpacing + holeSpacing/2
		if cy > pageH {
			break
		}
		// Circulo branco com borda cinza
		pdf.Circle(cx, cy, tractorHoleR, "FD")
	}
}

// zebraColor retorna a cor RGB da faixa zebrada escura para o tipo de papel.
func zebraColor(p PaperType) (r, g, b int) {
	switch p {
	case PaperGreenZebra:
		return 187, 247, 208 // green-200
	case PaperBlueZebra:
		return 191, 219, 254 // blue-200
	default:
		return 255, 255, 255
	}
}
