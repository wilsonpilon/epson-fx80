# Epson FX-80 Emulator -- Manual de Operacao

Versao 1.0 | Trabalho em curso

---

## Sumario

1. [Pre-requisitos](#1-pre-requisitos)
2. [Compilacao](#2-compilacao)
3. [Instalacao do driver](#3-instalacao-do-driver)
4. [Desinstalacao do driver](#4-desinstalacao-do-driver)
5. [Servico Windows (portmonitor)](#5-servico-windows-portmonitor)
6. [Interface grafica (ui.exe)](#6-interface-grafica-uiexe)
7. [Configuracoes de papel](#7-configuracoes-de-papel)
8. [Diagnostico e logs](#8-diagnostico-e-logs)
9. [Referencia de comandos ESC/P](#9-referencia-de-comandos-escp)

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
# Verifica versao do Go
go version
# Esperado: go version go1.22.x windows/amd64

# Verifica GCC (necessario para go-sqlite3)
gcc --version
# Esperado: gcc (tdm64-1) 10.3.0 ou superior

# Verifica politica de execucao do PowerShell
Get-ExecutionPolicy -Scope LocalMachine
# Se retornar "Restricted", execute como Admin:
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope LocalMachine
```

---

## 2. Compilacao

### Estrutura de diretorios esperada

```
C:\seu-projeto\
    go.mod
    build.ps1
    README.md
    MANUAL.md
    installer\
        install.ps1
        main.go
    portmonitor\
        main.go  monitor.go  processor.go  service.go
    pdfgen\
        pdfgen.go  options.go
    storage\
        storage.go
    ui\
        main.go  window.go  config.go  icon.go
```

### Compilacao completa

```powershell
# Na raiz do projeto (onde esta o go.mod)
# Primeiro download das dependencias
go mod tidy

# Compila todos os binarios para dist\
.\build.ps1
```

Apos a compilacao, a pasta `dist\` contera:

```
dist\
    portmonitor.exe   -- servico que intercepta os jobs de impressao
    installer.exe     -- alternativa ao install.ps1
    ui.exe            -- interface grafica na bandeja do sistema
```

### Compilacao individual (opcional)

```powershell
# Apenas o portmonitor
go build -o dist\portmonitor.exe .\portmonitor\

# Apenas a UI (sem janela de terminal)
go build -ldflags "-H windowsgui" -o dist\ui.exe .\ui\

# Apenas o installer
go build -o dist\installer.exe .\installer\
```

### Atualizar binarios em producao

Quando o servico ja esta instalado e rodando, use o modo `-CleanService`
para parar tudo, copiar os novos binarios e reiniciar:

```powershell
# Compila e faz deploy automatico
.\build.ps1
.\build.ps1 -CleanService
```

O `-CleanService` executa na sequencia:
1. Para o servico `EpsonFX80Monitor`
2. Encerra o processo `ui.exe`
3. Copia `portmonitor.exe`, `installer.exe` e `ui.exe` de `dist\` para `installer\`
4. Reinicia o servico
5. Inicia o `ui.exe`

---

## 3. Instalacao do driver

### Pelo script PowerShell (recomendado)

```powershell
# Abra o PowerShell como Administrador
cd C:\seu-projeto\installer
.\install.ps1
```

O script executa os seguintes passos:

| Passo | Acao |
|---|---|
| 1 | Verifica se o driver `Generic / Text Only` esta disponivel no Windows |
| 2 | Cria o diretorio `Documentos\EpsonFX80` para salvar os PDFs |
| 3 | Registra a porta `\\.\pipe\epson_fx80_emulator` no spooler |
| 4 | Adiciona a impressora `Epson FX-80 Emulator` no Windows |
| 5 | Grava configuracoes em `HKLM\SOFTWARE\EpsonFX80Emulator` |
| 6 | Instala e inicia o servico `EpsonFX80Monitor` |
| 7 | Cria atalho do `ui.exe` na pasta de inicializacao do Windows |

Se a impressora ja estiver instalada, o script detecta e pergunta se deve reinstalar:

```
[AVISO] A impressora 'Epson FX-80 Emulator' ja esta instalada.
Porta atual : \\.\pipe\epson_fx80_emulator
Driver atual: Generic / Text Only

Deseja remover e reinstalar? (S/N):
```

### Pelo installer.exe (alternativa)

```powershell
# Como Administrador
.\installer\installer.exe install
```

### Manualmente (passo a passo)

Se preferir fazer cada etapa manualmente via PowerShell:

```powershell
# 1. Verificar se o driver generico existe
Get-PrinterDriver -Name "Generic / Text Only"

# 2. Criar o diretorio de saida
New-Item -ItemType Directory -Path "$env:USERPROFILE\Documents\EpsonFX80" -Force

# 3. Adicionar a impressora com porta temporaria
Add-Printer -Name "Epson FX-80 Emulator" `
    -DriverName "Generic / Text Only" `
    -PortName "PORTPROMPT:" `
    -Comment "Emulador Epson FX-80 - Gera PDFs automaticamente" `
    -Location "Virtual"

# 4. Apontar a porta para o named pipe
Set-Printer -Name "Epson FX-80 Emulator" `
    -PortName "\\.\pipe\epson_fx80_emulator"

# 5. Verificar se a porta foi aplicada
(Get-Printer -Name "Epson FX-80 Emulator").PortName
# Esperado: \\.\pipe\epson_fx80_emulator

# 6. Gravar configuracoes no registro
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "OutputDir" /t REG_SZ /d "$env:USERPROFILE\Documents\EpsonFX80" /f
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "PaperType"   /t REG_DWORD /d 0 /f
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "TractorFeed" /t REG_DWORD /d 0 /f
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "Columns"     /t REG_DWORD /d 80 /f

# 7. Instalar o servico
sc create EpsonFX80Monitor binPath= "C:\seu-projeto\installer\portmonitor.exe" DisplayName= "Epson FX-80 Port Monitor" start= auto obj= LocalSystem
sc start EpsonFX80Monitor

# 8. Criar atalho de inicializacao
$wsh = New-Object -ComObject WScript.Shell
$sc  = $wsh.CreateShortcut("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\EpsonFX80UI.lnk")
$sc.TargetPath = "C:\seu-projeto\installer\ui.exe"
$sc.WindowStyle = 7
$sc.Save()

# 9. Iniciar a UI
Start-Process "C:\seu-projeto\installer\ui.exe"
```

---

## 4. Desinstalacao do driver

### Pelo script PowerShell (recomendado)

```powershell
cd C:\seu-projeto\installer
.\install.ps1 -Uninstall
```

### Pelo installer.exe (alternativa)

```powershell
.\installer\installer.exe uninstall
```

### Manualmente

```powershell
# 1. Parar e remover o servico
sc stop EpsonFX80Monitor
sc delete EpsonFX80Monitor

# 2. Remover a impressora
Remove-Printer -Name "Epson FX-80 Emulator"

# 3. Remover a porta
Remove-PrinterPort -Name "\\.\pipe\epson_fx80_emulator"

# 4. Remover entradas do registro
reg delete "HKLM\SOFTWARE\EpsonFX80Emulator" /f

# 5. Remover atalho de inicializacao
Remove-Item "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\EpsonFX80UI.lnk" -Force
```

---

## 5. Servico Windows (portmonitor)

O `portmonitor.exe` roda como servico Windows sob o nome `EpsonFX80Monitor`.
Ele escuta o named pipe `\\.\pipe\epson_fx80_emulator`, recebe os dados de
impressao do spooler do Windows, converte em PDF e registra o job no SQLite.

### Comandos do servico

```powershell
# Verificar status
sc query EpsonFX80Monitor

# Iniciar
sc start EpsonFX80Monitor

# Parar
sc stop EpsonFX80Monitor

# Reiniciar
sc stop EpsonFX80Monitor
sc start EpsonFX80Monitor

# Remover (desinstala apenas o servico, nao a impressora)
sc stop EpsonFX80Monitor
sc delete EpsonFX80Monitor

# Verificar configuracao do servico
sc qc EpsonFX80Monitor
```

### Rodar em modo debug (sem instalar como servico)

Util durante o desenvolvimento para ver os logs em tempo real no terminal:

```powershell
# Para o servico se estiver rodando
sc stop EpsonFX80Monitor

# Roda diretamente no terminal (Ctrl+C para encerrar)
.\installer\portmonitor.exe -debug
```

No modo debug, todos os logs aparecem no terminal e tambem sao gravados
em `portmonitor.log` ao lado do executavel.

### Estados possiveis do servico

| Estado | Descricao |
|---|---|
| `RUNNING` | Servico ativo, pipe aberto, aguardando jobs |
| `STOPPED` | Servico parado, jobs ficam na fila do spooler |
| `START_PENDING` | Iniciando (aguarde alguns segundos) |
| `STOP_PENDING` | Parando (aguarde alguns segundos) |
| Nao instalado | Execute `install.ps1` ou `sc create` manualmente |

---

## 6. Interface grafica (ui.exe)

O `ui.exe` aparece como icone na bandeja do sistema (system tray).
Nao abre janela ao iniciar -- fica silencioso na bandeja ate ser chamado.

### Menu da bandeja (clique com botao direito no icone)

| Item | Acao |
|---|---|
| Abrir gerenciador | Abre a janela principal |
| Abrir pasta de PDFs | Abre o Explorer na pasta de saida |
| Encerrar | Fecha o ui.exe completamente |

### Janela principal -- aba Historico

Exibe todos os jobs de impressao registrados no banco SQLite.

| Coluna | Descricao |
|---|---|
| Data/Hora | Quando o job foi recebido |
| Arquivo | Nome base do arquivo PDF gerado |
| Paginas | Numero de paginas do PDF |
| Tamanho | Tamanho em bytes dos dados recebidos |
| Acoes | Botao abrir PDF / botao remover do historico |

Botoes da barra:
- **Atualizar** -- recarrega a lista do banco
- **Abrir pasta** -- abre o Explorer na pasta dos PDFs
- **Limpar historico** -- remove todos os registros (PDFs nao sao deletados)

A lista atualiza automaticamente a cada 5 segundos.

### Janela principal -- aba Configuracoes

Permite ajustar as configuracoes de papel e o diretorio de saida.
Veja a secao [7. Configuracoes de papel](#7-configuracoes-de-papel) para detalhes.

### Inicializacao automatica com o Windows

O `install.ps1` cria automaticamente um atalho em:

```
%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\EpsonFX80UI.lnk
```

Para adicionar ou remover manualmente:

```powershell
# Abrir a pasta de inicializacao no Explorer
explorer shell:startup

# Remover o atalho
Remove-Item "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\EpsonFX80UI.lnk"
```

---

## 7. Configuracoes de papel

As configuracoes sao salvas em `HKLM\SOFTWARE\EpsonFX80Emulator` e aplicadas
a partir do proximo job de impressao.

### Pela interface grafica

1. Clique com botao direito no icone da bandeja
2. Selecione **Abrir gerenciador**
3. Clique na aba **Configuracoes**
4. Ajuste as opcoes desejadas
5. Clique em **Salvar configuracoes**

### Opcoes disponiveis

#### Tipo de papel

| Valor | Descricao |
|---|---|
| Branco | Papel branco simples, sem faixas |
| Zebrado Verde | Faixas alternadas verde claro / branco, uma faixa por linha |
| Zebrado Azul | Faixas alternadas azul claro / branco, uma faixa por linha |

#### Largura em colunas

| Valor | Uso tipico |
|---|---|
| 80 colunas | Relatorios padrao, cartas, texto geral |
| 132 colunas | Planilhas largas, relatorios contabeis, listagens de sistema |

A fonte e calculada automaticamente para caber exatamente o numero
de colunas escolhido na largura do papel A4.

#### Faixa de trator

Quando ativada, exibe nos dois lados do papel:
- Uma faixa cinza simulando a margem destacavel do papel continuo
- Furos circulares brancos com espacamento de 12.7mm (0.5 polegada),
  igual ao espacamento real das impressoras matriciais

### Pelo registro do Windows (manual)

```powershell
# Tipo de papel: 0=branco, 1=verde, 2=azul
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "PaperType" /t REG_DWORD /d 1 /f

# Furos de trator: 0=nao, 1=sim
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "TractorFeed" /t REG_DWORD /d 1 /f

# Colunas: 80 ou 132
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "Columns" /t REG_DWORD /d 132 /f

# Pasta de saida dos PDFs
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "OutputDir" /t REG_SZ /d "D:\MeusPDFs" /f
```

---

## 8. Diagnostico e logs

### Verificacao geral

```powershell
# Status resumido de tudo
.\installer\install.ps1 -Status

# Status detalhado da impressora
Get-Printer -Name "Epson FX-80 Emulator" | Format-List *

# Jobs na fila do spooler (deve estar vazio se o servico esta rodando)
Get-PrintJob -PrinterName "Epson FX-80 Emulator"

# Porta configurada
(Get-Printer -Name "Epson FX-80 Emulator").PortName
# Esperado: \\.\pipe\epson_fx80_emulator
```

### Logs

| Arquivo | Localizado em | Conteudo |
|---|---|---|
| `portmonitor.log` | Mesma pasta do portmonitor.exe | Jobs recebidos, erros de pipe, geracao de PDF |
| `ui.log` | Mesma pasta do ui.exe | Erros da interface grafica |

```powershell
# Ver log do portmonitor em tempo real
Get-Content .\installer\portmonitor.log -Wait

# Ver ultimas 50 linhas
Get-Content .\installer\portmonitor.log -Tail 50
```

### Problemas comuns

**Impressora aparece mas nao gera PDF**

```powershell
# 1. Verificar se o servico esta rodando
sc query EpsonFX80Monitor
# Se STOPPED: sc start EpsonFX80Monitor

# 2. Verificar se a porta esta correta
(Get-Printer -Name "Epson FX-80 Emulator").PortName
# Se nao for \\.\pipe\epson_fx80_emulator:
Set-Printer -Name "Epson FX-80 Emulator" -PortName "\\.\pipe\epson_fx80_emulator"

# 3. Verificar o log apos tentar imprimir
Get-Content .\installer\portmonitor.log -Tail 20
```

**Servico nao inicia**

```powershell
# Verificar se o executavel existe
Test-Path .\installer\portmonitor.exe

# Verificar permissao do servico
sc qc EpsonFX80Monitor

# Tentar rodar em modo debug para ver o erro
.\installer\portmonitor.exe -debug
```

**Erro ao compilar: CGo not enabled**

```powershell
# Verificar se CGO esta habilitado
go env CGO_ENABLED
# Se retornar 0:
$env:CGO_ENABLED = "1"
go build .\portmonitor\
```

**Erro ao compilar: gcc not found**

Instale o TDM-GCC e reinicie o terminal. Verifique:
```powershell
gcc --version
```

---

## 9. Referencia de comandos ESC/P

> Esta secao sera expandida conforme a emulacao dos comandos ESC/P
> da Epson FX-80 for implementada no projeto.

A Epson FX-80 usa o conjunto de comandos **ESC/P** (Epson Standard Code
for Printers). Todos os comandos iniciam com o caractere ESC (0x1B)
seguido de um ou mais bytes de controle.

### Estado atual da emulacao

O `processor.go` atualmente faz limpeza basica das sequencias ESC/P --
os comandos sao detectados e removidos do fluxo de texto para nao
aparecerem como lixo no PDF. A execucao real dos comandos (mudanca de
fonte, espacamento, graficos) esta no roadmap.

### Comandos implementados (limpeza apenas)

Os comandos abaixo sao reconhecidos e removidos do texto. Ainda nao
alteram a formatacao do PDF gerado.

| Comando | Bytes | Descricao |
|---|---|---|
| Reset | ESC @ | Inicializa a impressora |
| Negrito on | ESC E | Ativa modo negrito |
| Negrito off | ESC F | Desativa modo negrito |
| Dupla impressao on | ESC G | Ativa dupla impressao |
| Dupla impressao off | ESC H | Desativa dupla impressao |
| Italico on | ESC 4 | Ativa modo italico |
| Italico off | ESC 5 | Desativa modo italico |
| Sublinhado on | ESC - 1 | Ativa sublinhado |
| Sublinhado off | ESC - 0 | Desativa sublinhado |
| Expandido on | ESC W 1 | Ativa modo expandido (largura dupla) |
| Expandido off | ESC W 0 | Desativa modo expandido |
| Condensado on | ESC SI | Ativa modo condensado (17 cpi) |
| Condensado off | ESC DC2 | Desativa modo condensado |
| Avanco de linha | ESC J n | Avanca n/216 polegadas |
| Espacamento de linha | ESC A n | Define espacamento (n/72 pol) |
| Espacamento padrao | ESC 2 | Restaura espacamento 1/6 pol |
| Pular perfuracao | ESC N n | Define margem de perfuracao |

### Comandos a implementar

| Comando | Bytes | Descricao |
|---|---|---|
| Bit image (modo 0) | ESC K n1 n2 data | Graficos de pinos 60 dpi |
| Bit image (modo 1) | ESC L n1 n2 data | Graficos de pinos 120 dpi |
| Bit image (modo 2) | ESC Y n1 n2 data | Graficos de pinos 120 dpi alta vel |
| Bit image (modo 3) | ESC Z n1 n2 data | Graficos de pinos 240 dpi |
| Tab horizontal | HT (0x09) | Avanca ate a proxima parada de tab |
| Definir tabs | ESC D n... NUL | Define paradas de tab |
| Margem esquerda | ESC l n | Define margem esquerda |
| Margem direita | ESC Q n | Define margem direita |
| Comprimento pagina | ESC C n | Define comprimento em linhas |
| Comprimento pol | ESC C NUL n | Define comprimento em polegadas |
| Selecao de caractere | ESC R n | Seleciona tabela de caracteres internacionais |
| Paginacao | FF (0x0C) | Avanca para o inicio da proxima pagina |

---

*Manual gerado para o projeto Epson FX-80 Emulator*
*Ambiente: Go 1.22 / GoLand / Windows / PowerShell / Claude (Anthropic)*
