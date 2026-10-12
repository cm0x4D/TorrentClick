package ui

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"torrentclick/internal/client"
	"torrentclick/internal/config"
	"torrentclick/internal/history"
	"torrentclick/internal/magnet"
	"torrentclick/internal/sysdark"
)

// MainApp wraps the Fyne GUI application state.
type MainApp struct {
	FyneApp      fyne.App
	Window       fyne.Window
	Config       *config.Config
	Client       *client.Client
	History      *history.Manager
	DispatchTab  *DispatchTab
	ProfilesTab  *ProfilesTab
	HistoryTab   *HistoryTab
	SettingsTab  *SettingsTab
	Tabs         *container.AppTabs
}

// NewMainApp constructs and initializes the application UI.
func NewMainApp(cfg *config.Config, hist *history.Manager) *MainApp {
	a := app.NewWithID("com.torrentclick.app")
	w := a.NewWindow("TorrentClick")

	mApp := &MainApp{
		FyneApp: a,
		Window:  w,
		Config:  cfg,
		Client:  client.NewClient(),
		History: hist,
	}

	mApp.DispatchTab = NewDispatchTab(mApp)
	mApp.ProfilesTab = NewProfilesTab(mApp)
	mApp.HistoryTab = NewHistoryTab(mApp)
	mApp.SettingsTab = NewSettingsTab(mApp)

	mApp.Tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Dispatch", theme.MailSendIcon(), mApp.DispatchTab.Build()),
		container.NewTabItemWithIcon("Profiles", theme.SettingsIcon(), mApp.ProfilesTab.Build()),
		container.NewTabItemWithIcon("History", theme.HistoryIcon(), mApp.HistoryTab.Build()),
		container.NewTabItemWithIcon("Settings", theme.ComputerIcon(), mApp.SettingsTab.Build()),
	)

	w.SetContent(mApp.Tabs)
	w.Resize(fyne.NewSize(750, 600))
	w.CenterOnScreen()

	// Intercept window close: hide to system tray instead of exiting
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	mApp.ApplyTheme()
	mApp.SetupSystemTray()

	return mApp
}

// ApplyTheme synchronizes Fyne theme with macOS appearance and preferences.
func (m *MainApp) ApplyTheme() {
	switch m.Config.ThemeMode {
	case "dark":
		m.FyneApp.Settings().SetTheme(theme.DarkTheme())
	case "light":
		m.FyneApp.Settings().SetTheme(theme.LightTheme())
	default: // "auto"
		if sysdark.IsDarkMode() {
			m.FyneApp.Settings().SetTheme(theme.DarkTheme())
		} else {
			m.FyneApp.Settings().SetTheme(theme.LightTheme())
		}
	}
}

// HandleIncomingTarget processes a magnet link or .torrent file received from macOS or CLI.
// When the app is already running, the window is NOT shown or focused;
// the link or file is dispatched silently in the background with a desktop notification.
func (m *MainApp) HandleIncomingTarget(target string) {
	m.DispatchTab.SetTarget(target)
	// Silently trigger background dispatch without showing or focusing window
	m.DispatchTab.TriggerSendTarget(target)
}

// HandleIncomingMagnet is a convenience wrapper for HandleIncomingTarget.
func (m *MainApp) HandleIncomingMagnet(magnetURI string) {
	m.HandleIncomingTarget(magnetURI)
}

// SendMagnet dispatches the magnet link to the chosen profile.
func (m *MainApp) SendMagnet(mag *magnet.Magnet, profile *config.Profile, onComplete func(res *client.Result, err error)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		res, err := m.Client.Execute(ctx, profile, mag)

		// Check for 2FA OTP requirement
		if client.IsOTPRequired(err) {
			fyne.Do(func() {
				m.PromptOTP(
					"Two-Factor Authentication (2FA)",
					"Your Synology DSM account requires an OTP code. Enter the 6-digit verification code from your authenticator app:",
					func(code string) {
						if profile.CustomParams == nil {
							profile.CustomParams = make(map[string]string)
						}
						profile.CustomParams["otp_code"] = code
						m.SendMagnet(mag, profile, func(retryRes *client.Result, retryErr error) {
							if retryRes != nil && retryRes.Success {
								_ = m.Config.Save()
								m.ProfilesTab.loadProfileIntoForm(*profile)
							}
							if onComplete != nil {
								onComplete(retryRes, retryErr)
							}
						})
					},
				)
			})
			return
		}

		// Record history
		entry := history.Entry{
			ID:                fmt.Sprintf("%d", time.Now().UnixNano()),
			Timestamp:         time.Now(),
			MagnetURI:         mag.Raw,
			DisplayName:       mag.DisplayName,
			InfoHash:          mag.InfoHash,
			TargetProfileName: profile.Name,
			Success:           res != nil && res.Success,
			Duration:          0,
		}
		if res != nil {
			entry.StatusCode = res.StatusCode
			entry.Duration = res.Duration
			entry.ErrorMessage = res.ErrorMessage
		} else if err != nil {
			entry.ErrorMessage = err.Error()
		}

		m.History.Add(entry)

		fyne.Do(func() {
			m.HistoryTab.Refresh()

			// Optionally delete .torrent file after successful dispatch
			deletedFile := false
			if entry.Success && m.Config.DeleteTorrentAfterDispatch && mag.TorrentPath != "" {
				if _, statErr := os.Stat(mag.TorrentPath); statErr == nil {
					if removeErr := os.Remove(mag.TorrentPath); removeErr != nil {
						log.Printf("Warning: failed to delete torrent file %s: %v", mag.TorrentPath, removeErr)
					} else {
						deletedFile = true
						log.Printf("Deleted torrent file after successful dispatch: %s", mag.TorrentPath)
					}
				}
			}

			// Desktop Notification
			if m.Config.NotificationsEnabled {
				if entry.Success {
					msg := fmt.Sprintf("%s -> %s (%d OK)", mag.DisplayName, profile.Name, entry.StatusCode)
					if deletedFile {
						msg += " (file deleted)"
					}
					m.FyneApp.SendNotification(fyne.NewNotification(
						"Torrent Sent Successfully",
						msg,
					))
				} else {
					m.FyneApp.SendNotification(fyne.NewNotification(
						"Torrent Dispatch Failed",
						fmt.Sprintf("%s: %s", mag.DisplayName, entry.ErrorMessage),
					))
				}
			}

			if onComplete != nil {
				onComplete(res, err)
			}

			// Hide window if configured and successful
			if entry.Success && m.Config.HideWindowAfterSend {
				m.Window.Hide()
			}
		})
	}()
}

// PromptOTP displays a dialog requesting the 6-digit 2FA OTP code.
func (m *MainApp) PromptOTP(title, message string, onSubmitted func(code string)) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("6-digit code (e.g. 123456)")

	formItem := widget.NewFormItem("OTP Code", entry)
	d := dialog.NewForm(title, "Verify", "Cancel", []*widget.FormItem{formItem}, func(confirmed bool) {
		if confirmed {
			code := strings.TrimSpace(entry.Text)
			if code != "" && onSubmitted != nil {
				onSubmitted(code)
			}
		}
	}, m.Window)
	d.Resize(fyne.NewSize(380, 180))
	d.Show()
}

// ShowInfoDialog displays a standard information modal.
func (m *MainApp) ShowInfoDialog(title, message string) {
	dialog.ShowInformation(title, message, m.Window)
}

// ShowErrorDialog displays an error modal.
func (m *MainApp) ShowErrorDialog(title string, err error) {
	dialog.ShowError(fmt.Errorf("%s: %w", title, err), m.Window)
}

// OpenURLInBrowser opens a URL using the default macOS browser.
func OpenURLInBrowser(targetURL string) {
	_, err := url.ParseRequestURI(targetURL)
	if err == nil {
		_ = exec.Command("open", targetURL).Start()
	}
}
