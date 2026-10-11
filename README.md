# TorrentClick 🧲⚡

A modern macOS desktop application written in **Go** and **Fyne** that registers as the system URL handler for torrent `magnet:` links. When a magnet link is clicked anywhere on macOS (Safari, Chrome, Firefox, Slack, etc.) or pasted manually, TorrentClick intercepts it and dispatches a configurable REST API call using **Go text/template** and authentication to your remote or local torrent client.

> [!WARNING]
> **Disclaimer**: This entire project was vibe coded without even having a detailed look at the code and has only been tested to work with Synology DSM.

---

## Features

- **macOS Magnet URL Scheme Handler**:
  - Registers `CFBundleURLTypes` for `magnet:` links in macOS `Info.plist`.
  - Handles **cold boots** (app opened via `os.Args` when clicking a magnet link).
  - Handles **warm starts** (app already running, receives macOS Cocoa Apple Events `kInternetEventClass` / `kAEGetURL` via a lightweight Cgo bridge).
- **macOS System Tray / Menu Bar Icon**:
  - Resides in the macOS menu bar status item tray with a custom template icon.
  - Tray menu provides options to Show Window, inspect Active Client, switch target clients directly, open Preferences, and Quit.
  - Closing the main window (red close button) hides to the tray instead of terminating the process.
- **Silent Background Interception**:
  - When the application is already running, clicking any `magnet:` link processes and dispatches it **completely silently in the background**.
  - The window **never pops up or steals focus**.
  - A native macOS desktop notification confirms when the torrent is queued or reports errors.
- **Configurable REST API Dispatch Engine with Go Templates**:
  - Dynamic request URLs, endpoints, headers, query parameters, and request bodies.
  - Template variables: `{{.Magnet}}`, `{{.MagnetEscaped}}`, `{{.InfoHash}}`, `{{.DisplayName}}`, `{{.SessionID}}`, `{{.Trackers}}`, `{{index .Config "param"}}`.
  - Template helper functions: `urlquery`, `json`, `trim`, `lower`, `upper`, `default`.
- **Pre-configured Client Presets**:
  - **Synology Download Station**: Automated DSM WebAPI authentication (`SYNO.API.Auth` login with `_sid` session token retrieval and error code 105 auto-relogin) + `SYNO.DownloadStation.Task` create task call.
  - **qBittorrent**: Web API (`/api/v2/auth/login` session cookie + `/api/v2/torrents/add`).
  - **Transmission**: JSON-RPC over HTTP (`/transmission/rpc`) with automatic `409 Conflict` `X-Transmission-Session-Id` handshake and Basic Auth.
  - **Deluge**: Web UI JSON-RPC (`/json`).
  - **Generic Webhook**: Custom JSON POST endpoint with Bearer Token or custom headers.
- **Desktop Theme Synchronization**:
  - Automatically matches macOS Dark Mode / Light Mode at startup.
  - Listens to native `AppleInterfaceThemeChangedNotification` events in real time to switch themes dynamically when macOS appearance changes.
  - Configurable in the Settings tab (`Match macOS System (Auto)`, `Dark Mode`, `Light Mode`).
- **Modern Fyne UI**:
  - **Dispatch Tab**: Live parsing of magnet links (name, info hash, trackers count), destination profile picker, one-click send, response status and inspection modal.
  - **Profiles Tab**: Visual profile editor and preset loader, test connection button.
  - **History Tab**: Timestamped log of past dispatches with status codes and one-click resend.
  - **Settings Tab**: Auto-send on click toggle, desktop notifications, auto-hide window after dispatch, and LaunchServices registration tool.
- **Automated Task Runner**:
  - Managed via `go-task` (`Taskfile.yml`) for compilation, tests, icon generation, packaging, and testing.

---

## Project Structure

```
TorrentClick/
├── Taskfile.yml              # Task runner configuration (build, test, package, open)
├── FyneApp.toml              # Fyne app metadata
├── cmd/
│   └── torrentclick/
│       └── main.go           # Application entrypoint & Apple Event dispatcher
├── internal/
│   ├── client/               # REST API engine & client templates
│   │   ├── client.go         # HTTP caller, auth handshakes, template evaluation
│   │   └── client_test.go    # Unit tests for generic webhooks, Synology DSM, Transmission
│   ├── config/               # Configuration storage & profile models
│   │   ├── config.go         # Profile definition & persistence (~/Library/Application Support)
│   │   └── config_test.go    # Config management unit tests
│   ├── history/              # Persistent dispatch history
│   │   └── history.go
│   ├── magnet/               # Magnet URI parser & template context builder
│   │   ├── magnet.go
│   │   └── magnet_test.go
│   ├── ui/                   # Fyne UI views
│   │   ├── app.go            # UI root window, tabs, notification dispatcher
│   │   ├── tab_dispatch.go   # Magnet link inspection and send view
│   │   ├── tab_profiles.go   # Client profile management & preset editor
│   │   ├── tab_history.go    # Dispatch history list & resend
│   │   └── tab_settings.go   # LaunchServices registration & preferences
│   └── urlhandler/           # macOS URL scheme handler
│       ├── handler.go        # Unified Go URL subscription channel
│       ├── darwin_url.h      # C header for Cocoa event listener
│       ├── darwin_url.m      # Objective-C NSAppleEventManager implementation
│       ├── url_darwin.go     # Cgo bridge to handleDarwinURL
│       ├── url_darwin_test.go
│       └── url_other.go      # Non-Darwin stub
├── scripts/
│   └── gen_icon.go           # High-resolution 512x512 app icon generator
└── assets/
    └── icon.png              # App icon
```

---

## Build & Run with Task

Use `task` (`go-task`):

```bash
# List all available tasks
task --list

# Run tests
task test

# Build binary into bin/torrentclick
task build

# Run directly from source
task run

# Package macOS app bundle with magnet: handler and register with LaunchServices
task package

# Open the packaged TorrentClick.app
task open

# Test macOS URL scheme interception
task test-magnet
```

> **Note on `task` binary:** If `task` invokes Taskwarrior on your machine, run `~/go/bin/task` or alias `task=~/go/bin/task`.

---

## Configuration & Template Examples

User configuration is automatically saved in `~/Library/Application Support/TorrentClick/config.json`.

### 1. Synology Download Station
- **Base URL**: `http://192.168.1.100:5000`
- **Endpoint**: `/webapi/DownloadStation/task.cgi`
- **Method**: `POST`
- **Content-Type**: `application/x-www-form-urlencoded`
- **Auth Type**: `synology_login`
- **Username / Password**: Your DSM credentials
- **Body Template**:
  ```
  api=SYNO.DownloadStation.Task&version=1&method=create&uri={{.MagnetEscaped}}&_sid={{.SessionID}}
  ```

### 2. qBittorrent Web UI
- **Base URL**: `http://localhost:8080`
- **Endpoint**: `/api/v2/torrents/add`
- **Method**: `POST`
- **Content-Type**: `application/x-www-form-urlencoded`
- **Auth Type**: `qbittorrent_login`
- **Username / Password**: Your Web UI credentials (default: `admin` / `adminadmin`)
- **Body Template**:
  ```
  urls={{.Magnet}}&savepath={{index .Config "savepath"}}&category={{index .Config "category"}}
  ```

### 3. Transmission
- **Base URL**: `http://localhost:9091`
- **Endpoint**: `/transmission/rpc`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Auth Type**: `transmission_session`
- **Username / Password**: Optional (if RPC auth is enabled)
- **Body Template**:
  ```json
  {"method": "torrent-add", "arguments": {"filename": "{{.Magnet}}"}}
  ```

### 4. Deluge Web UI
- **Base URL**: `http://localhost:8112`
- **Endpoint**: `/json`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Body Template**:
  ```json
  {"method": "core.add_torrent_magnet", "params": ["{{.Magnet}}", {}], "id": 1}
  ```

### 5. Custom Webhook
- **Base URL**: `https://api.my-server.com`
- **Endpoint**: `/api/v1/torrents`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Auth Type**: `bearer`
- **Token**: `secret_api_key`
- **Body Template**:
  ```json
  {
    "magnet": "{{.Magnet}}",
    "name": "{{.DisplayName}}",
    "hash": "{{.InfoHash}}",
    "trackers": {{json .Trackers}}
  }
  ```

---

## macOS Magnet Handler Registration

When you run `task package`:
1. `fyne package` generates `TorrentClick.app`.
2. `PlistBuddy` injects the `CFBundleURLTypes` entry for `magnet:` into `TorrentClick.app/Contents/Info.plist`:
   ```xml
   <key>CFBundleURLTypes</key>
   <array>
       <dict>
           <key>CFBundleURLName</key>
           <string>com.torrentclick.app.magnet</string>
           <key>CFBundleURLSchemes</key>
           <array>
               <string>magnet</string>
           </array>
       </dict>
   </array>
   ```
3. `lsregister -f TorrentClick.app` registers the bundle with macOS LaunchServices.
4. When you click any `magnet:` link in a web browser or run `open "magnet:?xt=..."`, macOS launches or brings `TorrentClick.app` to the foreground and forwards the URL directly.
