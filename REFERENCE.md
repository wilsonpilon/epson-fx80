# Epson FX-80 Emulator -- Referencia Rapida

Versao 0.1.5

---

## Compilacao

```powershell
go mod tidy                    # baixa dependencias
.\build.ps1                    # compila portmonitor.exe, installer.exe, ui.exe
.\build.ps1 -CleanService      # para servico + ui.exe, copia dist\ para installer\, reinicia
```

---

## Instalacao e desinstalacao

```powershell
cd installer
.\install.ps1                  # instala impressora, servico e atalho de inicializacao
.\install.ps1 -Uninstall       # remove tudo
.\install.ps1 -Status          # exibe estado atual
```

---

## Servico Windows

```powershell
sc query  EpsonFX80Monitor     # status: RUNNING / STOPPED
sc start  EpsonFX80Monitor     # iniciar servico
sc stop   EpsonFX80Monitor     # parar servico
sc qc     EpsonFX80Monitor     # configuracao (path do binario, tipo de start)
sc delete EpsonFX80Monitor     # remover servico (nao remove a impressora)

# Reinstalar o servico manualmente
sc create EpsonFX80Monitor binPath= "C:\projeto\installer\portmonitor.exe" start= auto obj= LocalSystem
sc start  EpsonFX80Monitor
```

---

## Impressora

```powershell
# Verificar
Get-Printer -Name "Epson FX-80 Emulator" | Format-List Name, DriverName, PortName, PrinterStatus
(Get-Printer -Name "Epson FX-80 Emulator").PortName    # deve ser \\.\pipe\epson_fx80_emulator

# Corrigir porta (se necessario)
Set-Printer -Name "Epson FX-80 Emulator" -PortName "\\.\pipe\epson_fx80_emulator"

# Jobs na fila (deve estar vazio quando o servico esta rodando)
Get-PrintJob -PrinterName "Epson FX-80 Emulator"

# Limpar fila
Get-PrintJob -PrinterName "Epson FX-80 Emulator" | Remove-PrintJob
```

---

## Modo debug (sem servico)

```powershell
sc stop EpsonFX80Monitor
.\installer\portmonitor.exe -debug      # logs ao vivo, Ctrl+C para encerrar
```

---

## Logs

```powershell
# Tempo real
Get-Content .\installer\portmonitor.log -Wait

# Ultimas N linhas
Get-Content .\installer\portmonitor.log -Tail 50
Get-Content .\installer\ui.log          -Tail 20

# Limpar log
Clear-Content .\installer\portmonitor.log
```

---

## Registro do Windows

```powershell
# Ler todas as configuracoes
reg query "HKLM\SOFTWARE\EpsonFX80Emulator"
reg query "HKLM\SOFTWARE\EpsonFX80Emulator\Fonts"

# Papel
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "PaperType"   /t REG_DWORD /d 0 /f   # 0=branco
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "PaperType"   /t REG_DWORD /d 1 /f   # 1=verde
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "PaperType"   /t REG_DWORD /d 2 /f   # 2=azul

# Colunas
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "Columns"     /t REG_DWORD /d 80  /f
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "Columns"     /t REG_DWORD /d 132 /f

# Trator
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "TractorFeed" /t REG_DWORD /d 0 /f   # sem
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "TractorFeed" /t REG_DWORD /d 1 /f   # com

# Pasta de saida
reg add "HKLM\SOFTWARE\EpsonFX80Emulator" /v "OutputDir"   /t REG_SZ /d "D:\MeusPDFs" /f

# Fontes (subchave separada)
$f = "HKLM\SOFTWARE\EpsonFX80Emulator\Fonts"
reg add $f /v "Regular"   /t REG_SZ /d "C:\...\fonts\ttf\regular\fonte.ttf"   /f
reg add $f /v "Bold"      /t REG_SZ /d "C:\...\fonts\ttf\bold\fonte-bold.ttf" /f
reg add $f /v "Italic"    /t REG_SZ /d "C:\...\fonts\ttf\italic\fonte-it.ttf" /f
```

---

## Interface grafica (ui.exe)

```powershell
# Iniciar manualmente
.\installer\ui.exe

# Encerrar processo
Stop-Process -Name "ui" -Force

# Atalho de inicializacao
explorer shell:startup          # abre a pasta; EpsonFX80UI.lnk deve estar aqui

# Adicionar atalho manualmente
$wsh = New-Object -ComObject WScript.Shell
$sc  = $wsh.CreateShortcut("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\EpsonFX80UI.lnk")
$sc.TargetPath = "C:\projeto\installer\ui.exe"
$sc.WindowStyle = 7
$sc.Save()
```

---

## Diagnostico rapido

```powershell
# Pipeline de diagnostico completo
Write-Host "=== Impressora ==="
Get-Printer -Name "Epson FX-80 Emulator" | Select-Object Name, DriverName, PortName, PrinterStatus

Write-Host "`n=== Servico ==="
sc query EpsonFX80Monitor

Write-Host "`n=== Fila ==="
Get-PrintJob -PrinterName "Epson FX-80 Emulator"

Write-Host "`n=== Ultimas linhas do log ==="
Get-Content .\installer\portmonitor.log -Tail 10 -ErrorAction SilentlyContinue

Write-Host "`n=== Registro ==="
reg query "HKLM\SOFTWARE\EpsonFX80Emulator"
```

---

## Fontes TTF -- subpastas em installer\fonts\ttf\

| Subpasta | Modo |
|---|---|
| `regular\` | Regular |
| `bold\` | Negrito (Bold) |
| `italic\` | Italico (Italic) |
| `condensed\` | Condensado |
| `expanded\` | Expandido |
| `bold-italic\` | Negrito + Italico |
| `condensed-bold\` | Condensado + Negrito |
| `condensed-italic\` | Condensado + Italico |
| `expanded-bold\` | Expandido + Negrito |
| `expanded-italic\` | Expandido + Italico |
| `expanded-bold-italic\` | Expandido + Negrito + Italico |
| `condensed-bold-italic\` | Condensado + Negrito + Italico |

Apos alterar fontes: parar e reiniciar o servico para recarregar.

---

## Binarios e localizacao

| Binario | Localizacao padrao | Funcao |
|---|---|---|
| `portmonitor.exe` | `installer\` | Servico Windows |
| `installer.exe` | `installer\` | Alternativa ao install.ps1 |
| `ui.exe` | `installer\` | Interface grafica |
| `epson_fx80.db` | Ao lado do portmonitor.exe | Banco SQLite de jobs |
| `portmonitor.log` | Ao lado do portmonitor.exe | Log do servico |
| `ui.log` | Ao lado do ui.exe | Log da UI |

---

## Versao e build stamp

```powershell
# Ver versao atual do binario compilado (no rodape da janela ou titulo)
# Formato: 0.1.5 - build 0x6A01DF28

# Calcular build stamp manualmente
$ts  = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$hex = "0x{0:X}" -f $ts
Write-Host "$hex  ($ts)"
```

---

*Epson FX-80 Emulator v0.1.5 -- Go 1.22 / Windows / PowerShell / Claude (Anthropic)*
