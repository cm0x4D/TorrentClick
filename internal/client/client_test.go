package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"torrentclick/internal/config"
	"torrentclick/internal/magnet"
)

func TestExecuteGenericWebhook(t *testing.T) {
	var receivedBody map[string]interface{}
	var authHdr string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHdr = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "queued"}`))
	}))
	defer server.Close()

	mag, err := magnet.Parse("magnet:?xt=urn:btih:fedcba9876543210&dn=Arch+Linux")
	if err != nil {
		t.Fatal(err)
	}

	p := &config.Profile{
		ID:           "test-generic",
		BaseURL:      server.URL,
		Endpoint:     "/api/torrents",
		Method:       "POST",
		ContentType:  "application/json",
		AuthType:     config.AuthBearer,
		Token:        "secret-token-123",
		BodyTemplate: `{"magnet": "{{.Magnet}}", "name": "{{.DisplayName}}", "hash": "{{.InfoHash}}"}`,
	}

	c := NewClient()
	res, err := c.Execute(context.Background(), p, mag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success result, got error: %s", res.ErrorMessage)
	}
	if authHdr != "Bearer secret-token-123" {
		t.Errorf("expected Bearer secret-token-123, got: %s", authHdr)
	}
	if receivedBody["name"] != "Arch Linux" {
		t.Errorf("expected Arch Linux in body, got: %v", receivedBody["name"])
	}
	if receivedBody["hash"] != "fedcba9876543210" {
		t.Errorf("expected hash in body, got: %v", receivedBody["hash"])
	}
}

func TestExecuteSynologyLoginFlow(t *testing.T) {
	loginCalled := false
	taskCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "auth.cgi") {
			loginCalled = true
			if r.URL.Query().Get("account") == "testuser" && r.URL.Query().Get("passwd") == "testpass" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"sid":"syno_sess_999"},"success":true}`))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"error":{"code":400},"success":false}`))
			}
			return
		}

		if strings.Contains(r.URL.Path, "task.cgi") {
			taskCalled = true
			cookieHdr := r.Header.Get("Cookie")
			if strings.Contains(cookieHdr, "syno_sess_999") {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"success":true}`))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"error":{"code":105},"success":false}`))
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mag, err := magnet.Parse("magnet:?xt=urn:btih:1122334455667788&dn=SynoTest")
	if err != nil {
		t.Fatal(err)
	}

	p := &config.Profile{
		ID:           "test-syno",
		Type:         "synology",
		BaseURL:      server.URL,
		Endpoint:     "/webapi/DownloadStation/task.cgi",
		Method:       "POST",
		ContentType:  "application/x-www-form-urlencoded",
		AuthType:     config.AuthSynologyLogin,
		Username:     "testuser",
		Password:     "testpass",
		BodyTemplate: `api=SYNO.DownloadStation.Task&version=1&method=create&uri={{.MagnetEscaped}}&_sid={{.SessionID}}`,
	}

	c := NewClient()
	res, err := c.Execute(context.Background(), p, mag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !loginCalled {
		t.Errorf("Synology login was not called")
	}
	if !taskCalled {
		t.Errorf("Synology task endpoint was not called")
	}
	if !res.Success {
		t.Errorf("expected success, got: %s", res.ErrorMessage)
	}

	// Subsequent execution should reuse cached session ID without calling login again
	loginCalled = false
	_, err = c.Execute(context.Background(), p, mag)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if loginCalled {
		t.Errorf("Synology login called again despite cached session")
	}
}

func TestExecuteTransmissionHandshake(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		token := r.Header.Get("X-Transmission-Session-Id")
		if token == "" {
			w.Header().Set("X-Transmission-Session-Id", "tx_session_token_xyz")
			w.WriteHeader(http.StatusConflict) // 409
			return
		}
		if token == "tx_session_token_xyz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	mag, err := magnet.Parse("magnet:?xt=urn:btih:aabbccddeeff&dn=TxTest")
	if err != nil {
		t.Fatal(err)
	}

	p := &config.Profile{
		ID:           "test-transmission",
		Type:         "transmission",
		BaseURL:      server.URL,
		Endpoint:     "/transmission/rpc",
		Method:       "POST",
		ContentType:  "application/json",
		AuthType:     config.AuthTransmissionSession,
		BodyTemplate: `{"method":"torrent-add","arguments":{"filename":"{{.Magnet}}"}}`,
	}

	c := NewClient()
	res, err := c.Execute(context.Background(), p, mag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success, got: %s", res.ErrorMessage)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (initial 409 + retry), got %d", callCount)
	}
}
