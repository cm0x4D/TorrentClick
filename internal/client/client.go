package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"torrentclick/internal/config"
	"torrentclick/internal/magnet"
)

// Result records the outcome of sending a torrent request.
type Result struct {
	Success       bool          `json:"success"`
	StatusCode    int           `json:"status_code"`
	StatusText    string        `json:"status_text"`
	RequestURL    string        `json:"request_url"`
	RequestMethod string        `json:"request_method"`
	RequestBody   string        `json:"request_body"`
	ResponseBody  string        `json:"response_body"`
	Duration      time.Duration `json:"duration"`
	ErrorMessage  string        `json:"error_message,omitempty"`
}

// Client handles executing templated HTTP requests for torrent clients.
type Client struct {
	httpClient *http.Client
	sessionMu  sync.RWMutex
	sessions   map[string]string // profileID -> sessionID
}

// NewClient initializes a new Client instance.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		sessions: make(map[string]string),
	}
}

// GetSession returns a cached session ID for a profile.
func (c *Client) GetSession(profileID string) string {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return c.sessions[profileID]
}

// SetSession updates the cached session ID for a profile.
func (c *Client) SetSession(profileID, sessionID string) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	c.sessions[profileID] = sessionID
}

// ClearSession clears the cached session ID for a profile.
func (c *Client) ClearSession(profileID string) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	delete(c.sessions, profileID)
}

// Execute performs the templated REST call for the given profile and magnet link.
func (c *Client) Execute(ctx context.Context, p *config.Profile, m *magnet.Magnet) (*Result, error) {
	if p == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	if m == nil {
		return nil, fmt.Errorf("magnet cannot be nil")
	}

	start := time.Now()

	// 1. Handle specialized pre-auth / session initialization
	sessionID := c.GetSession(p.ID)
	if sessionID == "" {
		var err error
		sessionID, err = c.authenticateIfNeeded(ctx, p)
		if err != nil {
			return &Result{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Authentication failed: %v", err),
				Duration:     time.Since(start),
			}, err
		}
		if sessionID != "" {
			c.SetSession(p.ID, sessionID)
		}
	}

	// 2. Perform the main request (with retry on auth invalidation if applicable)
	res, retryAuth, err := c.doRequest(ctx, p, m, sessionID)
	if retryAuth {
		c.ClearSession(p.ID)
		sessionID, authErr := c.authenticateIfNeeded(ctx, p)
		if authErr == nil {
			c.SetSession(p.ID, sessionID)
			res, _, err = c.doRequest(ctx, p, m, sessionID)
		}
	}

	res.Duration = time.Since(start)
	return res, err
}

func (c *Client) doRequest(ctx context.Context, p *config.Profile, m *magnet.Magnet, sessionID string) (*Result, bool, error) {
	data := m.ToTemplateContext(p.CustomParams, sessionID)

	// Build full target URL
	fullURLStr := strings.TrimRight(p.BaseURL, "/") + "/" + strings.TrimLeft(p.Endpoint, "/")
	renderedURL, err := renderTemplate("url", fullURLStr, data)
	if err != nil {
		return &Result{Success: false, ErrorMessage: fmt.Sprintf("failed to render URL template: %v", err)}, false, err
	}

	parsedURL, err := url.Parse(renderedURL)
	if err != nil {
		return &Result{Success: false, ErrorMessage: fmt.Sprintf("invalid rendered URL: %v", err)}, false, err
	}

	// Render query parameters
	if len(p.QueryParams) > 0 {
		q := parsedURL.Query()
		for k, vTmpl := range p.QueryParams {
			renderedVal, err := renderTemplate(k, vTmpl, data)
			if err != nil {
				return &Result{Success: false, ErrorMessage: fmt.Sprintf("failed to render query param %s: %v", k, err)}, false, err
			}
			q.Set(k, renderedVal)
		}
		parsedURL.RawQuery = q.Encode()
	}

	// Render body
	var reqBodyBytes []byte
	var reqBodyStr string
	if p.BodyTemplate != "" {
		renderedBody, err := renderTemplate("body", p.BodyTemplate, data)
		if err != nil {
			return &Result{Success: false, ErrorMessage: fmt.Sprintf("failed to render body template: %v", err)}, false, err
		}
		reqBodyStr = renderedBody
		reqBodyBytes = []byte(renderedBody)
	}

	method := strings.ToUpper(strings.TrimSpace(p.Method))
	if method == "" {
		method = http.MethodPost
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), bytes.NewReader(reqBodyBytes))
	if err != nil {
		return &Result{Success: false, ErrorMessage: fmt.Sprintf("failed to build HTTP request: %v", err)}, false, err
	}

	if p.ContentType != "" {
		httpReq.Header.Set("Content-Type", p.ContentType)
	}

	// Render and set custom headers
	for k, vTmpl := range p.Headers {
		renderedHdr, err := renderTemplate("hdr_"+k, vTmpl, data)
		if err == nil {
			httpReq.Header.Set(k, renderedHdr)
		}
	}

	// Handle standard auth types
	switch p.AuthType {
	case config.AuthBasic:
		httpReq.SetBasicAuth(p.Username, p.Password)
	case config.AuthBearer:
		httpReq.Header.Set("Authorization", "Bearer "+p.Token)
	case config.AuthHeader:
		if p.HeaderKey != "" {
			val, _ := renderTemplate("hdr_auth", p.HeaderValue, data)
			if val == "" {
				val = p.Token
			}
			httpReq.Header.Set(p.HeaderKey, val)
		}
	case config.AuthTransmissionSession:
		if p.Username != "" || p.Password != "" {
			httpReq.SetBasicAuth(p.Username, p.Password)
		}
		if sessionID != "" {
			httpReq.Header.Set("X-Transmission-Session-Id", sessionID)
		}
	case config.AuthQBittorrentLogin:
		if sessionID != "" {
			httpReq.Header.Set("Cookie", sessionID)
		}
	case config.AuthSynologyLogin:
		// Synology session can be passed via _sid param or cookie
		if sessionID != "" {
			httpReq.Header.Set("Cookie", "id="+sessionID)
		}
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &Result{
			Success:       false,
			RequestURL:    parsedURL.String(),
			RequestMethod: method,
			RequestBody:   reqBodyStr,
			ErrorMessage:  fmt.Sprintf("HTTP request error: %v", err),
		}, false, err
	}
	defer resp.Body.Close()

	respBodyBytes, _ := io.ReadAll(resp.Body)
	respBodyStr := string(respBodyBytes)

	// Check for Transmission 409 session handshake
	if p.AuthType == config.AuthTransmissionSession && resp.StatusCode == http.StatusConflict {
		newSession := resp.Header.Get("X-Transmission-Session-Id")
		if newSession != "" {
			c.SetSession(p.ID, newSession)
			return c.doRequest(ctx, p, m, newSession)
		}
	}

	// Check if Synology returned session error (e.g. error code 105)
	if p.AuthType == config.AuthSynologyLogin {
		if strings.Contains(respBodyStr, `"error":105`) || strings.Contains(respBodyStr, `"error":106`) {
			return &Result{
				Success:       false,
				StatusCode:    resp.StatusCode,
				StatusText:    resp.Status,
				RequestURL:    parsedURL.String(),
				RequestMethod: method,
				RequestBody:   reqBodyStr,
				ResponseBody:  respBodyStr,
			}, true, nil // signal to retry auth
		}
	}

	// Check if qBittorrent returned 403 Forbidden
	if p.AuthType == config.AuthQBittorrentLogin && resp.StatusCode == http.StatusForbidden {
		return &Result{
			Success:       false,
			StatusCode:    resp.StatusCode,
			StatusText:    resp.Status,
			RequestURL:    parsedURL.String(),
			RequestMethod: method,
			RequestBody:   reqBodyStr,
			ResponseBody:  respBodyStr,
		}, true, nil
	}

	// Evaluate success: 2xx status code and (if Synology) "success": true
	isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 300
	if p.Type == "synology" && !strings.Contains(respBodyStr, `"success":true`) {
		isSuccess = false
	}

	errMsg := ""
	if !isSuccess {
		errMsg = fmt.Sprintf("Server returned status %d: %s", resp.StatusCode, respBodyStr)
	}

	return &Result{
		Success:       isSuccess,
		StatusCode:    resp.StatusCode,
		StatusText:    resp.Status,
		RequestURL:    parsedURL.String(),
		RequestMethod: method,
		RequestBody:   reqBodyStr,
		ResponseBody:  respBodyStr,
		ErrorMessage:  errMsg,
	}, false, nil
}

// authenticateIfNeeded handles client-specific login requests to obtain session tokens.
func (c *Client) authenticateIfNeeded(ctx context.Context, p *config.Profile) (string, error) {
	switch p.AuthType {
	case config.AuthSynologyLogin:
		return c.loginSynology(ctx, p)
	case config.AuthQBittorrentLogin:
		return c.loginQBittorrent(ctx, p)
	default:
		return "", nil
	}
}

func (c *Client) loginSynology(ctx context.Context, p *config.Profile) (string, error) {
	loginURL := fmt.Sprintf("%s/webapi/auth.cgi?api=SYNO.API.Auth&version=3&method=login&account=%s&passwd=%s&session=DownloadStation&format=sid",
		strings.TrimRight(p.BaseURL, "/"),
		url.QueryEscape(p.Username),
		url.QueryEscape(p.Password),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loginURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Synology login network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Synology login response: %w", err)
	}

	var parsed struct {
		Success bool `json:"success"`
		Data    struct {
			Sid string `json:"sid"`
		} `json:"data"`
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse Synology login JSON (%s): %w", string(body), err)
	}

	if !parsed.Success || parsed.Data.Sid == "" {
		return "", fmt.Errorf("Synology login failed with error code %d (response: %s)", parsed.Error.Code, string(body))
	}

	return parsed.Data.Sid, nil
}

func (c *Client) loginQBittorrent(ctx context.Context, p *config.Profile) (string, error) {
	loginURL := fmt.Sprintf("%s/api/v2/auth/login", strings.TrimRight(p.BaseURL, "/"))
	form := url.Values{
		"username": {p.Username},
		"password": {p.Password},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("qBittorrent login network error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if resp.StatusCode != http.StatusOK || strings.Contains(strings.ToLower(bodyStr), "fails") {
		return "", fmt.Errorf("qBittorrent login failed (status %d): %s", resp.StatusCode, bodyStr)
	}

	var sidCookie string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "SID" {
			sidCookie = cookie.String()
			break
		}
	}
	if sidCookie == "" {
		// Use raw set-cookie header if any
		sidCookie = resp.Header.Get("Set-Cookie")
	}

	return sidCookie, nil
}

var templateFuncMap = template.FuncMap{
	"urlquery": url.QueryEscape,
	"lower":    strings.ToLower,
	"upper":    strings.ToUpper,
	"trim":     strings.TrimSpace,
	"json": func(v interface{}) string {
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	},
	"default": func(defVal string, val string) string {
		if val == "" {
			return defVal
		}
		return val
	},
}

func renderTemplate(name, tmplStr string, data interface{}) (string, error) {
	if tmplStr == "" {
		return "", nil
	}
	t, err := template.New(name).Funcs(templateFuncMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parse error in %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execute error in %s: %w", name, err)
	}
	return buf.String(), nil
}
