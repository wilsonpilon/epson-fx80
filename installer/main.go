// installer/main.go
// Uso: installer.exe install | uninstall | status
// Deve ser executado como Administrador.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	PrinterName = "Epson FX-80 Emulator"
	DriverName  = "Generic / Text Only"
	PortName    = "FX80EMUL:"
	ServiceName = "EpsonFX80Monitor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: installer.exe <install|uninstall|status>")
		os.Exit(1)
	}
	switch strings.ToLower(os.Args[1]) {
	case "install":
		if err := install(); err != nil {
			fmt.Fprintln(os.Stderr, "Erro:", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := uninstall(); err != nil {
			fmt.Fprintln(os.Stderr, "Erro:", err)
			os.Exit(1)
		}
	case "status":
		status()
	default:
		fmt.Println("Uso: installer.exe <install|uninstall|status>")
		os.Exit(1)
	}
}

func install() error {
	fmt.Println("=== Instalando", PrinterName, "===")

	fmt.Print("1. Verificando driver... ")
	if out, err := runPS(`Get-PrinterDriver -Name "Generic / Text Only"`); err != nil || strings.TrimSpace(out) == "" {
		fmt.Println("[nao encontrado]")
		return fmt.Errorf("driver 'Generic / Text Only' nao disponivel no sistema")
	}
	fmt.Println("[OK]")

	outputDir := defaultOutputDir()
	fmt.Print("2. Criando diretorio de saida... ")
	os.MkdirAll(outputDir, 0755)
	fmt.Println("[OK]")

	fmt.Print("3. Criando porta... ")
	runPS(fmt.Sprintf(`Add-PrinterPort -Name "%s" -ErrorAction SilentlyContinue`, PortName))
	fmt.Println("[OK]")

	fmt.Print("4. Adicionando impressora... ")
	runPS(fmt.Sprintf(`Remove-Printer -Name "%s" -ErrorAction SilentlyContinue`, PrinterName))
	_, err := runPS(fmt.Sprintf(
		`Add-Printer -Name "%s" -DriverName "%s" -PortName "%s" -Comment "Emulador Epson FX-80" -Location "Virtual"`,
		PrinterName, DriverName, PortName,
	))
	if err != nil {
		return fmt.Errorf("falha ao adicionar impressora: %w", err)
	}
	fmt.Println("[OK]")

	fmt.Print("5. Gravando registro... ")
	regKey := `HKLM\SOFTWARE\EpsonFX80Emulator`
	exec.Command("reg", "add", regKey, "/v", "OutputDir", "/t", "REG_SZ", "/d", outputDir, "/f").Run()
	monitorExe := filepath.Join(filepath.Dir(exePath()), "portmonitor.exe")
	exec.Command("reg", "add", regKey, "/v", "MonitorExe", "/t", "REG_SZ", "/d", monitorExe, "/f").Run()
	exec.Command("reg", "add", regKey, "/v", "PortName", "/t", "REG_SZ", "/d", PortName, "/f").Run()
	exec.Command("reg", "add", regKey, "/v", "PrinterName", "/t", "REG_SZ", "/d", PrinterName, "/f").Run()
	fmt.Println("[OK]")

	fmt.Print("6. Instalando servico... ")
	if _, err := os.Stat(monitorExe); err == nil {
		exec.Command("sc", "stop", ServiceName).Run()
		exec.Command("sc", "delete", ServiceName).Run()
		exec.Command("sc", "create", ServiceName,
			"binPath=", monitorExe,
			"DisplayName=", "Epson FX-80 Port Monitor",
			"start=", "auto",
			"obj=", "LocalSystem",
		).Run()
		exec.Command("sc", "start", ServiceName).Run()
		fmt.Println("[OK]")
	} else {
		fmt.Println("[portmonitor.exe nao encontrado - compile primeiro]")
	}

	fmt.Println()
	fmt.Println("[OK] Impressora instalada! PDFs em:", outputDir)
	return nil
}

func uninstall() error {
	fmt.Println("=== Desinstalando", PrinterName, "===")
	exec.Command("sc", "stop", ServiceName).Run()
	exec.Command("sc", "delete", ServiceName).Run()
	runPS(fmt.Sprintf(`Remove-Printer -Name "%s" -ErrorAction SilentlyContinue`, PrinterName))
	runPS(fmt.Sprintf(`Remove-PrinterPort -Name "%s" -ErrorAction SilentlyContinue`, PortName))
	exec.Command("reg", "delete", `HKLM\SOFTWARE\EpsonFX80Emulator`, "/f").Run()
	fmt.Println("[OK] Desinstalado.")
	return nil
}

func status() {
	out, _ := runPS(fmt.Sprintf(`Get-Printer -Name "%s" | Select-Object Name,DriverName,PortName | Format-List`, PrinterName))
	if strings.TrimSpace(out) == "" {
		fmt.Println("Impressora: NAO INSTALADA")
	} else {
		fmt.Println("Impressora: INSTALADA")
		fmt.Println(out)
	}
}

func runPS(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func exePath() string {
	p, _ := os.Executable()
	return p
}

func defaultOutputDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents", "EpsonFX80")
}
