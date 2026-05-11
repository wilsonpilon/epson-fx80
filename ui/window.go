// ui/window.go
// Janela principal: lista de jobs, botoes de acao, aba de configuracoes.

package main

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/epson-fx80-emulator/storage"
)

const maxJobs = 200

// newMainWindow cria e configura a janela principal.
func newMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("Epson FX-80 Emulator - Gerenciador de Impressao")
	w.Resize(fyne.NewSize(820, 520))
	w.SetCloseIntercept(func() {
		// Fechar a janela apenas a esconde, nao encerra o app
		w.Hide()
	})

	w.SetContent(buildContent(w, a))
	return w
}

// buildContent monta o conteudo da janela com abas.
func buildContent(w fyne.Window, a fyne.App) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Historico", theme.DocumentIcon(), buildJobsTab(w)),
		container.NewTabItemWithIcon("Configuracoes", theme.SettingsIcon(), buildSettingsTab(w, a)),
		container.NewTabItemWithIcon("Sobre", theme.InfoIcon(), buildAboutTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

// ── Aba Historico ─────────────────────────────────────────────────────────────

func buildJobsTab(w fyne.Window) fyne.CanvasObject {
	// Cabecalho da tabela
	headers := container.NewGridWithColumns(5,
		boldLabel("Data/Hora"),
		boldLabel("Arquivo"),
		boldLabel("Paginas"),
		boldLabel("Tamanho"),
		boldLabel("Acoes"),
	)

	list := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewGridWithColumns(5,
				widget.NewLabel(""),
				widget.NewLabel(""),
				widget.NewLabel(""),
				widget.NewLabel(""),
				container.NewHBox(
					widget.NewButtonWithIcon("", theme.DocumentIcon(), nil),
					widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
				),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	// Estado
	var jobs []storage.Job

	// var declarado antes para permitir auto-referencia dentro do closure (botao deletar)
	var refresh func()
	refresh = func() {
		db, err := openDB()
		if err != nil {
			log.Printf("[ui] Erro abrindo banco: %v", err)
			return
		}
		defer db.Close()
		jobs, err = db.ListJobs(maxJobs)
		if err != nil {
			log.Printf("[ui] Erro listando jobs: %v", err)
			return
		}

		list.Length = func() int { return len(jobs) }

		list.UpdateItem = func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(jobs) {
				return
			}
			j := jobs[id]
			row := obj.(*fyne.Container)
			cells := row.Objects

			cells[0].(*widget.Label).SetText(j.CreatedAt.Local().Format("02/01/2006 15:04:05"))
			cells[1].(*widget.Label).SetText(j.Name)
			cells[2].(*widget.Label).SetText(fmt.Sprintf("%d", j.Pages))
			cells[3].(*widget.Label).SetText(formatBytes(j.ByteSize))

			btns := cells[4].(*fyne.Container)
			// Botao abrir PDF
			openBtn := btns.Objects[0].(*widget.Button)
			openBtn.Icon = theme.DocumentIcon()
			openBtn.SetText("")
			openBtn.OnTapped = func() {
				if err := openFile(j.PDFPath); err != nil {
					dialog.ShowError(fmt.Errorf("Nao foi possivel abrir o PDF:\n%v", err), w)
				}
			}
			// Botao deletar
			delBtn := btns.Objects[1].(*widget.Button)
			delBtn.Icon = theme.DeleteIcon()
			delBtn.SetText("")
			capturedID := j.ID
			delBtn.OnTapped = func() {
				dialog.ShowConfirm(
					"Confirmar exclusao",
					fmt.Sprintf("Deseja remover o job '%s'?\n(O arquivo PDF nao sera deletado)", j.Name),
					func(ok bool) {
						if !ok {
							return
						}
						db2, err := openDB()
						if err != nil {
							return
						}
						defer db2.Close()
						db2.DeleteJob(capturedID)
						refresh()
					},
					w,
				)
			}
		}

		list.Refresh()
	}

	// Botoes da toolbar
	refreshBtn := widget.NewButtonWithIcon("Atualizar", theme.ViewRefreshIcon(), refresh)
	clearAllBtn := widget.NewButtonWithIcon("Limpar historico", theme.DeleteIcon(), func() {
		dialog.ShowConfirm(
			"Limpar historico",
			"Remover todos os registros do historico?\n(Os arquivos PDF nao serao deletados)",
			func(ok bool) {
				if !ok {
					return
				}
				db, err := openDB()
				if err != nil {
					return
				}
				defer db.Close()
				// Deleta todos os jobs um a um
				js, _ := db.ListJobs(9999)
				for _, j := range js {
					db.DeleteJob(j.ID)
				}
				refresh()
			},
			w,
		)
	})
	openFolderBtn := widget.NewButtonWithIcon("Abrir pasta", theme.FolderOpenIcon(), func() {
		openOutputDir()
	})

	countLabel := widget.NewLabel("")

	// Auto-refresh a cada 5 segundos
	go func() {
		for {
			refresh()
			db, _ := openDB()
			if db != nil {
				n, _ := db.CountJobs()
				db.Close()
				countLabel.SetText(fmt.Sprintf("%d job(s) no historico", n))
			}
			time.Sleep(5 * time.Second)
		}
	}()

	toolbar := container.NewHBox(refreshBtn, openFolderBtn, clearAllBtn, widget.NewSeparator(), countLabel)
	return container.NewBorder(
		container.NewVBox(toolbar, headers),
		nil, nil, nil,
		list,
	)
}

// ── Aba Configuracoes ─────────────────────────────────────────────────────────

func buildSettingsTab(w fyne.Window, a fyne.App) fyne.CanvasObject {
	cfg := loadConfig()

	// -- Pasta de saida -------------------------------------------------------
	outputDirEntry := widget.NewEntry()
	outputDirEntry.SetText(cfg.OutputDir)
	outputDirEntry.SetPlaceHolder("Pasta onde os PDFs serao salvos")

	browseDirBtn := widget.NewButtonWithIcon("Escolher pasta...", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outputDirEntry.SetText(uri.Path())
		}, w)
	})

	// -- Tipo de papel --------------------------------------------------------
	paperSelect := widget.NewSelect(
		[]string{"Branco", "Zebrado Verde", "Zebrado Azul"},
		nil,
	)
	paperSelect.SetSelected([]string{"Branco", "Zebrado Verde", "Zebrado Azul"}[cfg.PaperType])

	// -- Colunas --------------------------------------------------------------
	colsSelect := widget.NewSelect([]string{"80 colunas", "132 colunas"}, nil)
	if cfg.Columns == 132 {
		colsSelect.SetSelected("132 colunas")
	} else {
		colsSelect.SetSelected("80 colunas")
	}

	// -- Furos de trator ------------------------------------------------------
	tractorCheck := widget.NewCheck("Exibir faixa de trator com furos laterais", nil)
	tractorCheck.SetChecked(cfg.TractorFeed)

	// -- Preview descritivo ---------------------------------------------------
	previewLabel := widget.NewLabel("")
	updatePreview := func() {
		paper := paperSelect.Selected
		cols := colsSelect.Selected
		tractor := "sem trator"
		if tractorCheck.Checked {
			tractor = "com trator"
		}
		previewLabel.SetText("Proximo PDF: " + paper + " / " + cols + " / " + tractor)
	}
	paperSelect.OnChanged = func(_ string) { updatePreview() }
	colsSelect.OnChanged = func(_ string) { updatePreview() }
	tractorCheck.OnChanged = func(_ bool) { updatePreview() }
	updatePreview()

	// -- Servico --------------------------------------------------------------
	serviceStatus := widget.NewLabel("Verificando...")
	refreshStatus := func() {
		serviceStatus.SetText(serviceStatusText())
	}
	go refreshStatus()

	restartSvcBtn := widget.NewButtonWithIcon("Reiniciar servico", theme.ViewRefreshIcon(), func() {
		if err := restartService(); err != nil {
			dialog.ShowError(fmt.Errorf("Erro ao reiniciar servico:\n%v", err), w)
		} else {
			time.Sleep(time.Second)
			refreshStatus()
		}
	})

	// -- Salvar ---------------------------------------------------------------
	saveBtn := widget.NewButtonWithIcon("Salvar configuracoes", theme.DocumentSaveIcon(), func() {
		paperType := paperSelect.SelectedIndex()
		cols := 80
		if colsSelect.Selected == "132 colunas" {
			cols = 132
		}
		newCfg := Config{
			OutputDir:   outputDirEntry.Text,
			PaperType:   paperType,
			TractorFeed: tractorCheck.Checked,
			Columns:     cols,
		}
		if err := saveConfig(newCfg); err != nil {
			dialog.ShowError(fmt.Errorf("Erro ao salvar:\n%v", err), w)
			return
		}
		dialog.ShowInformation("Configuracoes salvas",
			"Configuracoes salvas com sucesso.\nO proximo job usara o novo papel.",
			w)
	})

	form := container.NewVBox(
		widget.NewSeparator(),
		widget.NewRichTextFromMarkdown("### Pasta de saida dos PDFs"),
		container.NewBorder(nil, nil, nil, browseDirBtn, outputDirEntry),
		widget.NewSeparator(),
		widget.NewRichTextFromMarkdown("### Configuracoes de papel"),
		container.NewGridWithColumns(2,
			widget.NewLabel("Tipo de papel:"),
			paperSelect,
			widget.NewLabel("Largura:"),
			colsSelect,
		),
		tractorCheck,
		previewLabel,
		widget.NewSeparator(),
		widget.NewRichTextFromMarkdown("### Servico Windows (EpsonFX80Monitor)"),
		container.NewHBox(widget.NewLabel("Status:"), serviceStatus),
		restartSvcBtn,
		widget.NewSeparator(),
		saveBtn,
	)

	return container.NewPadded(container.NewVScroll(form))
}

// ── Aba Sobre ─────────────────────────────────────────────────────────────────

func buildAboutTab() fyne.CanvasObject {
	title := widget.NewRichTextFromMarkdown("# Epson FX-80 Emulator")
	desc := widget.NewRichTextFromMarkdown(`
Impressora virtual para Windows que emula uma Epson FX-80.

Recebe jobs de impressao de qualquer aplicativo Windows e gera arquivos PDF automaticamente.

**Componentes:**
- **Impressora virtual**: aparece em "Impressoras e Scanners" do Windows
- **Port Monitor**: servico que intercepta os dados de impressao
- **PDF Generator**: converte o texto em PDF com fonte Arial
- **Gerenciador**: esta janela

**Pasta de PDFs:** os arquivos sao salvos em Documentos\EpsonFX80

**Versao:** 1.0.0
`)
	return container.NewPadded(container.NewVBox(title, desc))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func boldLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

func formatBytes(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	}
}
