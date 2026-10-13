package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// AuthType represents how authentication is handled for a client profile.
type AuthType string

const (
	AuthNone                AuthType = "none"
	AuthBasic               AuthType = "basic"
	AuthBearer              AuthType = "bearer"
	AuthHeader              AuthType = "header"
	AuthSynologyLogin       AuthType = "synology_login"
	AuthQBittorrentLogin    AuthType = "qbittorrent_login"
	AuthTransmissionSession AuthType = "transmission_session"
)

// Profile defines the configuration and templates for interacting with a torrent client or REST endpoint.
type Profile struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Type         string            `json:"type"` // "synology", "qbittorrent", "transmission", "deluge", "generic"
	BaseURL      string            `json:"base_url"`
	Endpoint     string            `json:"endpoint"`
	Method       string            `json:"method"`
	ContentType  string            `json:"content_type"`
	AuthType     AuthType          `json:"auth_type"`
	Username     string            `json:"username,omitempty"`
	Password     string            `json:"password,omitempty"`
	Token        string            `json:"token,omitempty"`
	HeaderKey    string            `json:"header_key,omitempty"`
	HeaderValue  string            `json:"header_value,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	QueryParams  map[string]string `json:"query_params,omitempty"`
	BodyTemplate string            `json:"body_template,omitempty"`
	CustomParams map[string]string `json:"custom_params,omitempty"`
}

// Config holds all app configuration settings.
type Config struct {
	ActiveProfileID           string    `json:"active_profile_id"`
	AutoSend                  bool      `json:"auto_send"`
	NotificationsEnabled      bool      `json:"notifications_enabled"`
	HideWindowAfterSend       bool      `json:"hide_window_after_send"`
	DeleteTorrentAfterDispatch bool      `json:"delete_torrent_after_dispatch"`
	ThemeMode                  string    `json:"theme_mode"` // "auto", "dark", "light"
	IconDisplayMode            string    `json:"icon_display_mode"` // "both", "tray", "dock"
	Profiles                   []Profile `json:"profiles"`
	mu                        sync.RWMutex
}

// DefaultProfiles provides ready-to-use profiles for popular clients.
func DefaultProfiles() []Profile {
	return []Profile{
		{
			ID:          "synology-download-station",
			Name:        "Synology Download Station",
			Description: "Synology DSM WebAPI DownloadStation.Task create",
			Type:        "synology",
			BaseURL:     "http://synology.local:5000",
			Endpoint:    "/webapi/DownloadStation/task.cgi",
			Method:      "POST",
			ContentType: "application/x-www-form-urlencoded",
			AuthType:    AuthSynologyLogin,
			Username:    "admin",
			Password:    "",
			BodyTemplate: "api=SYNO.DownloadStation.Task&version=1&method=create&uri={{.MagnetEscaped}}&_sid={{.SessionID}}",
			CustomParams: map[string]string{
				"destination": "",
			},
		},
		{
			ID:          "qbittorrent",
			Name:        "qBittorrent Web UI",
			Description: "qBittorrent v4+ Web API (/api/v2/torrents/add)",
			Type:        "qbittorrent",
			BaseURL:     "http://localhost:8080",
			Endpoint:    "/api/v2/torrents/add",
			Method:      "POST",
			ContentType: "application/x-www-form-urlencoded",
			AuthType:    AuthQBittorrentLogin,
			Username:    "admin",
			Password:    "adminadmin",
			BodyTemplate: "urls={{.Magnet}}&savepath={{index .Config \"savepath\"}}&category={{index .Config \"category\"}}",
			CustomParams: map[string]string{
				"savepath": "",
				"category": "",
			},
		},
		{
			ID:          "transmission",
			Name:        "Transmission RPC",
			Description: "Transmission JSON-RPC (HTTP /transmission/rpc)",
			Type:        "transmission",
			BaseURL:     "http://localhost:9091",
			Endpoint:    "/transmission/rpc",
			Method:      "POST",
			ContentType: "application/json",
			AuthType:    AuthTransmissionSession,
			Username:    "",
			Password:    "",
			BodyTemplate: `{"method": "torrent-add", "arguments": {"filename": "{{.Magnet}}"}}`,
			CustomParams: map[string]string{
				"download-dir": "",
			},
		},
		{
			ID:          "deluge",
			Name:        "Deluge Web UI",
			Description: "Deluge Web JSON-RPC (/json)",
			Type:        "deluge",
			BaseURL:     "http://localhost:8112",
			Endpoint:    "/json",
			Method:      "POST",
			ContentType: "application/json",
			AuthType:    AuthNone,
			Password:    "deluge",
			BodyTemplate: `{"method": "core.add_torrent_magnet", "params": ["{{.Magnet}}", {}], "id": 1}`,
		},
		{
			ID:          "generic-webhook",
			Name:        "Generic REST Webhook",
			Description: "Custom JSON POST endpoint with Bearer Token auth",
			Type:        "generic",
			BaseURL:     "https://api.example.com",
			Endpoint:    "/api/torrents",
			Method:      "POST",
			ContentType: "application/json",
			AuthType:    AuthBearer,
			Token:       "your-api-token-here",
			BodyTemplate: `{"magnet": "{{.Magnet}}", "name": "{{.DisplayName}}", "hash": "{{.InfoHash}}"}`,
			Headers: map[string]string{
				"X-Source": "TorrentClick",
			},
		},
	}
}

// NewDefault creates a Config with defaults.
func NewDefault() *Config {
	profiles := DefaultProfiles()
	return &Config{
		ActiveProfileID:            profiles[0].ID,
		AutoSend:                   false,
		NotificationsEnabled:       true,
		HideWindowAfterSend:        false,
		DeleteTorrentAfterDispatch: false,
		ThemeMode:                  "auto",
		IconDisplayMode:            "both",
		Profiles:                   profiles,
	}
}

// GetActiveProfile returns the currently selected profile.
func (c *Config) GetActiveProfile() *Profile {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, p := range c.Profiles {
		if p.ID == c.ActiveProfileID {
			pCopy := p
			return &pCopy
		}
	}
	if len(c.Profiles) > 0 {
		pCopy := c.Profiles[0]
		return &pCopy
	}
	return nil
}

// FindProfile returns the profile with the given ID.
func (c *Config) FindProfile(id string) *Profile {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, p := range c.Profiles {
		if p.ID == id {
			pCopy := p
			return &pCopy
		}
	}
	return nil
}

// SaveProfile updates or adds a profile.
func (c *Config) SaveProfile(p Profile) {
	c.mu.Lock()
	defer c.mu.Unlock()

	found := false
	for i, existing := range c.Profiles {
		if existing.ID == p.ID {
			c.Profiles[i] = p
			found = true
			break
		}
	}
	if !found {
		c.Profiles = append(c.Profiles, p)
	}
}

// DeleteProfile removes a profile by ID.
func (c *Config) DeleteProfile(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.Profiles) <= 1 {
		return fmt.Errorf("cannot delete the only profile")
	}

	idx := -1
	for i, p := range c.Profiles {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("profile not found: %s", id)
	}

	c.Profiles = append(c.Profiles[:idx], c.Profiles[idx+1:]...)
	if c.ActiveProfileID == id {
		c.ActiveProfileID = c.Profiles[0].ID
	}
	return nil
}

// ConfigFilePath returns the platform-appropriate config file path.
func ConfigFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	dir := filepath.Join(configDir, "TorrentClick")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load loads the configuration from disk, creating default if not exists.
func Load() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return NewDefault(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := NewDefault()
			_ = cfg.Save()
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		cfg.Profiles = DefaultProfiles()
		cfg.ActiveProfileID = cfg.Profiles[0].ID
	}

	if cfg.IconDisplayMode == "" {
		cfg.IconDisplayMode = "both"
	}

	return &cfg, nil
}

// Save writes the configuration to disk.
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
