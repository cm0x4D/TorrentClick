package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SettingsTab manages application preferences and macOS LaunchServices registration.
type SettingsTab struct {
	app          *MainApp
	autoSend     *widget.Check
	notify       *widget.Check
	hideAfter    *widget.Check
	statusBanner *widget.Label
}

// NewSettingsTab creates a new SettingsTab.
func NewSettingsTab(app *MainApp) *SettingsTab {
	return &SettingsTab{app: app}
}

// Build creates the Fyne container for the Settings tab.
func (s *SettingsTab) Build() fyne.CanvasObject {
	s.autoSend = widget.NewCheck("Auto-send immediately when magnet link is received", func(val bool) {
		s.app.Config.AutoSend = val
	})
	s.autoSend.SetChecked(s.app.Config.AutoSend)

	s.notify = widget.NewCheck("Show macOS desktop notifications on dispatch outcome", func(val bool) {
		s.app.Config.NotificationsEnabled = val
	})
	s.notify.SetChecked(s.app.Config.NotificationsEnabled)

	s.hideAfter = widget.NewCheck("Hide application window after successful dispatch", func(val bool) {
		s.app.Config.HideWindowAfterSend = val
	})
	s.hideAfter.SetChecked(s.app.Config.HideWindowAfterSend)

	themeSelect := widget.NewSelect([]string{
		"Match macOS System (Auto)",
		"Dark Mode",
		"Light Mode",
	}, func(val string) {
		switch val {
		case "Dark Mode":
			s.app.Config.ThemeMode = "dark"
		case "Light Mode":
			s.app.Config.ThemeMode = "light"
		default:
			s.app.Config.ThemeMode = "auto"
		}
		s.app.ApplyTheme()
		_ = s.app.Config.Save()
	})

	switch s.app.Config.ThemeMode {
	case "dark":
		themeSelect.SetSelected("Dark Mode")
	case "light":
		themeSelect.SetSelected("Light Mode")
	default:
		themeSelect.SetSelected("Match macOS System (Auto)")
	}

	saveBtn := widget.NewButtonWithIcon("Save Preferences", theme.DocumentSaveIcon(), func() {
		if err := s.app.Config.Save(); err != nil {
			dialog.ShowError(err, s.app.Window)
		} else {
			dialog.ShowInformation("Saved", "Preferences saved successfully.", s.app.Window)
		}
	})
	saveBtn.Importance = widget.HighImportance

	prefCard := widget.NewCard("Appearance & Behavior", "", container.NewVBox(
		widget.NewLabel("Desktop Appearance Theme:"),
		themeSelect,
		widget.NewSeparator(),
		s.autoSend,
		s.notify,
		s.hideAfter,
		container.NewHBox(saveBtn),
	))

	// macOS Integration Section
	s.statusBanner = widget.NewLabel("Click below to register this app as the macOS handler for magnet: links.")

	registerBtn := widget.NewButtonWithIcon("Register with macOS LaunchServices", theme.StorageIcon(), func() {
		s.registerLaunchServices()
	})
	registerBtn.Importance = widget.MediumImportance

	testSchemeBtn := widget.NewButtonWithIcon("Trigger Test magnet: Link via macOS", theme.MediaPlayIcon(), func() {
		cmd := exec.Command("open", "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=macOS+Integration+Test")
		if err := cmd.Run(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to open magnet URL: %w", err), s.app.Window)
		} else {
			s.statusBanner.SetText("Triggered test link via 'open magnet:...'")
		}
	})

	macCard := widget.NewCard("macOS URL Handler Integration", "", container.NewVBox(
		widget.NewLabel(
			"TorrentClick registers for the 'magnet:' URL scheme in its Info.plist.\n"+
				"When packaged into TorrentClick.app, macOS will route magnet link clicks directly to this tool.\n"+
				"You can also use 'task package' or the button below to register with macOS LaunchServices.",
		),
		s.statusBanner,
		container.NewHBox(registerBtn, testSchemeBtn),
	))

	content := container.NewVBox(
		widget.NewLabelWithStyle("Preferences & System Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		prefCard,
		macCard,
	)

	return container.NewVScroll(content)
}

func (s *SettingsTab) registerLaunchServices() {
	// Locate executable bundle
	exePath, err := os.Executable()
	if err != nil {
		dialog.ShowError(err, s.app.Window)
		return
	}

	// If running inside .app bundle, find the bundle root
	bundlePath := exePath
	if idx := filepath.Clean(exePath); true {
		for dir := idx; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			if filepath.Ext(dir) == ".app" {
				bundlePath = dir
				break
			}
		}
	}

	lsregister := "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	cmd := exec.Command(lsregister, "-f", bundlePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.statusBanner.SetText(fmt.Sprintf("Registration error: %v (%s)", err, string(out)))
		dialog.ShowError(fmt.Errorf("lsregister failed: %w", err), s.app.Window)
	} else {
		msg := fmt.Sprintf("Registered '%s' with macOS LaunchServices successfully!", filepath.Base(bundlePath))
		s.statusBanner.SetText(msg)
		dialog.ShowInformation("Registered", msg, s.app.Window)
	}
}
