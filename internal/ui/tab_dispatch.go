package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"torrentclick/internal/client"
	"torrentclick/internal/magnet"
)

// DispatchTab manages the primary magnet / torrent inspection and dispatch UI.
type DispatchTab struct {
	app            *MainApp
	magnetEntry    *widget.Entry
	nameLabel      *widget.Label
	hashLabel      *widget.Label
	trackersLabel  *widget.Label
	sourceLabel    *widget.Label
	profileSelect  *widget.Select
	sendBtn        *widget.Button
	statusLabel    *widget.Label
	detailsBtn     *widget.Button
	lastResult     *client.Result
	parsedMagnet   *magnet.Magnet
	profileMapping map[string]string // Profile Name -> Profile ID
}

// NewDispatchTab creates a new DispatchTab.
func NewDispatchTab(app *MainApp) *DispatchTab {
	return &DispatchTab{
		app:            app,
		profileMapping: make(map[string]string),
	}
}

// Build creates the Fyne container for the Dispatch tab.
func (d *DispatchTab) Build() fyne.CanvasObject {
	d.magnetEntry = widget.NewMultiLineEntry()
	d.magnetEntry.SetPlaceHolder("Paste magnet link, file path, or open a .torrent file here")
	d.magnetEntry.Wrapping = fyne.TextWrapBreak
	d.magnetEntry.OnChanged = func(val string) {
		d.parseAndUpdate(val)
	}

	pasteBtn := widget.NewButtonWithIcon("Paste from Clipboard", theme.ContentPasteIcon(), func() {
		cb := d.app.Window.Clipboard()
		if cb != nil {
			text := cb.Content()
			if text != "" {
				d.magnetEntry.SetText(text)
			}
		}
	})

	openFileBtn := widget.NewButtonWithIcon("Open .torrent...", theme.FolderOpenIcon(), func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			path := reader.URI().Path()
			if path != "" {
				d.SetTarget(path)
			}
		}, d.app.Window)
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".torrent"}))
		fileDialog.Show()
	})

	clearBtn := widget.NewButtonWithIcon("Clear", theme.ContentClearIcon(), func() {
		d.magnetEntry.SetText("")
		d.clearDetails()
	})

	sampleBtn := widget.NewButtonWithIcon("Insert Sample Magnet", theme.MediaPlayIcon(), func() {
		sample := "magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335380dc1f74c26&dn=Ubuntu+24.04+LTS+Desktop&tr=http%3A%2F%2Ftorrent.ubuntu.com%3A6969%2Fannounce"
		d.magnetEntry.SetText(sample)
	})

	topBtns := container.NewHBox(openFileBtn, pasteBtn, clearBtn, sampleBtn)

	// Torrent Details Card
	d.nameLabel = widget.NewLabelWithStyle("No torrent loaded", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	d.hashLabel = widget.NewLabel("Hash: -")
	d.trackersLabel = widget.NewLabel("Trackers: -")
	d.sourceLabel = widget.NewLabel("Source: -")

	detailsCard := widget.NewCard("Torrent Information", "", container.NewVBox(
		container.NewHBox(widget.NewLabelWithStyle("Name:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), d.nameLabel),
		d.hashLabel,
		d.trackersLabel,
		d.sourceLabel,
	))

	// Profile selection
	d.profileSelect = widget.NewSelect([]string{}, func(selectedName string) {
		if id, ok := d.profileMapping[selectedName]; ok {
			d.app.Config.ActiveProfileID = id
			_ = d.app.Config.Save()
			d.app.UpdateTrayMenu()
		}
	})
	d.RefreshProfiles()

	// Send Button
	d.sendBtn = widget.NewButtonWithIcon("Send to Torrent Client", theme.MailSendIcon(), func() {
		d.TriggerSend()
	})
	d.sendBtn.Importance = widget.HighImportance

	// Status & Details
	d.statusLabel = widget.NewLabelWithStyle("Ready", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	d.detailsBtn = widget.NewButtonWithIcon("View Response", theme.InfoIcon(), func() {
		d.showResponseDetails()
	})
	d.detailsBtn.Hide()

	statusBox := container.NewHBox(d.statusLabel, d.detailsBtn)

	controlsCard := widget.NewCard("Dispatch Destination", "", container.NewVBox(
		widget.NewLabel("Target Profile:"),
		d.profileSelect,
		container.NewVBox(d.sendBtn),
		statusBox,
	))

	content := container.NewVBox(
		widget.NewLabelWithStyle("Magnet & Torrent Dispatcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, topBtns, nil, nil, container.NewVScroll(d.magnetEntry)),
		detailsCard,
		controlsCard,
	)

	return container.NewVScroll(content)
}

func (d *DispatchTab) parseAndUpdate(raw string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		d.clearDetails()
		return
	}

	mag, err := magnet.Parse(trimmed)
	if err != nil {
		d.nameLabel.SetText("Invalid Magnet / Torrent")
		d.hashLabel.SetText(fmt.Sprintf("Error: %v", err))
		d.trackersLabel.SetText("")
		d.sourceLabel.SetText("")
		d.parsedMagnet = nil
		return
	}

	d.parsedMagnet = mag
	d.nameLabel.SetText(mag.DisplayName)
	d.hashLabel.SetText(fmt.Sprintf("Info Hash: %s (%s)", mag.InfoHash, mag.InfoHashType))
	if len(mag.Trackers) > 0 {
		d.trackersLabel.SetText(fmt.Sprintf("Trackers (%d): %s", len(mag.Trackers), strings.Join(mag.Trackers, ", ")))
	} else {
		d.trackersLabel.SetText("Trackers: None specified")
	}

	if mag.TorrentPath != "" {
		d.sourceLabel.SetText(fmt.Sprintf("Source: File (%s)", filepath.Base(mag.TorrentPath)))
	} else {
		d.sourceLabel.SetText("Source: Magnet Link")
	}
}

func (d *DispatchTab) clearDetails() {
	d.parsedMagnet = nil
	d.nameLabel.SetText("No torrent loaded")
	d.hashLabel.SetText("Hash: -")
	d.trackersLabel.SetText("Trackers: -")
	d.sourceLabel.SetText("Source: -")
	d.statusLabel.SetText("Ready")
	d.detailsBtn.Hide()
}

// RefreshProfiles reloads the profile dropdown list.
func (d *DispatchTab) RefreshProfiles() {
	options := make([]string, 0, len(d.app.Config.Profiles))
	d.profileMapping = make(map[string]string)

	var activeName string
	for _, p := range d.app.Config.Profiles {
		options = append(options, p.Name)
		d.profileMapping[p.Name] = p.ID
		if p.ID == d.app.Config.ActiveProfileID {
			activeName = p.Name
		}
	}

	d.profileSelect.Options = options
	if activeName != "" {
		d.profileSelect.SetSelected(activeName)
	} else if len(options) > 0 {
		d.profileSelect.SetSelected(options[0])
	}
}

// SetTarget populates the target input box with a magnet link or torrent file path.
func (d *DispatchTab) SetTarget(raw string) {
	d.magnetEntry.SetText(raw)
}

// SetMagnet populates the magnet input box (kept for compatibility).
func (d *DispatchTab) SetMagnet(raw string) {
	d.SetTarget(raw)
}

// TriggerSend sends the current parsed magnet or torrent to the active profile.
func (d *DispatchTab) TriggerSend() {
	if d.parsedMagnet == nil {
		raw := strings.TrimSpace(d.magnetEntry.Text)
		if raw == "" {
			d.statusLabel.SetText("Please enter a magnet link or .torrent file path first")
			return
		}
		var err error
		d.parsedMagnet, err = magnet.Parse(raw)
		if err != nil {
			d.statusLabel.SetText(fmt.Sprintf("Invalid input: %v", err))
			return
		}
	}

	activeProfile := d.app.Config.GetActiveProfile()
	if activeProfile == nil {
		d.statusLabel.SetText("No active profile configured")
		return
	}

	d.sendBtn.Disable()
	d.statusLabel.SetText(fmt.Sprintf("Sending to %s...", activeProfile.Name))
	d.detailsBtn.Hide()

	d.app.SendMagnet(d.parsedMagnet, activeProfile, func(res *client.Result, err error) {
		d.lastResult = res
		d.sendBtn.Enable()

		if res != nil && res.Success {
			d.statusLabel.SetText(fmt.Sprintf("Success! HTTP %d OK in %v", res.StatusCode, res.Duration.Round(10*time.Millisecond)))
			d.detailsBtn.Show()
		} else {
			errMsg := "Dispatch failed"
			if res != nil && res.ErrorMessage != "" {
				errMsg = res.ErrorMessage
			} else if err != nil {
				errMsg = err.Error()
			}
			d.statusLabel.SetText(fmt.Sprintf("Failed: %s", errMsg))
			if res != nil {
				d.detailsBtn.Show()
			}
		}
	})
}

func (d *DispatchTab) showResponseDetails() {
	if d.lastResult == nil {
		return
	}

	content := fmt.Sprintf(
		"Request URL:\n%s %s\n\nRequest Body:\n%s\n\nHTTP Status: %d %s\nDuration: %v\n\nResponse Body:\n%s",
		d.lastResult.RequestMethod,
		d.lastResult.RequestURL,
		d.lastResult.RequestBody,
		d.lastResult.StatusCode,
		d.lastResult.StatusText,
		d.lastResult.Duration,
		d.lastResult.ResponseBody,
	)

	entry := widget.NewMultiLineEntry()
	entry.SetText(content)
	entry.Wrapping = fyne.TextWrapBreak

	dModal := dialog.NewCustom(
		"API Call Details",
		"Close",
		container.NewScroll(container.NewGridWrap(fyne.NewSize(600, 350), entry)),
		d.app.Window,
	)
	dModal.Resize(fyne.NewSize(650, 420))
	dModal.Show()
}
