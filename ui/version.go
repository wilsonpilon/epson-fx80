// ui/version.go
// Versao e build injetadas em tempo de compilacao via -ldflags.
//
// Exemplo de uso no build.ps1:
//   $buildTime = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
//   $buildHex  = "0x{0:X}" -f $buildTime
//   go build -ldflags "-X main.Version=0.1.3 -X main.BuildStamp=$buildHex -H windowsgui" ...
//
// Se nao injetados (ex: go run), usa valores de desenvolvimento.

package main

import "fmt"

// Version e a versao semantica do programa (ex: 0.1.3).
// Injetada via: -ldflags "-X main.Version=0.1.3"
var Version = "0.1.5"

// BuildStamp e o Unix timestamp da compilacao em hexadecimal (ex: 0x6820A4F2).
// Injetada via: -ldflags "-X main.BuildStamp=0x6820A4F2"
var BuildStamp = "0xDEV00000"

// FullVersion retorna a string completa de versao para exibicao.
// Formato: "0.1.3 - build 0x6820A4F2"
func FullVersion() string {
	return fmt.Sprintf("%s - build %s", Version, BuildStamp)
}

// WindowTitle retorna o titulo da janela principal com a versao.
func WindowTitle() string {
	return fmt.Sprintf("Epson FX-80 Emulator %s", FullVersion())
}
