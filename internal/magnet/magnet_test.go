package magnet

import (
	"testing"
)

func TestParseValidMagnet(t *testing.T) {
	raw := "magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335380dc1f74c26&dn=Ubuntu+22.04+LTS&tr=http%3A%2F%2Ftracker.example.com%2Fannounce&tr=udp%3A%2F%2Ftracker2.example.com%3A80"
	m, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.InfoHash != "c12fe1c06bba254a9dc9f519b335380dc1f74c26" {
		t.Errorf("expected hash c12fe1c06bba254a9dc9f519b335380dc1f74c26, got %s", m.InfoHash)
	}
	if m.DisplayName != "Ubuntu 22.04 LTS" {
		t.Errorf("expected display name 'Ubuntu 22.04 LTS', got %s", m.DisplayName)
	}
	if len(m.Trackers) != 2 {
		t.Fatalf("expected 2 trackers, got %d", len(m.Trackers))
	}
	if m.Trackers[0] != "http://tracker.example.com/announce" {
		t.Errorf("expected tracker 0, got %s", m.Trackers[0])
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse("http://example.com/file.torrent")
	if err == nil {
		t.Errorf("expected error for non-magnet scheme")
	}

	_, err = Parse("")
	if err == nil {
		t.Errorf("expected error for empty string")
	}
}

func TestToTemplateContext(t *testing.T) {
	raw := "magnet:?xt=urn:btih:1234567890abcdef&dn=Test+Movie"
	m, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := map[string]string{
		"DownloadDir": "/volume1/downloads",
		"Category":    "movies",
	}
	ctx := m.ToTemplateContext(cfg, "session-xyz")

	if ctx["DisplayName"] != "Test Movie" {
		t.Errorf("context display name mismatch")
	}
	if ctx["SessionID"] != "session-xyz" {
		t.Errorf("context session mismatch")
	}
	cfgMap := ctx["Config"].(map[string]string)
	if cfgMap["Category"] != "movies" {
		t.Errorf("expected category 'movies', got %s", cfgMap["Category"])
	}
}
