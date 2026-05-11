# Epson FX-80 Emulator

Emulador de impressora matricial Epson FX-80 para Windows.

O projeto cria uma impressora virtual no Windows que qualquer aplicativo pode usar normalmente -- o sistema operacional a enxerga como uma impressora real. Os dados de impressao sao interceptados, processados e convertidos em arquivos PDF que reproduzem a estetica dos documentos da epoca: papel zebrado, furos de trator, fonte monoespaco e layout de 80 ou 132 colunas.

---

## Objetivo

Emular 100% o comportamento de uma impressora matricial Epson FX-80, incluindo:

- Recepcao de jobs de impressao de qualquer aplicativo Windows
- Interpretacao de comandos ESC/P (linguagem de controle da Epson FX-80)
- Geracao de PDFs fieis ao visual da epoca
- Suporte a papel continuo com furos de trator
- Suporte a papel zebrado verde ou azul (faixas alternadas por linha)
- Suporte a 80 e 132 colunas
- Interface de gerenciamento com historico de jobs

> **Trabalho em curso.** A interpretacao completa do conjunto de comandos ESC/P da FX-80 ainda esta sendo implementada. O estado atual processa texto simples com limpeza basica de sequencias de escape. A emulacao grafica completa (graficos de pinos, modos condensado/expandido/negrito via ESC) sera adicionada progressivamente.

---

## Como funciona

```
[Qualquer aplicativo Windows]
        |
        |  API de impressao do Windows
        v
[Driver: Generic / Text Only]
        |
        |  Porta: \\.\pipe\epson_fx80_emulator
        v
[portmonitor.exe  --  servico Windows]
        |
        |-- Le os dados brutos do job
        |-- Limpa sequencias de escape ESC/P
        |-- Aplica configuracoes de papel
        v
[pdfgen  --  gerador de PDF]
        |
        |-- Fonte Courier monoespaco
        |-- Zebrado por linha (verde ou azul)
        |-- Faixa de trator com furos laterais
        |-- 80 ou 132 colunas
        v
[PDF salvo em Documentos\EpsonFX80\]
        |
        v
[ui.exe  --  bandeja do sistema]
        |-- Historico de jobs (SQLite)
        |-- Abre PDFs gerados
        |-- Configura tipo de papel, colunas e trator
        |-- Monitora o servico portmonitor
```

O driver nao e um driver kernel-mode customizado. Usamos o driver genererico
"Generic / Text Only" ja presente no Windows e redirecionamos a porta de impressao
para um named pipe (`\\.\pipe\epson_fx80_emulator`). O portmonitor.exe escuta
esse pipe como servico Windows e processa cada job recebido.

---

## Estrutura do projeto

```
epson-fx80-emulator/
    go.mod                      modulo Go e dependencias
    build.ps1                   script de compilacao e deploy
    README.md                   este arquivo

    installer/
        install.ps1             instala/desinstala a impressora no Windows
        main.go                 versao Go do installer (compilavel como .exe)

    portmonitor/
        main.go                 entry point do servico
        service.go              interface com o SCM do Windows
        monitor.go              loop do named pipe
        processor.go            orquestra leitura, conversao e armazenamento

    pdfgen/
        pdfgen.go               converte texto em PDF com opcoes de papel
        options.go              le configuracoes de papel do registro

    storage/
        storage.go              banco SQLite com historico de jobs

    ui/
        main.go                 entry point da UI e system tray
        window.go               janela principal (historico + configuracoes)
        config.go               leitura e escrita de config no registro
        icon.go                 icone PNG embutido (impressora matricial)
```

---

## Ambiente de desenvolvimento

| Item | Detalhe |
|---|---|
| Linguagem | Go 1.22 |
| IDE | GoLand (JetBrains) |
| Sistema operacional | Windows 10 / 11 (x64) |
| Scripts | PowerShell 5+ |
| UI | Fyne v2.4.5 |
| PDF | go-pdf/fpdf v0.9.0 |
| Banco de dados | SQLite via mattn/go-sqlite3 v1.14.22 |
| Named pipe | Microsoft/go-winio v0.6.2 |
| Suporte de IA | Claude (Anthropic) |

> O pacote `go-sqlite3` requer CGo. E necessario ter GCC instalado para compilar.
> Recomendado: [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) para Windows.

---

## Compilacao

```powershell
# Na raiz do projeto
go mod tidy
.\build.ps1
```

O `build.ps1` compila os tres binarios em `dist\`:

| Binario | Funcao |
|---|---|
| `portmonitor.exe` | Servico Windows que intercepta os jobs |
| `installer.exe` | Instala/desinstala a impressora (alternativa ao .ps1) |
| `ui.exe` | Interface grafica na bandeja do sistema |

Para atualizar o servico em producao sem reiniciar manualmente:

```powershell
.\build.ps1 -CleanService
```

Isso para o servico, encerra o ui.exe, copia os novos binarios para `installer\`
e reinicia tudo automaticamente.

---

## Instalacao

```powershell
# PowerShell como Administrador
cd installer
.\install.ps1
```

O script realiza:

1. Verifica o driver `Generic / Text Only` no Windows
2. Cria o diretorio `Documentos\EpsonFX80` para os PDFs
3. Registra a porta `\\.\pipe\epson_fx80_emulator`
4. Adiciona a impressora `Epson FX-80 Emulator` no Windows
5. Grava configuracoes em `HKLM\SOFTWARE\EpsonFX80Emulator`
6. Instala e inicia o servico `EpsonFX80Monitor`
7. Cria atalho do `ui.exe` na inicializacao do Windows

Para desinstalar:

```powershell
.\install.ps1 -Uninstall
```

Para verificar o estado atual:

```powershell
.\install.ps1 -Status
```

---

## Configuracoes de papel

Acessiveis pelo icone na bandeja do sistema > Abrir gerenciador > Configuracoes:

| Opcao | Valores |
|---|---|
| Tipo de papel | Branco / Zebrado Verde / Zebrado Azul |
| Largura | 80 colunas / 132 colunas |
| Faixa de trator | Com furos laterais / Sem |
| Pasta de saida | Qualquer diretorio local |

As configuracoes sao salvas no registro do Windows e aplicadas a partir do proximo job.

---

## Diagnostico

```powershell
# Status do servico
sc query EpsonFX80Monitor

# Log do portmonitor
type C:\caminho\para\installer\portmonitor.log

# Log da UI
type C:\caminho\para\installer\ui.log

# Status completo
.\install.ps1 -Status
```

---

## Roadmap

- [x] Instalacao da impressora virtual no Windows
- [x] Interceptacao de jobs via named pipe
- [x] Geracao de PDF com fonte monoespaco
- [x] Papel zebrado (verde e azul) por linha
- [x] Faixa de trator com furos laterais
- [x] Suporte a 80 e 132 colunas
- [x] Historico de jobs em SQLite
- [x] Interface grafica com system tray
- [x] Configuracoes persistentes no registro
- [ ] Interpretacao completa do conjunto ESC/P da FX-80
- [ ] Modo condensado (17 cpi) e expandido (5 cpi)
- [ ] Negrito, italico e sublinhado via ESC
- [ ] Graficos de pinos (bit image graphics)
- [ ] Suporte a codepages internacionais
- [ ] Preview do PDF antes de salvar
- [ ] Notificacao na bandeja ao receber novo job
