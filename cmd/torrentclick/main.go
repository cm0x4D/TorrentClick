package main

import (
	"log"
	"os"
	"strings"

	"fyne.io/fyne/v2"

	"torrentclick/internal/config"
	"torrentclick/internal/history"
	"torrentclick/internal/sysdark"
	"torrentclick/internal/ui"
	"torrentclick/internal/urlhandler"
)

func main() {
	// 1. Initialize URL handler early to capture Apple Events and cold boot arguments
	urlChan := urlhandler.Init(os.Args)

	// 2. Load configurations and persistent history
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: failed to load config, using defaults: %v", err)
		cfg = config.NewDefault()
	}

	hist := history.NewManager(100)

	// 3. Construct and display UI
	appUI := ui.NewMainApp(cfg, hist)

	// 4. Background listener for incoming magnet URLs (warm or cold boot)
	go func() {
		for u := range urlChan {
			magnetURL := u
			log.Printf("Received magnet URL: %s", magnetURL)
			fyne.Do(func() {
				appUI.HandleIncomingMagnet(magnetURL)
			})
		}
	}()

	// 5. Background listener for macOS appearance changes (Dark/Light mode toggles)
	themeChan := sysdark.StartWatcher()
	go func() {
		for isDark := range themeChan {
			log.Printf("macOS appearance changed: DarkMode=%v", isDark)
			fyne.Do(func() {
				appUI.ApplyTheme()
			})
		}
	}()

	// 6. Check if started with magnet link; if so, run silently in background tray
	startedWithMagnet := false
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(arg)), "magnet:") {
			startedWithMagnet = true
			break
		}
	}

	if startedWithMagnet {
		appUI.Window.Hide()
		appUI.FyneApp.Run()
	} else {
		appUI.Window.ShowAndRun()
	}
}
