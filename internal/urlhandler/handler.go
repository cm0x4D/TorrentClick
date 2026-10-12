package urlhandler

import (
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	urlChan    = make(chan string, 10)
	mu         sync.Mutex
	lastTarget string
	lastTime   time.Time
)

// Init initializes platform URL event handling and inspects startup arguments.
func Init(args []string) <-chan string {
	registerPlatformURLHandler()

	// Inspect os.Args for cold-boot URLs or .torrent files
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "magnet:") || strings.HasSuffix(lower, ".torrent") {
			dispatchURL(trimmed)
			break
		}
		// Check if argument is an existing torrent file without extension or with path
		if fi, err := os.Stat(trimmed); err == nil && !fi.IsDir() && strings.HasSuffix(lower, ".torrent") {
			dispatchURL(trimmed)
			break
		}
	}

	return urlChan
}

// dispatchURL handles incoming URLs or file paths and avoids immediate duplicates.
func dispatchURL(u string) {
	trimmed := strings.TrimSpace(u)
	if trimmed == "" {
		return
	}

	// Normalize file:// URLs to clean filesystem paths
	if strings.HasPrefix(trimmed, "file://") {
		parsed, err := url.Parse(trimmed)
		if err == nil && parsed.Path != "" {
			trimmed = parsed.Path
		}
	}

	mu.Lock()
	now := time.Now()
	if trimmed == lastTarget && now.Sub(lastTime) < 2*time.Second {
		mu.Unlock()
		return
	}
	lastTarget = trimmed
	lastTime = now
	mu.Unlock()

	select {
	case urlChan <- trimmed:
	default:
		// Queue full, drop or drain
	}
}
