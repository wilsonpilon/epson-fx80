# install.ps1
# Uso: .\install.ps1 | .\install.ps1 -Uninstall | .\install.ps1 -Status
# Execute como Administrador.

param(
    [switch]$Uninstall,
    [switch]$Status
)

$PrinterName = "Epson FX-80 Emulator"
$DriverName  = "Generic / Text Only"
$PortName    = "\\.\pipe\epson_fx80_emulator"
$ServiceName = "EpsonFX80Monitor"
$ScriptDir   = Split-Path -Parent $MyInvocation.MyCommand.Path
$OutputDir   = Join-Path ([Environment]::GetFolderPath("MyDocuments")) "EpsonFX80"
$MonitorExe  = Join-Path $ScriptDir "portmonitor.exe"
$UIExe       = Join-Path $ScriptDir "ui.exe"
$StartupDir  = [Environment]::GetFolderPath("Startup")
$ShortcutPath = Join-Path $StartupDir "EpsonFX80UI.lnk"

function Write-Step($n, $msg) { Write-Host "$n. $msg" -NoNewline }
function Write-OK              { Write-Host " [OK]"        -ForegroundColor Green }
function Write-Warn($m)        { Write-Host " [AVISO]: $m" -ForegroundColor Yellow }
function Write-Fail($m)        { Write-Host " [ERRO]: $m"  -ForegroundColor Red }

function Assert-Admin {
    $cur = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not $cur.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Host "ERRO: Execute como Administrador." -ForegroundColor Red
        exit 1
    }
}

# Cria atalho .lnk usando WScript.Shell
function New-Shortcut($target, $shortcutPath, $description) {
    $wsh = New-Object -ComObject WScript.Shell
    $sc  = $wsh.CreateShortcut($shortcutPath)
    $sc.TargetPath       = $target
    $sc.WorkingDirectory = Split-Path $target
    $sc.Description      = $description
    $sc.WindowStyle      = 7  # minimizado (fica so na bandeja)
    $sc.Save()
}

function Show-Status {
    Write-Host ""
    Write-Host "=== Status: $PrinterName ===" -ForegroundColor Cyan
    $p = Get-Printer -Name $PrinterName -ErrorAction SilentlyContinue
    if ($p) {
        Write-Host "Impressora : INSTALADA" -ForegroundColor Green
        Write-Host "Driver     : $($p.DriverName)"
        Write-Host "Porta      : $($p.PortName)"
    } else {
        Write-Host "Impressora : NAO INSTALADA" -ForegroundColor Red
    }
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc) { Write-Host "Servico    : $($svc.Status)" }
    else       { Write-Host "Servico    : NAO INSTALADO" -ForegroundColor Red }
    if (Test-Path $ShortcutPath) { Write-Host "Inicializacao: ATALHO CRIADO" -ForegroundColor Green }
    else                         { Write-Host "Inicializacao: sem atalho"    -ForegroundColor Yellow }
    Write-Host ""
}

function Install-Printer {
    Assert-Admin
    Write-Host ""
    Write-Host "=== Instalando $PrinterName ===" -ForegroundColor Cyan
    Write-Host ""

    # -- Deteccao de reinstalacao ------------------------------------------
    $existing = Get-Printer -Name $PrinterName -ErrorAction SilentlyContinue
    if ($existing) {
        Write-Host ""
        Write-Host "  [AVISO] A impressora '$PrinterName' ja esta instalada." -ForegroundColor Yellow
        Write-Host "  Porta atual : $($existing.PortName)"
        Write-Host "  Driver atual: $($existing.DriverName)"
        Write-Host ""
        $resp = Read-Host "  Deseja remover e reinstalar? (S/N)"
        if ($resp -notmatch '^[Ss]$') {
            Write-Host "  Instalacao cancelada pelo usuario." -ForegroundColor Yellow
            exit 0
        }
        Write-Host ""
        Write-Host "  Removendo instalacao anterior..." -NoNewline
        Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Milliseconds 500
        Remove-Printer     -Name $PrinterName -ErrorAction SilentlyContinue
        Remove-PrinterPort -Name $PortName    -ErrorAction SilentlyContinue
        # Remove atalho antigo de inicializacao
        Remove-Item -Path $ShortcutPath -ErrorAction SilentlyContinue
        Write-Host " [OK]" -ForegroundColor Green
        Write-Host ""
    }

    # 1. Driver
    Write-Step 1 "Verificando driver '$DriverName'..."
    $drv = Get-PrinterDriver -Name $DriverName -ErrorAction SilentlyContinue
    if (-not $drv) {
        Write-Warn "nao encontrado, tentando instalar..."
        $inf = "$env:SystemRoot\inf\ntprint.inf"
        rundll32 printui.dll,PrintUIEntry /ia /m $DriverName /h "x64" /v "Type 3 - User Mode" /f $inf
        Start-Sleep -Seconds 2
        $drv = Get-PrinterDriver -Name $DriverName -ErrorAction SilentlyContinue
        if (-not $drv) { Write-Fail "driver nao encontrado"; exit 1 }
        else { Write-OK }
    } else { Write-OK }

    # 2. Diretorio de saida
    Write-Step 2 "Criando diretorio '$OutputDir'..."
    try { New-Item -ItemType Directory -Path $OutputDir -Force -ErrorAction Stop | Out-Null; Write-OK }
    catch { Write-Warn $_.Exception.Message }

    # 3. Porta pipe
    Write-Step 3 "Configurando porta pipe..."
    $portExists = Get-PrinterPort -Name $PortName -ErrorAction SilentlyContinue
    if ($portExists) {
        Write-Host " (ja existe)" -ForegroundColor Yellow
    } else {
        try { Add-PrinterPort -Name $PortName -ErrorAction Stop; Write-OK }
        catch { Write-Host " (configurada via Set-Printer)" -ForegroundColor Yellow }
    }

    # 4. Impressora
    Write-Step 4 "Registrando impressora '$PrinterName'..."
    try {
        Add-Printer -Name $PrinterName -DriverName $DriverName -PortName "PORTPROMPT:" `
            -Comment "Emulador Epson FX-80 - Gera PDFs automaticamente" `
            -Location "Virtual" -ErrorAction Stop
        Set-Printer -Name $PrinterName -PortName $PortName -ErrorAction SilentlyContinue
        $check = (Get-Printer -Name $PrinterName).PortName
        if ($check -eq $PortName) {
            Write-OK
        } else {
            Write-Warn "porta=$check (esperado $PortName)"
            Write-Host "  -> Rode: Set-Printer -Name '$PrinterName' -PortName '$PortName'" -ForegroundColor DarkGray
        }
    } catch { Write-Fail $_.Exception.Message; exit 1 }

    # 5. Registro
    Write-Step 5 "Gravando configuracoes no registro..."
    $regKey = "HKLM\SOFTWARE\EpsonFX80Emulator"
    reg add $regKey /v "OutputDir"   /t REG_SZ /d $OutputDir   /f 2>&1 | Out-Null
    reg add $regKey /v "MonitorExe"  /t REG_SZ /d $MonitorExe  /f 2>&1 | Out-Null
    reg add $regKey /v "PortName"    /t REG_SZ /d $PortName     /f 2>&1 | Out-Null
    reg add $regKey /v "PrinterName" /t REG_SZ /d $PrinterName  /f 2>&1 | Out-Null
    Write-OK

    # 6. Servico portmonitor
    Write-Step 6 "Instalando servico portmonitor..."
    if (Test-Path $MonitorExe) {
        Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Milliseconds 500
        sc.exe create $ServiceName binPath= "`"$MonitorExe`"" DisplayName= "Epson FX-80 Port Monitor" start= auto obj= LocalSystem | Out-Null
        sc.exe start $ServiceName | Out-Null
        Write-OK
    } else {
        Write-Host " (portmonitor.exe nao encontrado - compile primeiro)" -ForegroundColor Yellow
        Write-Host "  Rode: sc create $ServiceName binPath= `"$MonitorExe`" start= auto" -ForegroundColor DarkGray
        Write-Host "         sc start $ServiceName" -ForegroundColor DarkGray
    }

    # 7. Atalho de inicializacao para ui.exe
    Write-Step 7 "Criando atalho de inicializacao para ui.exe..."
    if (Test-Path $UIExe) {
        try {
            New-Shortcut $UIExe $ShortcutPath "Epson FX-80 Emulator - Gerenciador de Impressao"
            Write-OK
            Write-Host "     -> $ShortcutPath" -ForegroundColor DarkGray
            # Inicia a UI ja agora sem esperar o proximo boot
            Start-Process $UIExe
        } catch {
            Write-Warn $_.Exception.Message
        }
    } else {
        Write-Host " (ui.exe nao encontrado - compile primeiro)" -ForegroundColor Yellow
        Write-Host "  Copie ui.exe para '$ScriptDir' e rode este script novamente." -ForegroundColor DarkGray
    }

    Write-Host ""
    Write-Host "----------------------------------------------------"
    Write-Host "[OK] Impressora '$PrinterName' instalada!" -ForegroundColor Green
    Write-Host "     PDFs em: $OutputDir"                  -ForegroundColor Cyan
    Write-Host "     UI:      $UIExe"                      -ForegroundColor Cyan
    Write-Host ""
}

function Uninstall-Printer {
    Assert-Admin
    Write-Host ""
    Write-Host "=== Desinstalando $PrinterName ===" -ForegroundColor Cyan
    Write-Host ""

    Write-Step 1 "Parando e removendo servico..."
    Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Write-OK

    Write-Step 2 "Removendo impressora..."
    try { Remove-Printer -Name $PrinterName -ErrorAction Stop; Write-OK } catch { Write-Warn $_.Exception.Message }

    Write-Step 3 "Removendo porta..."
    Remove-PrinterPort -Name $PortName -ErrorAction SilentlyContinue
    Write-OK

    Write-Step 4 "Removendo registro..."
    reg delete "HKLM\SOFTWARE\EpsonFX80Emulator" /f 2>&1 | Out-Null
    Write-OK

    Write-Step 5 "Removendo atalho de inicializacao..."
    if (Test-Path $ShortcutPath) {
        Remove-Item -Path $ShortcutPath -Force
        Write-OK
    } else {
        Write-Host " (nao encontrado)" -ForegroundColor Yellow
    }

    Write-Host ""
    Write-Host "[OK] Desinstalacao concluida." -ForegroundColor Green
    Write-Host ""
}

if ($Status)        { Show-Status }
elseif ($Uninstall) { Uninstall-Printer }
else                { Install-Printer }
