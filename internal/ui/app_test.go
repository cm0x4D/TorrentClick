package ui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"torrentclick/internal/client"
	"torrentclick/internal/config"
	"torrentclick/internal/history"
	"torrentclick/internal/magnet"
)

func TestDeleteTorrentAfterDispatch(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	testWin := testApp.NewWindow("Test")
	defer testWin.Close()

	// 1. Mock server that returns HTTP 200 OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	torrentPath := filepath.Join(tmpDir, "sample.torrent")
	if err := os.WriteFile(torrentPath, []byte("d8:announce27:http://tracker.example.com/4:infod6:lengthi1048576e4:name10:sample.iso12:piece lengthi16384e6:pieces20:12345678901234567890ee"), 0644); err != nil {
		t.Fatal(err)
	}

	mag, err := magnet.Parse(torrentPath)
	if err != nil {
		t.Fatalf("failed to parse torrent: %v", err)
	}

	profile := &config.Profile{
		ID:           "test-server",
		Name:         "Test Server",
		BaseURL:      server.URL,
		Endpoint:     "/api/add",
		Method:       "POST",
		AuthType:     config.AuthNone,
		BodyTemplate: `{"magnet": "{{.Magnet}}"}`,
	}

	cfg := config.NewDefault()
	cfg.DeleteTorrentAfterDispatch = true
	cfg.NotificationsEnabled = false

	hist := history.NewManager(10)
	mainApp := &MainApp{
		FyneApp: testApp,
		Window:  testWin,
		Config:  cfg,
		Client:  client.NewClient(),
		History: hist,
	}
	mainApp.HistoryTab = NewHistoryTab(mainApp)

	var wg sync.WaitGroup
	wg.Add(1)

	mainApp.SendMagnet(mag, profile, func(res *client.Result, err error) {
		defer wg.Done()
		if err != nil {
			t.Errorf("unexpected SendMagnet error: %v", err)
		}
		if res == nil || !res.Success {
			t.Errorf("expected success, got: %v", res)
		}
	})

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SendMagnet to complete")
	}

	// Verify that sample.torrent has been deleted from disk
	if _, statErr := os.Stat(torrentPath); !os.IsNotExist(statErr) {
		t.Errorf("expected %s to be deleted, but it still exists", torrentPath)
	}
}

func TestKeepTorrentWhenDeleteOptionDisabled(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	testWin := testApp.NewWindow("Test")
	defer testWin.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	torrentPath := filepath.Join(tmpDir, "keep.torrent")
	if err := os.WriteFile(torrentPath, []byte("d8:announce27:http://tracker.example.com/4:infod6:lengthi1048576e4:name8:keep.iso12:piece lengthi16384e6:pieces20:12345678901234567890ee"), 0644); err != nil {
		t.Fatal(err)
	}

	mag, err := magnet.Parse(torrentPath)
	if err != nil {
		t.Fatalf("failed to parse torrent: %v", err)
	}

	profile := &config.Profile{
		ID:           "test-server",
		Name:         "Test Server",
		BaseURL:      server.URL,
		Endpoint:     "/api/add",
		Method:       "POST",
		AuthType:     config.AuthNone,
		BodyTemplate: `{"magnet": "{{.Magnet}}"}`,
	}

	cfg := config.NewDefault()
	cfg.DeleteTorrentAfterDispatch = false // disabled
	cfg.NotificationsEnabled = false

	hist := history.NewManager(10)
	mainApp := &MainApp{
		FyneApp: testApp,
		Window:  testWin,
		Config:  cfg,
		Client:  client.NewClient(),
		History: hist,
	}
	mainApp.HistoryTab = NewHistoryTab(mainApp)

	var wg sync.WaitGroup
	wg.Add(1)

	mainApp.SendMagnet(mag, profile, func(res *client.Result, err error) {
		defer wg.Done()
		if err != nil {
			t.Errorf("unexpected SendMagnet error: %v", err)
		}
	})

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SendMagnet to complete")
	}

	// Verify that keep.torrent still exists on disk
	if _, statErr := os.Stat(torrentPath); statErr != nil {
		t.Errorf("expected %s to still exist, but got stat error: %v", torrentPath, statErr)
	}
}
