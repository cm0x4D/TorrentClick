package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"torrentclick/internal/client"
	"torrentclick/internal/config"
	"torrentclick/internal/magnet"
)

// ProfilesTab manages configuring and editing client REST templates.
type ProfilesTab struct {
	app             *MainApp
	profileSelect   *widget.Select
	currentID       string

	nameEntry       *widget.Entry
	descEntry       *widget.Entry
	urlEntry        *widget.Entry
	endpointEntry   *widget.Entry
	methodSelect    *widget.Select
	ctEntry         *widget.Entry
	authSelect      *widget.Select
	userEntry       *widget.Entry
	passEntry       *widget.Entry
	tokenEntry      *widget.Entry
	hdrKeyEntry     *widget.Entry
	hdrValEntry     *widget.Entry
	bodyEntry       *widget.Entry
	totpSecretEntry *widget.Entry
	deviceIDEntry   *widget.Entry

	saveBtn         *widget.Button
	testBtn         *widget.Button
	deleteBtn       *widget.Button
	makeActiveBtn   *widget.Button
}

// NewProfilesTab initializes the profiles editor tab.
func NewProfilesTab(app *MainApp) *ProfilesTab {
	return &ProfilesTab{
		app: app,
	}
}

// Build creates the Fyne container for the Profiles tab.
func (p *ProfilesTab) Build() fyne.CanvasObject {
	p.nameEntry = widget.NewEntry()
	p.descEntry = widget.NewEntry()
	p.urlEntry = widget.NewEntry()
	p.endpointEntry = widget.NewEntry()
	p.ctEntry = widget.NewEntry()
	p.userEntry = widget.NewEntry()
	p.passEntry = widget.NewPasswordEntry()
	p.tokenEntry = widget.NewPasswordEntry()
	p.hdrKeyEntry = widget.NewEntry()
	p.hdrValEntry = widget.NewEntry()

	p.totpSecretEntry = widget.NewPasswordEntry()
	p.totpSecretEntry.SetPlaceHolder("Optional: 32-char Base32 TOTP secret for hands-free 2FA")

	p.deviceIDEntry = widget.NewEntry()
	p.deviceIDEntry.SetPlaceHolder("Auto-filled once 2FA OTP is verified on Synology")

	p.methodSelect = widget.NewSelect([]string{"POST", "GET", "PUT", "PATCH"}, nil)
	p.authSelect = widget.NewSelect([]string{
		string(config.AuthNone),
		string(config.AuthBasic),
		string(config.AuthBearer),
		string(config.AuthHeader),
		string(config.AuthSynologyLogin),
		string(config.AuthQBittorrentLogin),
		string(config.AuthTransmissionSession),
	}, func(val string) {
		p.updateAuthFieldVisibility(config.AuthType(val))
	})

	p.bodyEntry = widget.NewMultiLineEntry()
	p.bodyEntry.Wrapping = fyne.TextWrapBreak
	p.bodyEntry.SetPlaceHolder("Go template for request body, e.g. urls={{.Magnet}} or {\"uri\": \"{{.Magnet}}\"}")

	// Profile selector header
	p.profileSelect = widget.NewSelect([]string{}, func(selectedName string) {
		for _, prof := range p.app.Config.Profiles {
			if prof.Name == selectedName {
				p.loadProfileIntoForm(prof)
				break
			}
		}
	})

	newPresetSelect := widget.NewSelect([]string{
		"Preset: Synology DSM",
		"Preset: qBittorrent",
		"Preset: Transmission",
		"Preset: Deluge",
		"Preset: Generic Webhook",
	}, func(selected string) {
		p.loadPreset(selected)
	})
	newPresetSelect.PlaceHolder = "+ Add New from Preset..."

	headerBox := container.NewHBox(
		widget.NewLabelWithStyle("Select Profile:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.profileSelect,
		newPresetSelect,
	)

	// Action Buttons
	p.saveBtn = widget.NewButtonWithIcon("Save Profile", theme.DocumentSaveIcon(), func() {
		p.saveCurrentProfile()
	})
	p.saveBtn.Importance = widget.HighImportance

	p.makeActiveBtn = widget.NewButtonWithIcon("Set as Active", theme.ConfirmIcon(), func() {
		p.app.Config.ActiveProfileID = p.currentID
		_ = p.app.Config.Save()
		p.app.DispatchTab.RefreshProfiles()
		p.app.UpdateTrayMenu()
		dialog.ShowInformation("Profile Activated", fmt.Sprintf("'%s' is now the active client profile.", p.nameEntry.Text), p.app.Window)
	})

	p.testBtn = widget.NewButtonWithIcon("Test Connection", theme.MediaPlayIcon(), func() {
		p.testCurrentProfile()
	})

	p.deleteBtn = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		p.deleteCurrentProfile()
	})
	p.deleteBtn.Importance = widget.DangerImportance

	btnBar := container.NewHBox(p.saveBtn, p.makeActiveBtn, p.testBtn, p.deleteBtn)

	// Templating helper accordion / tips
	tmplTips := widget.NewLabel(
		"Available Go Template Variables:\n" +
			" • {{.Magnet}} - Full raw magnet link\n" +
			" • {{.MagnetEscaped}} - URL-encoded magnet link\n" +
			" • {{.InfoHash}} - Torrent BTIH InfoHash\n" +
			" • {{.DisplayName}} / {{.Name}} - Torrent title\n" +
			" • {{.SessionID}} - Authenticated session/token\n" +
			" • {{index .Config \"param\"}} - Custom parameter\n" +
			"Functions: urlquery, lower, upper, trim, json, default",
	)

	form := container.NewVBox(
		widget.NewLabelWithStyle("Profile Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			widget.NewLabel("Profile Name:"), p.nameEntry,
			widget.NewLabel("Description:"), p.descEntry,
			widget.NewLabel("Base Server URL:"), p.urlEntry,
			widget.NewLabel("Endpoint Path:"), p.endpointEntry,
			widget.NewLabel("HTTP Method:"), p.methodSelect,
			widget.NewLabel("Content-Type:"), p.ctEntry,
			widget.NewLabel("Authentication:"), p.authSelect,
			widget.NewLabel("Username:"), p.userEntry,
			widget.NewLabel("Password:"), p.passEntry,
			widget.NewLabel("Token / Bearer / SID:"), p.tokenEntry,
			widget.NewLabel("Custom Header Key:"), p.hdrKeyEntry,
			widget.NewLabel("Custom Header Value:"), p.hdrValEntry,
			widget.NewLabel("2FA TOTP Secret Key:"), p.totpSecretEntry,
			widget.NewLabel("Trusted Device ID:"), p.deviceIDEntry,
		),
		widget.NewLabelWithStyle("Request Body Template:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWrap(fyne.NewSize(650, 100), p.bodyEntry),
		widget.NewAccordion(widget.NewAccordionItem("Template Syntax Reference", tmplTips)),
		btnBar,
	)

	p.refreshProfileOptions()

	return container.NewVScroll(container.NewVBox(headerBox, widget.NewSeparator(), form))
}

func (p *ProfilesTab) updateAuthFieldVisibility(auth config.AuthType) {
	// Enable/disable fields according to auth type
	switch auth {
	case config.AuthSynologyLogin:
		p.userEntry.Enable()
		p.passEntry.Enable()
		p.tokenEntry.Enable() // can be used for manual SID bypass
		p.hdrKeyEntry.Disable()
		p.hdrValEntry.Disable()
		p.totpSecretEntry.Enable()
		p.deviceIDEntry.Enable()
	case config.AuthBasic, config.AuthQBittorrentLogin:
		p.userEntry.Enable()
		p.passEntry.Enable()
		p.tokenEntry.Disable()
		p.hdrKeyEntry.Disable()
		p.hdrValEntry.Disable()
		p.totpSecretEntry.Disable()
		p.deviceIDEntry.Disable()
	case config.AuthBearer:
		p.userEntry.Disable()
		p.passEntry.Disable()
		p.tokenEntry.Enable()
		p.hdrKeyEntry.Disable()
		p.hdrValEntry.Disable()
		p.totpSecretEntry.Disable()
		p.deviceIDEntry.Disable()
	case config.AuthHeader:
		p.userEntry.Disable()
		p.passEntry.Disable()
		p.tokenEntry.Disable()
		p.hdrKeyEntry.Enable()
		p.hdrValEntry.Enable()
		p.totpSecretEntry.Disable()
		p.deviceIDEntry.Disable()
	case config.AuthTransmissionSession:
		p.userEntry.Enable()
		p.passEntry.Enable()
		p.tokenEntry.Disable()
		p.hdrKeyEntry.Disable()
		p.hdrValEntry.Disable()
		p.totpSecretEntry.Disable()
		p.deviceIDEntry.Disable()
	default:
		p.userEntry.Disable()
		p.passEntry.Disable()
		p.tokenEntry.Disable()
		p.hdrKeyEntry.Disable()
		p.hdrValEntry.Disable()
		p.totpSecretEntry.Disable()
		p.deviceIDEntry.Disable()
	}
}

func (p *ProfilesTab) refreshProfileOptions() {
	options := make([]string, 0, len(p.app.Config.Profiles))
	for _, prof := range p.app.Config.Profiles {
		options = append(options, prof.Name)
	}
	p.profileSelect.Options = options

	active := p.app.Config.GetActiveProfile()
	if active != nil {
		p.profileSelect.SetSelected(active.Name)
		p.loadProfileIntoForm(*active)
	} else if len(p.app.Config.Profiles) > 0 {
		p.profileSelect.SetSelected(p.app.Config.Profiles[0].Name)
		p.loadProfileIntoForm(p.app.Config.Profiles[0])
	}
}

func (p *ProfilesTab) loadProfileIntoForm(prof config.Profile) {
	p.currentID = prof.ID
	p.nameEntry.SetText(prof.Name)
	p.descEntry.SetText(prof.Description)
	p.urlEntry.SetText(prof.BaseURL)
	p.endpointEntry.SetText(prof.Endpoint)
	p.methodSelect.SetSelected(prof.Method)
	p.ctEntry.SetText(prof.ContentType)
	p.authSelect.SetSelected(string(prof.AuthType))
	p.userEntry.SetText(prof.Username)
	p.passEntry.SetText(prof.Password)
	p.tokenEntry.SetText(prof.Token)
	p.hdrKeyEntry.SetText(prof.HeaderKey)
	p.hdrValEntry.SetText(prof.HeaderValue)
	p.bodyEntry.SetText(prof.BodyTemplate)

	if prof.CustomParams != nil {
		p.totpSecretEntry.SetText(prof.CustomParams["totp_secret"])
		p.deviceIDEntry.SetText(prof.CustomParams["device_id"])
	} else {
		p.totpSecretEntry.SetText("")
		p.deviceIDEntry.SetText("")
	}

	p.updateAuthFieldVisibility(prof.AuthType)
}

func (p *ProfilesTab) saveCurrentProfile() {
	if strings.TrimSpace(p.nameEntry.Text) == "" {
		dialog.ShowError(fmt.Errorf("profile name cannot be empty"), p.app.Window)
		return
	}
	if strings.TrimSpace(p.urlEntry.Text) == "" {
		dialog.ShowError(fmt.Errorf("server base URL cannot be empty"), p.app.Window)
		return
	}

	if p.currentID == "" {
		p.currentID = fmt.Sprintf("profile-%d", time.Now().UnixNano())
	}

	existing := p.app.Config.FindProfile(p.currentID)
	customParams := make(map[string]string)
	if existing != nil && existing.CustomParams != nil {
		for k, v := range existing.CustomParams {
			customParams[k] = v
		}
	}

	if totpSec := strings.TrimSpace(p.totpSecretEntry.Text); totpSec != "" {
		customParams["totp_secret"] = totpSec
	} else {
		delete(customParams, "totp_secret")
	}

	if devID := strings.TrimSpace(p.deviceIDEntry.Text); devID != "" {
		customParams["device_id"] = devID
	} else {
		delete(customParams, "device_id")
	}

	updated := config.Profile{
		ID:           p.currentID,
		Name:         strings.TrimSpace(p.nameEntry.Text),
		Description:  strings.TrimSpace(p.descEntry.Text),
		BaseURL:      strings.TrimSpace(p.urlEntry.Text),
		Endpoint:     strings.TrimSpace(p.endpointEntry.Text),
		Method:       p.methodSelect.Selected,
		ContentType:  strings.TrimSpace(p.ctEntry.Text),
		AuthType:     config.AuthType(p.authSelect.Selected),
		Username:     p.userEntry.Text,
		Password:     p.passEntry.Text,
		Token:        p.tokenEntry.Text,
		HeaderKey:    p.hdrKeyEntry.Text,
		HeaderValue:  p.hdrValEntry.Text,
		BodyTemplate: p.bodyEntry.Text,
		CustomParams: customParams,
	}

	p.app.Config.SaveProfile(updated)
	_ = p.app.Config.Save()

	p.app.DispatchTab.RefreshProfiles()
	p.refreshProfileOptions()
	p.app.UpdateTrayMenu()
	dialog.ShowInformation("Saved", fmt.Sprintf("Profile '%s' saved successfully.", updated.Name), p.app.Window)
}

func (p *ProfilesTab) deleteCurrentProfile() {
	if len(p.app.Config.Profiles) <= 1 {
		dialog.ShowError(fmt.Errorf("cannot delete the only profile"), p.app.Window)
		return
	}

	dialog.ShowConfirm("Confirm Delete", fmt.Sprintf("Delete profile '%s'?", p.nameEntry.Text), func(confirm bool) {
		if !confirm {
			return
		}
		_ = p.app.Config.DeleteProfile(p.currentID)
		_ = p.app.Config.Save()
		p.app.DispatchTab.RefreshProfiles()
		p.refreshProfileOptions()
		p.app.UpdateTrayMenu()
	}, p.app.Window)
}

func (p *ProfilesTab) loadPreset(presetName string) {
	var target config.Profile
	defaults := config.DefaultProfiles()

	switch {
	case strings.Contains(presetName, "Synology"):
		target = defaults[0]
	case strings.Contains(presetName, "qBittorrent"):
		target = defaults[1]
	case strings.Contains(presetName, "Transmission"):
		target = defaults[2]
	case strings.Contains(presetName, "Deluge"):
		target = defaults[3]
	default:
		target = defaults[4]
	}

	target.ID = fmt.Sprintf("profile-%d", time.Now().UnixNano())
	target.Name = target.Name + " (Copy)"
	p.app.Config.SaveProfile(target)
	_ = p.app.Config.Save()
	p.app.DispatchTab.RefreshProfiles()
	p.refreshProfileOptions()
	p.profileSelect.SetSelected(target.Name)
	p.loadProfileIntoForm(target)
}

func (p *ProfilesTab) testCurrentProfile() {
	customParams := make(map[string]string)
	if totpSec := strings.TrimSpace(p.totpSecretEntry.Text); totpSec != "" {
		customParams["totp_secret"] = totpSec
	}
	if devID := strings.TrimSpace(p.deviceIDEntry.Text); devID != "" {
		customParams["device_id"] = devID
	}

	testProf := config.Profile{
		ID:           p.currentID,
		Name:         p.nameEntry.Text,
		BaseURL:      p.urlEntry.Text,
		Endpoint:     p.endpointEntry.Text,
		Method:       p.methodSelect.Selected,
		ContentType:  p.ctEntry.Text,
		AuthType:     config.AuthType(p.authSelect.Selected),
		Username:     p.userEntry.Text,
		Password:     p.passEntry.Text,
		Token:        p.tokenEntry.Text,
		HeaderKey:    p.hdrKeyEntry.Text,
		HeaderValue:  p.hdrValEntry.Text,
		BodyTemplate: p.bodyEntry.Text,
		CustomParams: customParams,
	}

	testMag, _ := magnet.Parse("magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=TorrentClick+Test+Torrent")

	p.testBtn.Disable()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		res, err := p.app.Client.Execute(ctx, &testProf, testMag)

		// Check if 2FA OTP is required
		if client.IsOTPRequired(err) {
			fyne.Do(func() {
				p.testBtn.Enable()
				p.app.PromptOTP(
					"Two-Factor Authentication Required",
					"Synology DSM requires an OTP code. Enter the 6-digit code from your authenticator app:",
					func(code string) {
						testProf.CustomParams["otp_code"] = code
						p.runOTPTestRetry(testProf, testMag)
					},
				)
			})
			return
		}

		fyne.Do(func() {
			p.testBtn.Enable()

			var msg string
			if res != nil && res.Success {
				msg = fmt.Sprintf("Connection & Request Succeeded!\nHTTP %d %s\nDuration: %v\n\nResponse:\n%s",
					res.StatusCode, res.StatusText, res.Duration, res.ResponseBody)
				dialog.ShowInformation("Test Succeeded", msg, p.app.Window)
			} else {
				errStr := ""
				if res != nil {
					errStr = fmt.Sprintf("Status %d %s\nError: %s\nBody:\n%s", res.StatusCode, res.StatusText, res.ErrorMessage, res.ResponseBody)
				} else if err != nil {
					errStr = err.Error()
				}
				dialog.ShowError(fmt.Errorf("Test Failed:\n%s", errStr), p.app.Window)
			}
		})
	}()
}

func (p *ProfilesTab) runOTPTestRetry(testProf config.Profile, testMag *magnet.Magnet) {
	p.testBtn.Disable()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		res, err := p.app.Client.Execute(ctx, &testProf, testMag)

		fyne.Do(func() {
			p.testBtn.Enable()

			if res != nil && res.Success {
				// Persist acquired device_id
				if newDevID := testProf.CustomParams["device_id"]; newDevID != "" {
					p.deviceIDEntry.SetText(newDevID)
					p.saveCurrentProfile()
				}

				msg := fmt.Sprintf("2FA Verified & Connection Succeeded!\nHTTP %d %s\n\nTrusted device token was registered and saved automatically. Future logins will not require 2FA.",
					res.StatusCode, res.StatusText)
				dialog.ShowInformation("2FA Verification Successful", msg, p.app.Window)
			} else {
				errStr := ""
				if res != nil {
					errStr = fmt.Sprintf("Status %d %s\nError: %s", res.StatusCode, res.StatusText, res.ErrorMessage)
				} else if err != nil {
					errStr = err.Error()
				}
				dialog.ShowError(fmt.Errorf("2FA Verification Failed:\n%s", errStr), p.app.Window)
			}
		})
	}()
}
