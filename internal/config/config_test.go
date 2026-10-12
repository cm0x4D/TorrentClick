package config

import (
	"os"
	"testing"
)

func TestDefaultProfiles(t *testing.T) {
	profiles := DefaultProfiles()
	if len(profiles) < 4 {
		t.Fatalf("expected at least 4 default profiles, got %d", len(profiles))
	}

	foundSynology := false
	foundQBit := false
	foundTransmission := false
	for _, p := range profiles {
		if p.ID == "synology-download-station" {
			foundSynology = true
		}
		if p.ID == "qbittorrent" {
			foundQBit = true
		}
		if p.ID == "transmission" {
			foundTransmission = true
		}
	}

	if !foundSynology {
		t.Errorf("missing Synology profile")
	}
	if !foundQBit {
		t.Errorf("missing qBittorrent profile")
	}
	if !foundTransmission {
		t.Errorf("missing Transmission profile")
	}
}

func TestConfigSaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "torrentclick-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := NewDefault()
	cfg.ActiveProfileID = "qbittorrent"
	cfg.DeleteTorrentAfterDispatch = true
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.ActiveProfileID != "qbittorrent" {
		t.Errorf("expected active profile qbittorrent, got %s", loaded.ActiveProfileID)
	}

	if !loaded.DeleteTorrentAfterDispatch {
		t.Errorf("expected DeleteTorrentAfterDispatch to be true, got false")
	}

	p := loaded.GetActiveProfile()
	if p == nil || p.ID != "qbittorrent" {
		t.Errorf("expected active profile to be found and match")
	}
}

func TestConfigProfileManagement(t *testing.T) {
	cfg := NewDefault()
	initialLen := len(cfg.Profiles)

	newProfile := Profile{
		ID:       "custom-test",
		Name:     "Custom Test",
		BaseURL:  "http://example.com",
		Endpoint: "/torrent",
		Method:   "POST",
	}
	cfg.SaveProfile(newProfile)

	if len(cfg.Profiles) != initialLen+1 {
		t.Errorf("expected %d profiles, got %d", initialLen+1, len(cfg.Profiles))
	}

	found := cfg.FindProfile("custom-test")
	if found == nil || found.Name != "Custom Test" {
		t.Errorf("failed to retrieve saved profile")
	}

	if err := cfg.DeleteProfile("custom-test"); err != nil {
		t.Errorf("failed to delete profile: %v", err)
	}

	if cfg.FindProfile("custom-test") != nil {
		t.Errorf("profile still exists after deletion")
	}
}
