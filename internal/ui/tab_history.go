package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"torrentclick/internal/history"
)

// HistoryTab manages the history log display and retry actions.
type HistoryTab struct {
	app      *MainApp
	list     *widget.List
	entries  []history.Entry
	emptyLbl *widget.Label
}

// NewHistoryTab creates a new HistoryTab.
func NewHistoryTab(app *MainApp) *HistoryTab {
	return &HistoryTab{
		app:     app,
		entries: make([]history.Entry, 0),
	}
}

// Build creates the Fyne container for the History tab.
func (h *HistoryTab) Build() fyne.CanvasObject {
	h.emptyLbl = widget.NewLabelWithStyle("No torrents dispatched yet.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	h.list = widget.NewList(
		func() int {
			return len(h.entries)
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.ConfirmIcon())
			title := widget.NewLabelWithStyle("Title", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			subtitle := widget.NewLabel("Details")
			resendBtn := widget.NewButtonWithIcon("Resend", theme.ViewRefreshIcon(), nil)

			return container.NewBorder(nil, nil, icon, resendBtn, container.NewVBox(title, subtitle))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(h.entries) {
				return
			}
			entry := h.entries[id]
			border := obj.(*fyne.Container)
			icon := border.Objects[1].(*widget.Icon)
			resendBtn := border.Objects[2].(*widget.Button)
			vbox := border.Objects[0].(*fyne.Container)
			title := vbox.Objects[0].(*widget.Label)
			subtitle := vbox.Objects[1].(*widget.Label)

			if entry.Success {
				icon.SetResource(theme.ConfirmIcon())
			} else {
				icon.SetResource(theme.ErrorIcon())
			}

			title.SetText(entry.DisplayName)

			statusText := fmt.Sprintf("HTTP %d", entry.StatusCode)
			if !entry.Success {
				statusText = fmt.Sprintf("Failed: %s", entry.ErrorMessage)
			}
			subtitle.SetText(fmt.Sprintf("%s • %s • %s",
				entry.TargetProfileName,
				statusText,
				entry.Timestamp.Format("2006-01-02 15:04:05"),
			))

			resendBtn.OnTapped = func() {
				h.app.DispatchTab.SetMagnet(entry.MagnetURI)
				h.app.Tabs.SelectIndex(0) // Switch to Dispatch
			}
		},
	)

	clearBtn := widget.NewButtonWithIcon("Clear History", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Clear History", "Are you sure you want to clear dispatch history?", func(confirm bool) {
			if confirm {
				h.app.History.Clear()
				h.Refresh()
			}
		}, h.app.Window)
	})

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		h.Refresh()
	})

	topBar := container.NewHBox(
		widget.NewLabelWithStyle("Recent Torrent Dispatches", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		refreshBtn,
		clearBtn,
	)

	h.Refresh()

	return container.NewBorder(topBar, nil, nil, nil, h.list)
}

// Refresh reloads history entries into the list.
func (h *HistoryTab) Refresh() {
	h.entries = h.app.History.List()
	if h.list != nil {
		h.list.Refresh()
	}
}
