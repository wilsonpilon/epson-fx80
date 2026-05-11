# OUTLINE -- Epson FX-80 Emulator

Este documento descreve o historico de decisoes, a arquitetura atual e o
estado do projeto em detalhes suficientes para que um desenvolvedor ou IA
possa continuar o desenvolvimento a partir deste ponto sem perder contexto.

Versao documentada: **0.1.3**
Data: 2026-05-11
Ambiente: Go 1.22 / GoLand / Windows 10-11 x64 / PowerShell / Claude (Anthropic)

---

## 1. Objetivo do projeto

Criar um driver de impressora virtual para Windows que emula 100% uma
impressora matricial Epson FX-80. Qualquer aplicativo Windows pode imprimir
nela normalmente. Os dados sao capturados, processados e salvos como PDF
com visual fiel ao papel continuo da epoca (zebrado, furos de trator, fonte
monoespaco, 80 ou 132 colunas).

Trata-se de um trabalho em curso. A emulacao completa do conjunto de
comandos ESC/P ainda esta sendo implementada progressivamente.

---

## 2. Decisoes de arquitetura e por que foram tomadas

### 2.1 Por que nao usar um driver kernel-mode

Drivers kernel-mode no Windows exigem assinatura digital obrigatoria
(WHQL), compilacao com WDK e processo de instalacao complexo. A alternativa
adotada foi usar o driver `Generic / Text Only` que ja vem instalado no
Windows e redirecionar a porta de impressao para um named pipe. Simples,
sem dependencias externas e funciona em qualquer Windows 10/11.

### 2.2 Como o named pipe funciona como porta de impressora

O spooler do Windows, ao imprimir em uma "Local Port", chama `CreateFile()`
com o nome da porta. No Windows, `CreateFile("\\.\pipe\nome")` abre um
named pipe. Logo, se o nome da porta for o path do pipe, o spooler escreve
diretamente nele.

A porta deve ser configurada assim (nao via `Add-PrinterPort`, mas via
`Set-Printer` depois de adicionar com porta generica):

```powershell
Add-Printer -Name "Epson FX-80 Emulator" -DriverName "Generic / Text Only" -PortName "PORTPROMPT:"
Set-Printer  -Name "Epson FX-80 Emulator" -PortName "\\.\pipe\epson_fx80_emulator"
```

Tentativas de usar `Add-PrinterPort` com o nome do pipe falhavam em alguns
sistemas. A solucao estavel foi: adicionar com porta generica, depois
trocar com `Set-Printer`.

### 2.3 Linguagem e bibliotecas

| Componente | Escolha | Motivo |
|---|---|---|
| Linguagem | Go 1.22 | Preferencia do autor; binarios estaticos; sem runtime |
| Named pipe | go-winio | Unica biblioteca Go madura para named pipes no Windows |
| PDF | go-pdf/fpdf | Leve, sem dependencias nativas, suporte a Courier |
| Banco | mattn/go-sqlite3 | Padrao para SQLite em Go; requer CGo/GCC |
| UI | fyne.io/fyne v2.4.5 | Framework Go nativo para desktop; suporta system tray |
| Registro | golang.org/x/sys/windows/registry | Acesso direto ao registro sem cgo extra |
| Scripts | PowerShell | Ja presente no Windows; cmdlets de impressora nativos |

### 2.4 Icone do system tray deve ser PNG, nao SVG

O Fyne no Windows nao renderiza SVG como icone de bandeja -- aparece
quadrado cinza. A solucao foi converter o SVG para PNG 256x256 com cairosvg
e embutir o PNG em base64 diretamente no codigo Go (ui/icon.go).

### 2.5 Scripts PowerShell devem ser ASCII puro

Caracteres UTF-8 como travessao (U+2014), checkmark (U+2713) e linhas
de caixa (U+2500) causam erros de parse no PowerShell dependendo da
configuracao de encoding do terminal. Todos os .ps1 foram mantidos em
ASCII puro, substituindo esses caracteres por equivalentes ASCII.

### 2.6 Configuracoes no registro do Windows

Todas as configuracoes ficam em `HKLM\SOFTWARE\EpsonFX80Emulator`.
O uso de `New-Item` do PowerShell para criar chaves HKLM falha com
"registry access is not allowed" mesmo como admin em alguns contextos.
A solucao foi usar `reg.exe add` diretamente, que funciona de forma
consistente.

### 2.7 Servico Windows para o portmonitor

O portmonitor roda como servico Windows (`sc create ... start= auto`).
Isso garante que ele inicie automaticamente com o Windows e rode em
background. Tambem pode rodar no terminal com `-debug` para desenvolvimento.

---

## 3. Estrutura de arquivos atual

```
epson-fx80-emulator/
    go.mod                  modulo: github.com/epson-fx80-emulator
    build.ps1               compila e faz deploy; suporta -CleanService
    README.md               descricao geral e ambiente
    MANUAL.md               instrucoes operacionais completas
    CHANGELOG.md            historico de versoes
    OUTLINE.md              este arquivo

    installer/
        install.ps1         instala/desinstala a impressora (script principal)
        main.go             versao Go do installer (gera installer.exe)

    portmonitor/
        main.go             entry point; detecta servico vs terminal vs debug
        service.go          interface svc.Handler para o SCM do Windows
        monitor.go          loop do named pipe; aceita conexoes do spooler
        processor.go        orquestra: leitura -> limpeza ESC/P -> PDF -> SQLite

    pdfgen/
        pdfgen.go           gera PDF com opcoes de papel (zebrado, trator, colunas)
        options.go          le Options do registro do Windows

    storage/
        storage.go          SQLite: open, migrate, insert, list, delete, count

    ui/
        main.go             entry point; configura system tray e janela
        window.go           janela principal: aba Historico, Configuracoes, Sobre
        config.go           le/grava Config no registro; helpers (openFile, etc.)
        icon.go             PNG 256x256 da impressora matricial em base64
```

---

## 4. Fluxo completo de um job de impressao

```
1. Usuario imprime de qualquer aplicativo para "Epson FX-80 Emulator"
2. Spooler do Windows chama CreateFile("\\.\pipe\epson_fx80_emulator")
3. Spooler escreve os bytes do job no pipe
4. monitor.go aceita a conexao e chama handleJob() em goroutine
5. handleJob() le todos os bytes com io.ReadAll()
6. processor.go chama bytesToText() para converter bytes em UTF-8
7. cleanControlChars() remove sequencias ESC/P e caracteres de controle
8. pdfgen.LoadOptions() le as configuracoes de papel do registro
9. pdfgen.Generate() cria o PDF com fonte Courier, papel e trator configurados
10. PDF salvo em OutputDir com nome: YYYYMMDD_HHMMSS_jobNNNN.pdf
11. storage.InsertJob() registra o job no SQLite
12. ui.exe detecta o novo job no auto-refresh de 5 segundos e exibe na lista
```

---

## 5. Configuracoes no registro do Windows

Chave: `HKLM\SOFTWARE\EpsonFX80Emulator`

| Valor | Tipo | Descricao | Default |
|---|---|---|---|
| OutputDir | REG_SZ | Pasta onde os PDFs sao salvos | Documentos\EpsonFX80 |
| MonitorExe | REG_SZ | Caminho completo do portmonitor.exe | (definido pelo install.ps1) |
| PortName | REG_SZ | Nome da porta (informativo) | \\.\pipe\epson_fx80_emulator |
| PrinterName | REG_SZ | Nome da impressora (informativo) | Epson FX-80 Emulator |
| PaperType | REG_DWORD | 0=branco, 1=verde zebrado, 2=azul zebrado | 0 |
| TractorFeed | REG_DWORD | 0=sem trator, 1=com faixa e furos laterais | 0 |
| Columns | REG_DWORD | 80 ou 132 | 80 |

---

## 6. API publica dos pacotes Go

### pdfgen

```go
// Tipos
type PaperType int  // PaperWhite=0, PaperGreenZebra=1, PaperBlueZebra=2
type Columns   int  // Columns80=80, Columns132=132
type Options struct {
    Paper       PaperType
    Cols        Columns
    TractorFeed bool
}

// Funcoes
func DefaultOptions() Options
func LoadOptions() Options                              // le do registro
func Generate(pdfPath, text string, opts Options) (int, error)
```

### storage

```go
type Job struct {
    ID, Name, PDFPath string
    Pages, ByteSize   int
    CreatedAt         time.Time
}
type DB struct { ... }

func Open(path string) (*DB, error)
func (db *DB) Close() error
func (db *DB) InsertJob(j Job) error
func (db *DB) ListJobs(limit int) ([]Job, error)
func (db *DB) DeleteJob(id int64) error
func (db *DB) CountJobs() (int, error)
```

### portmonitor (package main -- nao importavel)

Constantes relevantes:
```go
const ServiceName = "EpsonFX80Monitor"
const PipeName    = `\\.\pipe\epson_fx80_emulator`
```

Funcoes principais:
```go
func runMonitorWithStop(stop <-chan struct{})  // loop principal do pipe
func processJob(jobID int, baseName string, data []byte) error
func bytesToText(data []byte) string
func cleanControlChars(s string) string
func skipEscSeq(runes []rune, i int) int
```

### ui (package main -- nao importavel)

```go
type Config struct {
    OutputDir   string
    PaperType   int   // 0=branco, 1=verde, 2=azul
    TractorFeed bool
    Columns     int   // 80 ou 132
}
func loadConfig() Config
func saveConfig(cfg Config) error
func printerIcon() fyne.Resource  // PNG 256x256 em base64
```

---

## 7. Scripts PowerShell

### build.ps1

```
.\build.ps1               -- compila portmonitor.exe, installer.exe, ui.exe para dist\
.\build.ps1 -CleanService -- para EpsonFX80Monitor, encerra ui.exe,
                             copia dist\ para installer\, reinicia tudo
```

Variaveis importantes:
- `$OutDir`      = `$PSScriptRoot\dist`
- `$InstDir`     = `$PSScriptRoot\installer`
- `$ServiceName` = `EpsonFX80Monitor`
- `$env:CGO_ENABLED = "1"` (obrigatorio para go-sqlite3)

### installer/install.ps1

```
.\install.ps1             -- instala a impressora
.\install.ps1 -Uninstall  -- desinstala tudo
.\install.ps1 -Status     -- mostra status atual
```

Variaveis importantes:
- `$PrinterName` = `Epson FX-80 Emulator`
- `$DriverName`  = `Generic / Text Only`
- `$PortName`    = `\\.\pipe\epson_fx80_emulator`
- `$ServiceName` = `EpsonFX80Monitor`
- `$MonitorExe`  = `$ScriptDir\portmonitor.exe`
- `$UIExe`       = `$ScriptDir\ui.exe`
- `$ShortcutPath`= `shell:startup\EpsonFX80UI.lnk`

---

## 8. Problemas conhecidos e solucoes ja aplicadas

| Problema | Causa | Solucao aplicada |
|---|---|---|
| Icone cinza no system tray | Fyne nao renderiza SVG como icone no Windows | PNG 256x256 em base64 em icon.go |
| Script PS1 com erro de parse | Caracteres UTF-8 no arquivo | Todos .ps1 em ASCII puro |
| Registro nao grava com New-Item | Token de admin insuficiente em alguns contextos | Usar reg.exe add diretamente |
| Porta nao redireciona para pipe | Add-PrinterPort rejeita nome de pipe | Adicionar com PORTPROMPT:, depois Set-Printer |
| Erro ao copiar ui.exe no -CleanService | ui.exe rodando e bloqueando o arquivo | Stop-Process -Name "ui" antes de Copy-Item |
| "no Go files" no build | go build com path absoluto no Windows | Usar ./ relativo apos Set-Location $PSScriptRoot |
| Servico nao recebe jobs | Porta apontava para FX80EMUL: literal | Trocar porta para o path do named pipe |

---

## 9. Estado atual da emulacao ESC/P

O `cleanControlChars()` em `portmonitor/processor.go` detecta e remove
sequencias ESC/P do fluxo de texto. Os comandos sao **ignorados** (nao
executados) -- apenas removidos para nao aparecerem como lixo no PDF.

Sequencias reconhecidas para remocao (tamanho fixo):
```
ESC @ ESC E ESC F ESC G ESC H ESC 4 ESC 5 ESC 6 ESC 7 ESC 8 ESC 9
ESC < ESC = ESC > ESC M ESC P ESC T ESC O  (0 bytes de parametro)
ESC W ESC A ESC J ESC N ESC Q ESC R ESC S ESC U ESC i ESC l ESC 3 ESC 1
(1 byte de parametro)
```

**Proximo passo na emulacao:** implementar execucao real dos comandos,
comecando pelos mais comuns: ESC E/F (negrito), ESC 4/5 (italico),
ESC W (expandido), ESC SI/DC2 (condensado).

---

## 10. Proximos passos recomendados

### Imediatos (versao 0.2.0)
1. Implementar execucao real dos comandos ESC/P basicos no `processor.go`
   - Negrito (ESC E / ESC F) -- usar `pdf.SetFont(..., "B", ...)`
   - Italico (ESC 4 / ESC 5) -- usar `pdf.SetFont(..., "I", ...)`
   - Expandido (ESC W 1/0) -- dobrar o tamanho da fonte
   - Condensado (ESC SI) -- reduzir o tamanho da fonte (17 cpi)
2. Notificacao na bandeja do sistema quando um novo PDF for gerado
3. Preview descritivo do PDF na UI antes de abrir

### Medio prazo (versao 0.3.0)
4. Graficos de pinos (bit image) -- ESC K, ESC L, ESC Y, ESC Z
5. Suporte a codepages internacionais -- ESC R n
6. Testes automatizados para o pdfgen com diferentes opcoes de papel

### Longo prazo (versao 1.0.0)
7. Emulacao 100% do ESC/P da FX-80 documentada no MANUAL.md
8. Suite de testes com jobs reais de sistemas da epoca (COBOL, dBase, etc.)

---

## 11. Como continuar o desenvolvimento

### Configurar o ambiente

```powershell
# 1. Instalar Go 1.22+ em go.dev/dl
# 2. Instalar TDM-GCC em jmeubank.github.io/tdm-gcc
# 3. Clonar/copiar o projeto para C:\seu-projeto\
# 4. Verificar ambiente
go version          # go1.22.x
gcc --version       # tdm64 10.3+

# 5. Baixar dependencias
cd C:\seu-projeto
go mod tidy

# 6. Compilar
.\build.ps1

# 7. Instalar a impressora (como Admin)
cd installer
.\install.ps1

# 8. Testar
# Abrir Bloco de Notas, digitar texto, Ctrl+P
# Selecionar "Epson FX-80 Emulator", imprimir
# PDF aparece em Documentos\EpsonFX80\
```

### Fluxo de desenvolvimento diario

```powershell
# Editar codigo no GoLand
# Compilar e atualizar servico
.\build.ps1
.\build.ps1 -CleanService

# Ver logs em tempo real
Get-Content .\installer\portmonitor.log -Wait

# Para desenvolvimento do portmonitor sem servico
sc stop EpsonFX80Monitor
.\installer\portmonitor.exe -debug
```

### Convencoes do projeto

- Todos os arquivos `.ps1` em ASCII puro (sem UTF-8 acima de 0x7F)
- Comentarios em portugues; codigo em ingles/portugues misturado (ok)
- Sem `fmt.Println` no portmonitor em producao -- usar `log.Printf`
- Toda configuracao persistente vai para `HKLM\SOFTWARE\EpsonFX80Emulator`
- PDF gerado sempre em `OutputDir` com nome `YYYYMMDD_HHMMSS_jobNNNN.pdf`
- Erros nao-fatais: logar e continuar; erros fatais: `log.Fatalf`

---

*OUTLINE gerado em 2026-05-11 para o Epson FX-80 Emulator v0.1.3*
*Desenvolvido com Go 1.22 + GoLand + Windows + PowerShell + Claude (Anthropic)*
