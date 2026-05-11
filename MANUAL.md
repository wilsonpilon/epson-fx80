# Epson FX-80 Emulator -- Manual de Operacao

Versao 0.1.5 | Trabalho em curso

---

## Sumario

1. [Pre-requisitos](#1-pre-requisitos)
2. [Compilacao](#2-compilacao)
3. [Instalacao do driver](#3-instalacao-do-driver)
4. [Desinstalacao do driver](#4-desinstalacao-do-driver)
5. [Servico Windows (portmonitor)](#5-servico-windows-portmonitor)
6. [Interface grafica (ui.exe)](#6-interface-grafica-uiexe)
7. [Configuracoes de papel](#7-configuracoes-de-papel)
8. [Configuracao de fontes TTF](#8-configuracao-de-fontes-ttf)
9. [Pagina de teste](#9-pagina-de-teste)
10. [Diagnostico e logs](#10-diagnostico-e-logs)
11. [Referencia de comandos ESC/P](#11-referencia-de-comandos-escp)

---

## 1. Pre-requisitos

### Software obrigatorio

| Software | Versao minima | Download |
|---|---|---|
| Go | 1.22 | https://go.dev/dl/ |
| TDM-GCC (CGo para SQLite) | 10.3+ | https://jmeubank.github.io/tdm-gcc/ |
| Windows | 10 ou 11 (x64) | -- |
| PowerShell | 5.1+ | ja incluso no Windows 10/11 |

### Verificando o ambiente

```powershell
go version        # go version go1.22.x windows/amd64
gcc --version     # gcc (tdm64-1) 10.3.0 ou superior
Get-ExecutionPolicy -Scope LocalMachine
# Se "Restricted": Set-ExecutionPolicy RemoteSigned -Scope LocalMachine
```

---

## 2. Compilacao

### Estrutura de diretorios esperada

```
C:\seu-projeto\
    go.mod
    build.ps1
    installer\
        install.ps1
        fonts\ttf\          <- fontes TTF opcionais
            regular\
            bold\
            italic\
            condensed\
            expanded\
            bold-italic\
            condensed-bold\
            condensed-italic\
            expanded-bold\
            expanded-italic\
            expanded-bold-italic\
            condensed-bold-italic\
    portmonitor\
    pdfgen\
    storage\
    fontmgr\
    ui\
```

### Compilacao completa

```powershell
go mod tidy        # baixa dependencias
.\build.ps1        # compila para dist\ e mostra versao+build
```

O `build.ps1` calcula o build stamp automaticamente:

```
=== Build: Epson FX-80 Emulator ===
Versao  : 0.1.5
Build   : 0x6A01DF28  (1778507560)
```

### Compilacao individual

```powershell
go build -o dist\portmonitor.exe .\portmonitor\
go build -o dist\installer.exe   .\installer\
go build -ldflags "-X main.Version=0.1.5 -X main.BuildStamp=0x6A01DF28 -H windowsgui" -o dist\ui.exe .\ui\
```

### Atualizar producao

```powershell
.\build.ps1                # compila
.\build.ps1 -CleanService  # para servico + ui.exe, copia dist\ para installer\, reinicia
```

---

## 3. Instalacao do driver

### Pelo script PowerShell (recomendado)

```powershell
cd C:\seu-projeto\installer
.\install.ps1
```

Passos executados:

| Passo | Acao |
|---|---|
| 1 | Verifica driver `Generic / Text Only` |
| 2 | Cria `Documentos\EpsonFX80` |
| 3 | Registra porta `\\.\pipe\epson_fx80_emulator` |
| 4 | Adiciona impressora e aponta para o pipe |
| 5 | Grava `HKLM\SOFTWARE\EpsonFX80Emulator` |
| 6 | Instala e inicia `EpsonFX80Monitor` |
| 7 | Cria atalho em `shell:startup` |

Se a impressora ja existir, o script detecta e pede confirmacao antes de reinstalar.

### Manualmente

```powershell
# Adiciona com porta generica, depois aponta para o pipe
Add-Printer -Name "Epson FX-80 Emulator" -DriverName "Generic / Text Only" -PortName "PORTPROMPT:"
Set-Printer  -Name "Epson FX-80 Emulator" -PortName "\\.\pipe\epson_fx80_emulator"

# Registro
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "OutputDir"   /t REG_SZ    /d "%USERPROFILE%\Documents\EpsonFX80" /f
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "PaperType"   /t REG_DWORD /d 0 /f
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "TractorFeed" /t REG_DWORD /d 0 /f
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "Columns"     /t REG_DWORD /d 80 /f

# Servico
sc create EpsonFX80Monitor binPath= "C:\projeto\installer\portmonitor.exe" start= auto obj= LocalSystem
sc start EpsonFX80Monitor
```

---

## 4. Desinstalacao do driver

```powershell
.\install.ps1 -Uninstall
```

### Manualmente

```powershell
sc stop EpsonFX80Monitor
sc delete EpsonFX80Monitor
Remove-Printer     -Name "Epson FX-80 Emulator"
Remove-PrinterPort -Name "\\.\pipe\epson_fx80_emulator"
reg delete "HKLM\SOFTWARE\EpsonFX80Emulator" /f
Remove-Item "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\EpsonFX80UI.lnk"
```

---

## 5. Servico Windows (portmonitor)

O `portmonitor.exe` roda como servico `EpsonFX80Monitor`. Escuta o pipe,
le os bytes do job, limpa ESC/P, aplica fontes TTF e gera o PDF.

### Comandos

```powershell
sc query  EpsonFX80Monitor    # status atual
sc start  EpsonFX80Monitor    # iniciar
sc stop   EpsonFX80Monitor    # parar
sc qc     EpsonFX80Monitor    # configuracao do servico
sc delete EpsonFX80Monitor    # remover (nao remove a impressora)
```

### Modo debug (terminal)

```powershell
sc stop EpsonFX80Monitor
.\installer\portmonitor.exe -debug   # Ctrl+C para encerrar
```

### Estados

| Estado | Descricao |
|---|---|
| `RUNNING` | Pipe aberto, aguardando jobs |
| `STOPPED` | Jobs ficam na fila do spooler |
| `START_PENDING` | Iniciando (aguarde) |
| Nao instalado | Execute `install.ps1` |

---

## 6. Interface grafica (ui.exe)

Aparece como icone na bandeja. Nao abre janela ao iniciar.

### Menu da bandeja (botao direito)

| Item | Acao |
|---|---|
| Abrir gerenciador | Abre a janela principal |
| Abrir pasta de PDFs | Explorer na pasta de saida |
| Encerrar | Fecha o ui.exe |

### Aba Historico

| Elemento | Descricao |
|---|---|
| Lista de jobs | Data, nome, paginas, tamanho, acoes |
| Botao abrir | Abre o PDF no visualizador padrao |
| Botao deletar | Remove do historico (PDF nao e apagado) |
| Atualizar | Recarrega do banco SQLite |
| Abrir pasta | Explorer na pasta dos PDFs |
| Pagina de teste | Gera PDF de teste com todas as fontes |
| Limpar historico | Remove todos os registros |

A lista atualiza automaticamente a cada 5 segundos.

### Rodape da janela

Exibe a versao e build stamp em todas as abas:
```
Epson FX-80 Emulator  0.1.5 - 0x6A01DF28
```

### Inicializacao automatica

O `install.ps1` cria atalho em `shell:startup`. Para gerenciar:

```powershell
explorer shell:startup    # abre a pasta de inicializacao
```

---

## 7. Configuracoes de papel

Salvas em `HKLM\SOFTWARE\EpsonFX80Emulator`. Aplicadas a partir do proximo job.

### Pela UI

Bandeja > Abrir gerenciador > Configuracoes > secao "Configuracoes de papel"

### Tipos de papel

| Valor | Codigo | Descricao |
|---|---|---|
| Branco | 0 | Papel branco simples |
| Zebrado Verde | 1 | Faixas alternadas verde/branco por linha |
| Zebrado Azul | 2 | Faixas alternadas azul/branco por linha |

### Colunas

| Valor | Uso tipico |
|---|---|
| 80 | Relatorios padrao, texto geral |
| 132 | Planilhas largas, listagens contabeis |

### Faixa de trator

Quando ativa: faixa cinza lateral com furos circulares, espacamento 12.7mm (0.5 pol).

### Pelo registro

```powershell
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "PaperType"   /t REG_DWORD /d 1 /f   # verde
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "TractorFeed" /t REG_DWORD /d 1 /f   # trator
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "Columns"     /t REG_DWORD /d 132 /f # 132 col
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "OutputDir"   /t REG_SZ    /d "D:\PDFs" /f
```

---

## 8. Configuracao de fontes TTF

Fontes TTF podem ser configuradas individualmente por modo de impressao.
Cada modo tem uma subpasta dedicada em `installer\fonts\ttf\`.

### Estrutura de pastas

```
installer\fonts\ttf\
    regular\              -> modo Regular
    bold\                 -> modo Bold (Negrito)
    italic\               -> modo Italic
    condensed\            -> modo Condensed
    expanded\             -> modo Expanded
    bold-italic\          -> modo Bold + Italic
    condensed-bold\       -> modo Condensed + Bold
    condensed-italic\     -> modo Condensed + Italic
    expanded-bold\        -> modo Expanded + Bold
    expanded-italic\      -> modo Expanded + Italic
    expanded-bold-italic\ -> modo Expanded + Bold + Italic
    condensed-bold-italic\-> modo Condensed + Bold + Italic
```

Coloque os arquivos `.ttf` ou `.otf` na subpasta do modo desejado.
A UI lista automaticamente os arquivos encontrados.

### Pela UI

Configuracoes > secao "Fontes TTF por modo de impressao" > selecione o arquivo por modo > Salvar

Apos salvar, reinicie o servico para aplicar:

```powershell
sc stop  EpsonFX80Monitor
sc start EpsonFX80Monitor
```

### Pelo registro

```powershell
$base = "HKLM\SOFTWARE\EpsonFX80Emulator\Fonts"
reg add $base /v "Regular"   /t REG_SZ /d "C:\...\fonts\ttf\regular\epson.ttf" /f
reg add $base /v "Bold"      /t REG_SZ /d "C:\...\fonts\ttf\bold\epson-bold.ttf" /f
reg add $base /v "Italic"    /t REG_SZ /d "C:\...\fonts\ttf\italic\epson-italic.ttf" /f
```

### Como o fpdf usa as fontes

Cada modo e registrado com uma familia unica no fpdf:
- `TTF_Regular` -> arquivo Regular
- `TTF_Bold`    -> arquivo Bold
- `TTF_Italic`  -> arquivo Italic
- etc.

Isso garante que `SetFont("TTF_Bold", "", size)` use o arquivo bold real,
sem depender de mapeamento de estilos da mesma familia.

---

## 9. Pagina de teste

Gera um PDF de diagnostico completo para verificar fontes e configuracoes.

### Pela UI

Aba Historico > botao "Pagina de teste"

Apos gerar, um dialogo pergunta se deseja abrir o PDF imediatamente.
O job e registrado no historico como qualquer outro.

### Conteudo do PDF

1. **Cabecalho**: versao, build stamp, data/hora
2. **Regua de 80 colunas**: dois digitos (dezena/unidade) + marcadores em 5 e 10
3. **ASCII imprimivel**: caracteres 32-126 em linhas de 64
4. **Bloco por modo** (12 modos):
   - Identificador do modo e nome do arquivo TTF
   - Frase: `Epson FX-80 Emulator Driver (Go)`
   - Digitos e simbolos: `1234567890 !@#$%^&*() ABC...abc...`

---

## 10. Diagnostico e logs

### Verificacao geral

```powershell
.\installer\install.ps1 -Status
Get-Printer -Name "Epson FX-80 Emulator" | Format-List *
(Get-Printer -Name "Epson FX-80 Emulator").PortName
Get-PrintJob -PrinterName "Epson FX-80 Emulator"
```

### Logs

| Arquivo | Conteudo |
|---|---|
| `portmonitor.log` | Jobs recebidos, fontes carregadas, erros |
| `ui.log` | Erros da interface grafica |

```powershell
Get-Content .\installer\portmonitor.log -Wait      # tempo real
Get-Content .\installer\portmonitor.log -Tail 50   # ultimas 50 linhas
```

### Problemas comuns

**Impressora aparece mas nao gera PDF**
```powershell
sc query EpsonFX80Monitor
# Se STOPPED: sc start EpsonFX80Monitor
(Get-Printer -Name "Epson FX-80 Emulator").PortName
# Se diferente de \\.\pipe\epson_fx80_emulator:
Set-Printer -Name "Epson FX-80 Emulator" -PortName "\\.\pipe\epson_fx80_emulator"
```

**Fontes nao mudam entre modos**
```powershell
# Verificar se o servico foi reiniciado apos salvar as fontes
sc stop  EpsonFX80Monitor
sc start EpsonFX80Monitor
Get-Content .\installer\portmonitor.log -Tail 20
# Deve mostrar linhas "[fonts] Regular -> arquivo.ttf"
```

**Erro ao compilar: CGo not enabled**
```powershell
$env:CGO_ENABLED = "1"
go build .\portmonitor\
```

---

## 11. Referencia de comandos ESC/P

> Esta secao sera expandida conforme a emulacao for implementada.

### Estado atual

O `processor.go` detecta e remove sequencias ESC/P do fluxo de texto.
Os comandos sao reconhecidos mas ainda nao executados (sem efeito no PDF).

### Comandos reconhecidos (limpeza apenas)

| Comando | Bytes | Descricao |
|---|---|---|
| Reset | ESC @ | Inicializa a impressora |
| Negrito on/off | ESC E / ESC F | Ativa/desativa negrito |
| Dupla impressao | ESC G / ESC H | Ativa/desativa dupla passagem |
| Italico on/off | ESC 4 / ESC 5 | Ativa/desativa italico |
| Sublinhado | ESC - 1/0 | Ativa/desativa sublinhado |
| Expandido | ESC W 1/0 | Largura dupla on/off |
| Condensado on | ESC SI (0x0F) | 17 cpi |
| Condensado off | ESC DC2 (0x12) | Retorna ao normal |
| Avanco de linha | ESC J n | Avanca n/216 polegadas |
| Espacamento | ESC A n | Define n/72 pol por linha |
| Espacamento padrao | ESC 2 | Restaura 1/6 pol |

### Comandos a implementar

| Comando | Bytes | Descricao |
|---|---|---|
| Bit image modo 0 | ESC K n1 n2 data | Graficos 60 dpi |
| Bit image modo 1 | ESC L n1 n2 data | Graficos 120 dpi |
| Bit image modo 2 | ESC Y n1 n2 data | Graficos 120 dpi alta vel |
| Bit image modo 3 | ESC Z n1 n2 data | Graficos 240 dpi |
| Tab horizontal | HT (0x09) | Proxima parada de tab |
| Definir tabs | ESC D n... NUL | Define paradas |
| Margem esquerda | ESC l n | Define margem |
| Margem direita | ESC Q n | Define margem |
| Comprimento pagina | ESC C n | Linhas por pagina |
| Codepage | ESC R n | Tabela de caracteres internacionais |

---

*Manual do Epson FX-80 Emulator v0.1.5*
*Go 1.22 / GoLand / Windows / PowerShell / Claude (Anthropic)*
