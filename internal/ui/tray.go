package ui

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/driver/desktop"

    "torrentclick/assets"
)

// SetupSystemTray configures the macOS menu bar / system tray icon and menu.
func (m *MainApp) SetupSystemTray() {
    desk, ok := m.FyneApp.(desktop.App)
    if !ok {
        return
    }

    trayIcon := fyne.NewStaticResource("tray_icon.png", assets.TrayIconBytes)
    desk.SetSystemTrayIcon(trayIcon)
    m.UpdateTrayMenu()
}

// UpdateTrayMenu updates the menu items in the system tray menu.
func (m *MainApp) UpdateTrayMenu() {
    desk, ok := m.FyneApp.(desktop.App)
    if !ok {
        return
    }

    showItem := fyne.NewMenuItem("Show TorrentClick", func() {
        m.Window.Show()
        m.Window.RequestFocus()
    })

    activeProfile := m.Config.GetActiveProfile()
    activeName := "None"
    if activeProfile != nil {
        activeName = activeProfile.Name
    }
    activeItem := fyne.NewMenuItem("Active: "+activeName, nil)
    activeItem.Disabled = true

    // Profiles submenu
    profileSubmenu := fyne.NewMenu("Switch Client")
    for _, p := range m.Config.Profiles {
        prof := p
        item := fyne.NewMenuItem(prof.Name, func() {
            m.Config.ActiveProfileID = prof.ID
            _ = m.Config.Save()
            m.DispatchTab.RefreshProfiles()
            m.ProfilesTab.refreshProfileOptions()
            m.UpdateTrayMenu()
        })
        if prof.ID == m.Config.ActiveProfileID {
            item.Checked = true
        }
        profileSubmenu.Items = append(profileSubmenu.Items, item)
    }
    switchItem := fyne.NewMenuItem("Switch Client", nil)
    switchItem.ChildMenu = profileSubmenu

    settingsItem := fyne.NewMenuItem("Preferences...", func() {
        m.Tabs.SelectIndex(3)
        m.Window.Show()
        m.Window.RequestFocus()
    })

    menu := fyne.NewMenu("TorrentClick",
        showItem,
        fyne.NewMenuItemSeparator(),
        activeItem,
        switchItem,
        fyne.NewMenuItemSeparator(),
        settingsItem,
        fyne.NewMenuItemSeparator(),
    )

    desk.SetSystemTrayMenu(menu)
}
