# Changelog

Todas as alteracoes relevantes do projeto Epson FX-80 Emulator serao documentadas aqui.

Formato baseado em [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versionamento seguindo [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.5] - 2026-05-11

### Adicionado
- Pacote `fontmgr`: gerenciamento completo de fontes TTF por modo de impressao
  - 12 modos suportados: Regular, Bold, Italic, Condensed, Expanded e todas as combinacoes
  - Escaneamento de subpastas em `installer\fonts\ttf\` (regular, bold, italic, etc.)
  - Mapeamento modo->arquivo salvo em `HKLM\SOFTWARE\EpsonFX80Emulator\Fonts`
- Selecao de fonte TTF por modo na aba Configuracoes da UI
  - Um `Select` por modo listando os `.ttf` disponiveis na subpasta correspondente
  - Fallback automatico para Courier quando nenhum TTF estiver configurado
- `pdfgen/testpage.go`: gerador dedicado da pagina de teste com controle direto de fonte
  - Cada modo registrado com familia unica no fpdf (`TTF_Regular`, `TTF_Bold`, etc.)
  - Garante distincao visual real entre os modos no PDF gerado
- Pagina de teste acessivel pela toolbar da aba Historico ("Pagina de teste")
  - Regua de 80 colunas com digitos superior/inferior e marcadores a cada 5/10 col
  - Caracteres ASCII imprimiveis (32-126)
  - Bloco por modo: cabecalho identificador + frase de teste + digitos/simbolos
  - Dialogo pos-geracao perguntando se deseja abrir o PDF imediatamente
- Versao e build stamp visiveis no rodape da janela principal (todas as abas)
  - Formato: `Epson FX-80 Emulator  0.1.5 - 0x6A01DF28`
  - Build stamp = Unix timestamp em hexadecimal, injetado via `-ldflags`
- `ui/version.go`: variaveis `Version` e `BuildStamp` injetaveis via ldflags
  - `FullVersion()` retorna `"0.1.5 - build 0x6A01DF28"`
  - `WindowTitle()` retorna o titulo completo para a janela
- `REFERENCE.md`: referencia rapida de todos os comandos operacionais

### Alterado
- `portmonitor/main.go`: pre-carga das fontes TTF ao iniciar o servico via `preloadFonts()`
  - Variavel global `fontManager` disponivel para todos os jobs
  - Log detalhado de cada fonte carregada por modo
- `portmonitor/processor.go`: injeta `fontManager.Map` nas opcoes do pdfgen
- `pdfgen/pdfgen.go`: campo `VersionLine` adicionado a `Options`; `registerFonts()` melhorado
- `build.ps1`: calcula e injeta `Version` e `BuildStamp` no `ui.exe` via ldflags
  - `$BuildHex = "0x{0:X}" -f [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()`
  - Corrigido escaping de ldflags: usa `& go build` e variavel `$UILDFlags` separada
- `README.md`: estrutura do projeto atualizada com novos pacotes e arquivos
- Versao global atualizada para 0.1.5

### Corrigido
- Fontes Bold/Italic/Condensed apareciam identicas a Regular na pagina de teste
  (cada modo agora usa familia fpdf separada em vez de estilo da mesma familia)
- Erro de escape `\t` e `\` em string Go na mensagem de pasta de fontes nao encontrada
- Erro de ldflags: `invalid value... parameter may not start with quote character`
  (corrigido com operador `&` e variavel intermediaria `$UILDFlags`)

---

## [0.1.3] - 2026-05-11

### Adicionado
- Configuracoes de papel acessiveis pela UI e pelo registro do Windows
  - Tipo de papel: Branco, Zebrado Verde, Zebrado Azul
  - Largura em colunas: 80 ou 132
  - Faixa de trator com furos laterais (espacamento real de 12.7mm)
- Preview descritivo em tempo real na aba Configuracoes
- Icone da impressora matricial no system tray em formato PNG embutido
- Opcao `-CleanService` no `build.ps1`
- Atalho do `ui.exe` na inicializacao do Windows
- Deteccao de reinstalacao no `install.ps1` com confirmacao

### Alterado
- `pdfgen`: fonte trocada para Courier; tamanho calculado automaticamente por colunas
- Porta configurada diretamente como `\\.\pipe\epson_fx80_emulator`

### Corrigido
- Icone cinza no system tray (SVG trocado por PNG base64)
- `[OK]` falso na gravacao do registro
- Encoding ASCII dos scripts .ps1

---

## [0.1.2] - 2026-05-11

### Adicionado
- Interface grafica `ui.exe` com system tray
- Aba Historico: lista jobs com acoes de abrir e deletar
- Aba Configuracoes: pasta de saida e status do servico
- Auto-refresh da lista a cada 5 segundos

---

## [0.1.1] - 2026-05-10

### Adicionado
- Impressora virtual `Epson FX-80 Emulator` registrada no Windows
- Driver `Generic / Text Only` com porta named pipe
- Servico Windows `EpsonFX80Monitor`
- Conversao de jobs em PDF com fonte Arial
- Limpeza de sequencias ESC/P
- Banco SQLite para historico de jobs
- `install.ps1` e `build.ps1`
- Modo `-debug` no portmonitor

---

## Versoes planejadas

### [0.2.0]
- Interpretacao real de comandos ESC/P (negrito, italico, condensado, expandido)
- Notificacao na bandeja ao receber novo job
- Preview do PDF antes de salvar

### [0.3.0]
- Graficos de pinos (bit image ESC K/L/Y/Z)
- Suporte a codepages internacionais (ESC R)

### [1.0.0]
- Emulacao 100% do ESC/P da Epson FX-80
