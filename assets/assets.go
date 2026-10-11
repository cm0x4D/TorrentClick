package assets

import _ "embed"

// TrayIconBytes contains the raw PNG data for the system tray icon.
//
//go:embed tray_icon.png
var TrayIconBytes []byte

// AppIconBytes contains the raw PNG data for the 512x512 application icon.
//
//go:embed icon.png
var AppIconBytes []byte
