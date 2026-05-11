# build.ps1
# Uso:
#   .\build.ps1                - compila todos os binarios para dist\
#   .\build.ps1 -CleanService  - para servico, copia dist\ para installer\, reinicia servico

param(
    [switch]$CleanService
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$OutDir      = Join-Path $PSScriptRoot "dist"
$InstDir     = Join-Path $PSScriptRoot "installer"
$ServiceName = "EpsonFX80Monitor"

function Write-Step($msg) { Write-Host $msg -NoNewline }
function Write-OK         { Write-Host " [OK]"  -ForegroundColor Green }
function Write-Warn($m)   { Write-Host " ($m)"  -ForegroundColor Yellow }
function Write-Fail($m)   { Write-Host " [ERRO]: $m" -ForegroundColor Red }

# -- Modo -CleanService --------------------------------------------------------

if ($CleanService) {
    Write-Host ""
    Write-Host "=== CleanService: $ServiceName ===" -ForegroundColor Cyan
    Write-Host ""

    Write-Step "Parando servico..."
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc -and $svc.Status -eq 'Running') {
        sc.exe stop $ServiceName | Out-Null
        Start-Sleep -Seconds 2
        Write-OK
    } else {
        Write-Warn "nao estava rodando"
    }

    Write-Step "Encerrando ui.exe..."
    $uiProc = Get-Process -Name "ui" -ErrorAction SilentlyContinue
    if ($uiProc) {
        $uiProc | Stop-Process -Force
        Start-Sleep -Milliseconds 800
        Write-OK
    } else {
        Write-Warn "nao estava rodando"
    }

    Write-Host ""
    foreach ($file in @("portmonitor.exe", "installer.exe", "ui.exe")) {
        $src = Join-Path $OutDir $file
        $dst = Join-Path $InstDir $file
        if (-not (Test-Path $src)) {
            Write-Host "  $file : nao encontrado em dist\ (compile primeiro com .\build.ps1)" -ForegroundColor Yellow
            continue
        }
        Write-Step "Copiando $file para installer\..."
        Copy-Item -Path $src -Destination $dst -Force
        Write-OK
    }

    Write-Host ""
    Write-Step "Iniciando servico..."
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc) {
        sc.exe start $ServiceName | Out-Null
        Write-OK
    } else {
        Write-Warn "servico nao instalado - rode installer\install.ps1 primeiro"
    }

    Write-Step "Iniciando ui.exe..."
    $uiExe = Join-Path $InstDir "ui.exe"
    if (Test-Path $uiExe) {
        Start-Process $uiExe
        Write-OK
    } else {
        Write-Warn "ui.exe nao encontrado em installer\"
    }

    Write-Host ""
    Write-Host "[OK] CleanService concluido." -ForegroundColor Green
    Write-Host ""
    exit 0
}

# -- Modo build normal ---------------------------------------------------------

New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

# Calcula build stamp: Unix timestamp em hexadecimal (ex: 0x6820A4F2)
$BuildUnix  = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$BuildHex   = "0x{0:X}" -f $BuildUnix
$Version    = "0.1.3"
$LDVersion  = "-X main.Version=$Version -X main.BuildStamp=$BuildHex"

Write-Host ""
Write-Host "=== Build: Epson FX-80 Emulator ===" -ForegroundColor Cyan
Write-Host "Raiz    : $PSScriptRoot"
Write-Host "Saida   : $OutDir"
Write-Host "Versao  : $Version"
Write-Host "Build   : $BuildHex  ($BuildUnix)"
Write-Host ""

$env:GOOS        = "windows"
$env:GOARCH      = "amd64"
$env:CGO_ENABLED = "1"

function Build($pkg, $out, $ldflags) {
    Write-Step "Compilando $out..."
    $outPath = Join-Path $OutDir $out
    if ($ldflags) {
        $result = go build -ldflags $ldflags -o $outPath "./$pkg" 2>&1
    } else {
        $result = go build -o $outPath "./$pkg" 2>&1
    }
    if ($LASTEXITCODE -ne 0) {
        Write-Host " [ERRO]" -ForegroundColor Red
        Write-Host $result
        exit 1
    }
    Write-OK
}

Write-Step "Baixando dependencias..."
$r = go mod download 2>&1
if ($LASTEXITCODE -ne 0) { Write-Host " [ERRO]" -ForegroundColor Red; Write-Host $r; exit 1 }
Write-OK

Build "portmonitor" "portmonitor.exe" ""
Build "installer"   "installer.exe"   ""
Build "ui"          "ui.exe"          "`"$LDVersion -H windowsgui`""

Write-Host ""
Write-Host "----------------------------------------------------"
Write-Host "[OK] Build concluido!" -ForegroundColor Green
Write-Host "     Versao  : $Version - build $BuildHex" -ForegroundColor Cyan
Write-Host "     Binarios: $OutDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "Proximos passos:"
Write-Host "  Atualizar servico : .\build.ps1 -CleanService"
Write-Host "  Primeira instalacao (como Admin):"
Write-Host "    cd installer"
Write-Host "    .\install.ps1"
Write-Host ""
