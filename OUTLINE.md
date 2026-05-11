# OUTLINE -- Epson FX-80 Emulator

Documento de contexto para continuidade do desenvolvimento.
Leia este arquivo antes de qualquer outra coisa ao retomar o projeto.

Versao documentada: **0.1.5**
Data: 2026-05-11
Ambiente: Go 1.22 / GoLand / Windows 10-11 x64 / PowerShell / Claude (Anthropic)

---

## 1. Objetivo do projeto

Criar um driver de impressora virtual para Windows que emula 100% uma
impressora matricial Epson FX-80. Qualquer aplicativo Windows pode imprimir
nela normalmente. Os dados sao capturados, processados e salvos como PDF
com visual fiel ao papel continuo da epoca.

Trabalho em curso. A emulacao completa do ESC/P esta no roadmap.

---

## 2. Decisoes de arquitetura

### 2.1 Driver sem kernel-mode

Drivers kernel-mode exigem assinatura WHQL e WDK. Usamos o driver
`Generic / Text Only` ja presente no Windows com a porta redirecionada
para um named pipe.

### 2.2 Named pipe como porta de impressora

O spooler chama `CreateFile()` com o nome da porta. `CreateFile("\\.\pipe\nome")`
abre um named pipe. Logo, o nome da porta pode ser o path do pipe diretamente.

Sequencia obrigatoria (Add com porta generica, depois Set):
```powershell
Add-Printer -Name "..." -PortName "PORTPROMPT:"
Set-Printer  -Name "..." -PortName "\\.\pipe\epson_fx80_emulator"
```
`Add-PrinterPort` com nome de pipe falha em alguns sistemas.

### 2.3 Fontes TTF com familias unicas no fpdf

O fpdf mapeia Bold/Italic como estilos de uma familia. Se a Regular, Bold e
Italic forem arquivos TTF diferentes, registrar como estilos da mesma familia
faz o fpdf ignorar os arquivos Bold/Italic. A solucao correta:

```go
pdf.AddUTF8Font("TTF_Regular", "", pathRegular)  // familia unica
pdf.AddUTF8Font("TTF_Bold",    "", pathBold)      // familia unica
pdf.AddUTF8Font("TTF_Italic",  "", pathItalic)    // familia unica
// Para usar: pdf.SetFont("TTF_Bold", "", size)
```

### 2.4 Icone do system tray deve ser PNG

Fyne no Windows nao renderiza SVG como icone de bandeja. PNG 256x256
embutido em base64 no codigo Go (ui/icon.go).

### 2.5 Scripts .ps1 em ASCII puro

Caracteres UTF-8 (travessao, checkmark, linhas de caixa) causam erros
de parse no PowerShell. Todos os .ps1 devem ser ASCII puro (<= 0x7F).

### 2.6 Registro via reg.exe

`New-Item` / `Set-ItemProperty` para HKLM falha em alguns contextos mesmo
como admin. Usar `reg.exe add` diretamente funciona de forma consistente.

### 2.7 ldflags para versao e build stamp

```powershell
$BuildHex  = "0x{0:X}" -f [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$LDVersion = "-X main.Version=0.1.5 -X main.BuildStamp=$BuildHex"
$UILDFlags = "$LDVersion -H windowsgui"
& go build -ldflags $UILDFlags -o dist\ui.exe .\ui\
```
Usar `& go build` (operador `&`) e variavel intermediaria `$UILDFlags`
evita o erro "parameter may not start with quote character".

---

## 3. Estrutura de arquivos atual

```
epson-fx80-emulator/
    go.mod                   modulo: github.com/epson-fx80-emulator
    build.ps1                compila; -CleanService para deploy
    README.md / MANUAL.md / OUTLINE.md / CHANGELOG.md / REFERENCE.md

    installer/
        install.ps1          instala/desinstala/status
        main.go              installer.exe
        fonts/ttf/           subpastas por modo (regular, bold, italic, ...)

    portmonitor/
        main.go              entry point; preloadFonts(); global fontManager
        service.go           svc.Handler para SCM
        monitor.go           loop named pipe; goroutine por job
        processor.go         bytesToText -> pdfgen.Generate + storage

    pdfgen/
        pdfgen.go            Generate(path, text, opts); registerFonts()
        options.go           LoadOptions() do registro
        testpage.go          GenerateTestPage(path, []FontEntry, versionLine)

    storage/
        storage.go           SQLite: Open, InsertJob, ListJobs, DeleteJob

    fontmgr/
        fontmgr.go           AllModes, SubfolderForMode, AvailableFonts, Save/Load

    ui/
        main.go              app Fyne; system tray; buildTrayMenu
        window.go            tabs: Historico, Configuracoes, Sobre; versionBar
        config.go            Config struct; loadConfig/saveConfig; executableDir
        icon.go              PNG 256x256 base64
        version.go           var Version, BuildStamp; FullVersion(); WindowTitle()
        testpage.go          generateTestPage(); buildFontEntries()
```

---

## 4. Fluxo completo de um job

```
1. App imprime para "Epson FX-80 Emulator"
2. Spooler -> CreateFile("\\.\pipe\epson_fx80_emulator")
3. monitor.go aceita conexao -> goroutine handleJob()
4. io.ReadAll() -> bytes do job
5. bytesToText() -> UTF-8; cleanControlChars() remove ESC/P
6. pdfgen.LoadOptions() -> papel, colunas, trator do registro
7. fontManager.Map injetado em opts.Fonts
8. pdfgen.Generate() -> PDF com TTFs por familia unica
9. storage.InsertJob() -> SQLite
10. ui.exe auto-refresh detecta novo job
```

---

## 5. Registro do Windows

Chave: `HKLM\SOFTWARE\EpsonFX80Emulator`

| Valor | Tipo | Default |
|---|---|---|
| OutputDir | REG_SZ | Documentos\EpsonFX80 |
| PaperType | REG_DWORD | 0 (branco) |
| TractorFeed | REG_DWORD | 0 (sem) |
| Columns | REG_DWORD | 80 |
| MonitorExe | REG_SZ | caminho do portmonitor.exe |
| PrinterName | REG_SZ | Epson FX-80 Emulator |

Subchave: `HKLM\SOFTWARE\EpsonFX80Emulator\Fonts`

| Valor | Descricao |
|---|---|
| Regular | Caminho absoluto do TTF Regular |
| Bold | Caminho absoluto do TTF Bold |
| Italic | Caminho absoluto do TTF Italic |
| Condensed | ... |
| Expanded | ... |
| BoldItalic | ... |
| CondensedBold | ... |
| CondensedItalic | ... |
| ExpandedBold | ... |
| ExpandedItalic | ... |
| ExpandedBoldItalic | ... |
| CondensedBoldItalic | ... |

---

## 6. API dos pacotes

### pdfgen

```go
type PaperType int  // PaperWhite=0, PaperGreenZebra=1, PaperBlueZebra=2
type Columns   int  // Columns80=80, Columns132=132
type Options struct {
    Paper, Cols, TractorFeed, Fonts fontmgr.FontMap, VersionLine string
}
type FontEntry struct {
    Mode fontmgr.Mode; Label, FontFile, FontName string
}
func DefaultOptions() Options
func LoadOptions() Options
func Generate(path, text string, opts Options) (int, error)
func GenerateTestPage(path string, entries []FontEntry, versionLine string) (int, error)
```

### fontmgr

```go
type Mode string  // ModeRegular, ModeBold, ..., ModeCondensedBoldItalic
var AllModes []Mode
func ModeLabel(m Mode) string
func SubfolderForMode(m Mode) string
type Manager struct { FontsDir string; Map FontMap }
func NewManager(execDir string) *Manager
func (m *Manager) AvailableFonts(mode Mode) []string
func (m *Manager) AvailableFontNames(mode Mode) []string
func (m *Manager) SelectedFont(mode Mode) string
func (m *Manager) SetFontByName(mode Mode, name string)
func (m *Manager) Save() error
func (m *Manager) HasFontsDir() bool
```

### storage

```go
type Job struct { ID int64; Name, PDFPath string; Pages, ByteSize int; CreatedAt time.Time }
func Open(path string) (*DB, error)
func (db *DB) InsertJob(Job) error
func (db *DB) ListJobs(limit int) ([]Job, error)
func (db *DB) DeleteJob(id int64) error
func (db *DB) CountJobs() (int, error)
```

### portmonitor globals

```go
const ServiceName = "EpsonFX80Monitor"
const PipeName    = `\\.\pipe\epson_fx80_emulator`
var   fontManager *fontmgr.Manager  // pre-carregado em preloadFonts()
```

### ui globals

```go
var Version    = "0.1.5"      // injetado via -ldflags
var BuildStamp = "0xDEV00000" // injetado via -ldflags
func FullVersion() string     // "0.1.5 - build 0x6A01DF28"
func WindowTitle() string
```

---

## 7. Problemas conhecidos e solucoes

| Problema | Causa | Solucao |
|---|---|---|
| Icone cinza no tray | Fyne nao renderiza SVG no Windows | PNG base64 em icon.go |
| .ps1 com erro de parse | UTF-8 acima de 0x7F | ASCII puro em todos .ps1 |
| Registro nao grava | Token admin insuficiente | reg.exe add direto |
| Porta nao redireciona | Add-PrinterPort rejeita pipe | Adicionar com PORTPROMPT:, depois Set-Printer |
| ui.exe bloqueado no copy | Processo rodando | Stop-Process -Name "ui" antes de Copy-Item |
| "no Go files" no build | go build com path absoluto | ./ relativo + Set-Location |
| Fontes nao mudam | Familias com mesmo nome no fpdf | Familia unica por modo (TTF_Bold etc.) |
| ldflags com aspas | PowerShell re-interpreta aspas | Operador & + variavel $UILDFlags |

---

## 8. Estado da emulacao ESC/P

Atual: `cleanControlChars()` remove sequencias ESC/P sem executar.
Proximo: implementar execucao real comecando por ESC E/F (negrito) e ESC 4/5 (italico).

---

## 9. Como continuar o desenvolvimento

```powershell
# Ambiente
go version; gcc --version

# Baixar dependencias
cd C:\projeto
go mod tidy

# Compilar e instalar
.\build.ps1
cd installer; .\install.ps1

# Fluxo diario
.\build.ps1
.\build.ps1 -CleanService

# Debug
sc stop EpsonFX80Monitor
.\installer\portmonitor.exe -debug
Get-Content .\installer\portmonitor.log -Wait
```

### Convencoes

- `.ps1` em ASCII puro (sem UTF-8 > 0x7F)
- Comentarios em portugues; codigo em portugues/ingles
- Sem `fmt.Println` no portmonitor -- usar `log.Printf`
- Configs em `HKLM\SOFTWARE\EpsonFX80Emulator`
- PDFs em `OutputDir` com nome `YYYYMMDD_HHMMSS_jobNNNN.pdf`
- Erros nao-fatais: log + continue; fatais: `log.Fatalf`
- Familias TTF no fpdf: sempre `"TTF_" + string(mode)`

---

*OUTLINE v0.1.5 -- 2026-05-11*
