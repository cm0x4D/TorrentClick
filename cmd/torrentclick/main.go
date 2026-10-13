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

	// 4. Background listener for incoming targets (magnet URLs or .torrent files, warm or cold boot)
	go func() {
		for u := range urlChan {
			target := u
			log.Printf("Received magnet link or torrent file: %s", target)
			fyne.Do(func() {
				appUI.HandleIncomingTarget(target)
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

	// 6. Check if started with magnet link, custom torrent scheme, or .torrent file; if so, run silently in background tray
	startedWithTarget := false
	for _, arg := range os.Args[1:] {
		trimmed := strings.ToLower(strings.TrimSpace(arg))
		if strings.HasPrefix(trimmed, "magnet:") || strings.HasSuffix(trimmed, ".torrent") ||
			strings.HasPrefix(trimmed, "torrentclick:") || strings.HasPrefix(trimmed, "torrent:") {
			startedWithTarget = true
			break
		}
	}

	if startedWithTarget {
		appUI.Window.Hide()
		appUI.FyneApp.Run()
	} else {
		appUI.Window.ShowAndRun()
	}
}
