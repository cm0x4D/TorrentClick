package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry records an attempt to dispatch a torrent.
type Entry struct {
	ID                string        `json:"id"`
	Timestamp         time.Time     `json:"timestamp"`
	MagnetURI         string        `json:"magnet_uri"`
	DisplayName       string        `json:"display_name"`
	InfoHash          string        `json:"info_hash"`
	TargetProfileName string        `json:"target_profile_name"`
	Success           bool          `json:"success"`
	StatusCode        int           `json:"status_code"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	Duration          time.Duration `json:"duration"`
}

// Manager manages persistent torrent dispatch history.
type Manager struct {
	mu      sync.RWMutex
	entries []Entry
	maxSize int
}

// NewManager initializes the history manager.
func NewManager(maxSize int) *Manager {
	if maxSize <= 0 {
		maxSize = 100
	}
	m := &Manager{
		entries: make([]Entry, 0),
		maxSize: maxSize,
	}
	_ = m.Load()
	return m
}

func historyFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	dir := filepath.Join(configDir, "TorrentClick")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// Add appends an entry to the top of the history list and saves.
func (m *Manager) Add(e Entry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Prepend
	m.entries = append([]Entry{e}, m.entries...)
	if len(m.entries) > m.maxSize {
		m.entries = m.entries[:m.maxSize]
	}
	_ = m.saveLocked()
}

// List returns a copy of the recorded history entries.
func (m *Manager) List() []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Entry, len(m.entries))
	copy(result, m.entries)
	return result
}

// Clear wipes the history log.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make([]Entry, 0)
	_ = m.saveLocked()
}

func (m *Manager) Load() error {
	path, err := historyFilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return json.Unmarshal(data, &m.entries)
}

func (m *Manager) saveLocked() error {
	path, err := historyFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
