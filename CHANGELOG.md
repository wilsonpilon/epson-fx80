# Changelog

Todas as alteracoes relevantes do projeto Epson FX-80 Emulator serao documentadas aqui.

Formato baseado em [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versionamento seguindo [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.3] - 2026-05-11

### Adicionado
- Configuracoes de papel acessiveis pela UI e pelo registro do Windows
  - Tipo de papel: Branco, Zebrado Verde, Zebrado Azul
  - Largura em colunas: 80 ou 132
  - Faixa de trator com furos laterais (espacamento real de 12.7mm)
- Preview descritivo em tempo real na aba Configuracoes da UI
- Icone da impressora matricial no system tray em formato PNG embutido
  (corrige quadrado cinza que aparecia anteriormente com SVG)
- Opcao `-CleanService` no `build.ps1` para parar servico, copiar binarios e reiniciar em um comando
- Atalho do `ui.exe` criado automaticamente na inicializacao do Windows pelo `install.ps1`
- Deteccao de reinstalacao no `install.ps1` com aviso ao usuario e confirmacao antes de prosseguir
- `MANUAL.md` com instrucoes completas de compilacao, operacao, configuracao e referencia ESC/P
- `README.md` com descricao do projeto, arquitetura, ambiente de desenvolvimento e roadmap

### Alterado
- `pdfgen`: reescrito para suportar opcoes de papel; fonte trocada para Courier monoespaco
  com tamanho calculado automaticamente para caber o numero de colunas no papel A4
- `build.ps1`: separado em modo de compilacao e modo `-CleanService`; deploy automatico
  para a pasta `installer\` inclui parada e reinicio do `ui.exe`
- `install.ps1`: porta agora configurada diretamente como `\\.\pipe\epson_fx80_emulator`
  (corrige problema em que o spooler nao redirecionava para o named pipe)
- `config.go`: novos campos `PaperType`, `TractorFeed` e `Columns` adicionados ao `Config`
- Aba Configuracoes da UI expandida com selecao de papel, colunas e trator

### Corrigido
- Icone do system tray nao renderizava (Fyne no Windows nao suporta SVG como icone de bandeja)
- `[OK]` falso na etapa de gravacao do registro quando `New-Item` falhava silenciosamente
- Binarios bloqueados ao tentar copiar durante o `-CleanService` sem parar o `ui.exe` antes
- Encoding dos scripts `.ps1` corrigido para ASCII puro (eliminados caracteres UTF-8
  como travessoes e checkmarks que causavam erros de parse no PowerShell)

---

## [0.1.2] - 2026-05-11

### Adicionado
- Interface grafica `ui.exe` com icone na bandeja do sistema (system tray)
  - Aba Historico: lista jobs com data, nome, paginas, tamanho e botoes de acao
  - Aba Configuracoes: pasta de saida, status do servico, botao reiniciar
  - Aba Sobre: informacoes do projeto
  - Auto-refresh da lista a cada 5 segundos
- Menu da bandeja: Abrir gerenciador / Abrir pasta de PDFs / Encerrar
- Fechar a janela apenas a esconde; app permanece na bandeja
- `ui.exe` compilado com `-H windowsgui` para nao exibir janela de terminal

### Alterado
- `build.ps1` atualizado para compilar o terceiro binario `ui.exe`

---

## [0.1.1] - 2026-05-10

### Adicionado
- Impressora virtual `Epson FX-80 Emulator` registrada no Windows
- Driver baseado em `Generic / Text Only` (ja incluso no Windows, sem driver kernel)
- Porta de impressao redirecionada para named pipe `\\.\pipe\epson_fx80_emulator`
- Servico Windows `EpsonFX80Monitor` que escuta o named pipe via `go-winio`
- Conversao de jobs de impressao em PDF com fonte Arial via `go-pdf/fpdf`
- Limpeza de sequencias de escape ESC/P da Epson FX-80 antes da geracao do PDF
- Banco SQLite para historico de jobs (data, nome, caminho, paginas, tamanho)
- `install.ps1` para instalacao e desinstalacao da impressora no Windows
- `build.ps1` para compilacao dos binarios `portmonitor.exe` e `installer.exe`
- Modo `-debug` no `portmonitor.exe` para rodar no terminal sem instalar como servico
- Log gravado em `portmonitor.log` ao lado do executavel
- Configuracoes gravadas em `HKLM\SOFTWARE\EpsonFX80Emulator`

### Estrutura inicial do projeto
- `portmonitor/` -- servico Windows (main, service, monitor, processor)
- `pdfgen/` -- gerador de PDF
- `storage/` -- banco SQLite
- `installer/` -- scripts e binario de instalacao

---

## Versoes planejadas

### [0.2.0] -- em planejamento
- Interpretacao real de comandos ESC/P (negrito, italico, condensado, expandido)
- Notificacao na bandeja ao receber novo job
- Preview do PDF antes de salvar

### [0.3.0] -- em planejamento
- Graficos de pinos (bit image graphics ESC K / L / Y / Z)
- Suporte a codepages internacionais (ESC R)
- Emulacao de velocidade de impressao (som opcional)

### [1.0.0] -- objetivo final
- Emulacao 100% do conjunto de comandos ESC/P da Epson FX-80
- Documentacao completa do protocolo ESC/P no MANUAL.md
