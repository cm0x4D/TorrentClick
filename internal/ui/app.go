package ui

import (
	"context"
	"fmt"
	"net/url"
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

	mApp.ApplyTheme()

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

// HandleIncomingMagnet processes a magnet link received from macOS or CLI.
func (m *MainApp) HandleIncomingMagnet(magnetURI string) {
	m.Window.RequestFocus()
	m.DispatchTab.SetMagnet(magnetURI)

	// If AutoSend is enabled in user settings, immediately trigger dispatch
	if m.Config.AutoSend {
		m.DispatchTab.TriggerSend()
	} else {
		// Switch to Dispatch tab so user can review and click Send
		m.Tabs.SelectIndex(0)
	}
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

			// Desktop Notification
			if m.Config.NotificationsEnabled {
				if entry.Success {
					m.FyneApp.SendNotification(fyne.NewNotification(
						"Torrent Sent Successfully",
						fmt.Sprintf("%s -> %s (%d OK)", mag.DisplayName, profile.Name, entry.StatusCode),
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
