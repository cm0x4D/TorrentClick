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

func TestExecuteSynology2FAWorkflow(t *testing.T) {
	trustedDevice := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "auth.cgi") {
			otp := r.URL.Query().Get("otp_code")
			devID := r.URL.Query().Get("device_id")

			if devID == "trusted_device_abc123" {
				// Trusted device bypasses OTP!
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"sid":"syno_sess_trusted"},"success":true}`))
				return
			}

			if otp == "" {
				// No OTP provided -> 403 error
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"error":{"code":403,"errors":{"types":[{"type":"otp"}]}},"success":false}`))
				return
			}

			if otp == "123456" {
				trustedDevice = "trusted_device_abc123"
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"sid":"syno_sess_otp_ok","device_id":"trusted_device_abc123"},"success":true}`))
				return
			}

			// Wrong OTP -> 404 error
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"error":{"code":404},"success":false}`))
			return
		}

		if strings.Contains(r.URL.Path, "task.cgi") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true}`))
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
		ID:           "test-syno-2fa",
		Type:         "synology",
		BaseURL:      server.URL,
		Endpoint:     "/webapi/DownloadStation/task.cgi",
		Method:       "POST",
		ContentType:  "application/x-www-form-urlencoded",
		AuthType:     config.AuthSynologyLogin,
		Username:     "user",
		Password:     "pass",
		CustomParams: make(map[string]string),
		BodyTemplate: `api=SYNO.DownloadStation.Task&version=1&method=create&uri={{.MagnetEscaped}}&_sid={{.SessionID}}`,
	}

	c := NewClient()

	// 1. Initial attempt without OTP -> should return ErrOTPRequired
	_, err = c.Execute(context.Background(), p, mag)
	if err == nil || !IsOTPRequired(err) {
		t.Fatalf("expected ErrOTPRequired, got: %v", err)
	}

	// 2. Retry with OTP "123456"
	p.CustomParams["otp_code"] = "123456"
	res, err := c.Execute(context.Background(), p, mag)
	if err != nil {
		t.Fatalf("unexpected error after providing OTP: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got: %s", res.ErrorMessage)
	}

	// Verify device_id was saved to CustomParams
	if p.CustomParams["device_id"] != "trusted_device_abc123" {
		t.Errorf("expected device_id to be stored, got: %s", p.CustomParams["device_id"])
	}

	// 3. Subsequent execution with cleared session cache -> should login using device_id without OTP!
	c.ClearSession(p.ID)
	delete(p.CustomParams, "otp_code")
	res, err = c.Execute(context.Background(), p, mag)
	if err != nil {
		t.Fatalf("unexpected error with device_id: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success with device_id, got: %s", res.ErrorMessage)
	}
	if trustedDevice != "trusted_device_abc123" {
		t.Errorf("device was not marked trusted")
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
