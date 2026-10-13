# TorrentClick

A macOS desktop application written in **Go** and **Fyne** that registers as the system handler for torrent `magnet:` links and double-clicked `.torrent` files. When a magnet link is clicked anywhere on 
macOS (Safari, Chrome, Firefox, Slack, etc.) or a `.torrent` file is double-clicked in Finder, TorrentClick intercepts it and dispatches a configurable REST API call using **Go text/template** and 
authentication to your remote or local torrent client.

> [!WARNING]
> **Disclaimer**: This entire project was basically my test to try out vibecoding without even having a detailed look at the code. I had the exact requirement to automatically 
start downloads when clicking magnet links on a website, so I combined the necessary with the useful. It has only been tested to work with Synology DSM.
>
> I might improve it by hand in the future—especially the UI, which is ugly and not well integrated into the macOS desktop—yet it serves my needs for now. Pull requests are 
appreciated should anyone fix a template or a bug in the application itself!

---

## Features

- **macOS Magnet & Torrent URL Schemes (`magnet:`, `torrentclick:`, `torrent:`) & File Association**:
  - Registers `CFBundleURLTypes` for `magnet:`, `torrentclick:`, and `torrent:` URL schemes and `CFBundleDocumentTypes` for `.torrent` files in macOS `Info.plist`.
  - Handles **cold boots** and **warm starts** via Cocoa Apple Events (`kInternetEventClass` / `kAEGetURL` and `kCoreEventClass` / `kAEOpenDocuments`).
  - Supports custom `torrentclick://add?name=...&data=<base64>` and `torrent://` schemes with in-memory Base64 decoding.
- **Safari Web Extension & Zero-Disk-Write Interceptor**:
  - Automatically intercepts clicks on `.torrent` download links in Safari.
  - Fetches `.torrent` file directly into **browser RAM** (preserving authenticated tracker cookies) and forwards it via `torrentclick://` without ever saving to `~/Downloads` or touching the hard disk!
  - Hold **Option (⌥)** while clicking any torrent link to bypass interception and download to disk normally.
  - Includes a standalone **Userscript** (`extension/torrentclick.user.js`) for 1-click use in Tampermonkey or Userscripts for Safari.
- **Configurable macOS Dock & Menu Bar Tray Display**:
  - Configure whether TorrentClick displays in the macOS **Dock only**, **Menu Bar Tray only**, or **Both** in the Settings tab.
  - Dynamically changes `NSApplicationActivationPolicy` (Regular vs. Accessory) and `NSStatusItem` visibility at runtime without needing an app restart.
  - Clicking the Dock icon reopens the TorrentClick window when hidden.
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
  - **Synology Download Station**: Automated DSM WebAPI authentication (`SYNO.API.Auth` login with `_sid` session token retrieval and error code 105 auto-relogin) + 
    `SYNO.DownloadStation.Task` create task call.
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
  - **Settings Tab**: Auto-send on click toggle, desktop notifications, auto-hide window after dispatch, optional auto-deletion of `.torrent` files after successful download task creation, and LaunchServices registration tool.
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
│   ├── magnet/               # Magnet URI & .torrent bencode parser
│   │   ├── magnet.go
│   │   ├── magnet_test.go
│   │   ├── torrent.go        # Bencode decoder, SHA-1 info-hash calculation & payload builder
│   │   └── torrent_test.go
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

## macOS URL Scheme Registrations (`magnet:`, `torrentclick:`, `torrent:`)

When you run `task package`:
1. `fyne package` generates `TorrentClick.app`.
2. `plutil` injects the `CFBundleURLTypes` entry for `magnet:`, `torrentclick:`, and `torrent:` into `TorrentClick.app/Contents/Info.plist`:
   ```xml
   <key>CFBundleURLTypes</key>
   <array>
       <dict>
           <key>CFBundleURLName</key>
           <string>com.torrentclick.app.schemes</string>
           <key>CFBundleURLSchemes</key>
           <array>
               <string>magnet</string>
               <string>torrent</string>
               <string>torrentclick</string>
           </array>
       </dict>
   </array>
   ```
3. `lsregister -f TorrentClick.app` registers the bundle with macOS LaunchServices.
4. When you click any `magnet:` link or a `torrentclick://` link in Safari, macOS forwards the URL directly to `TorrentClick.app`.

---

## Safari Companion Extension & Userscript

TorrentClick includes a Safari companion that catches `.torrent` download links in memory, encodes the raw payload in Base64, and hands it off directly to `torrentclick://` without writing the file to disk.

### Option 1: 1-Click Userscript (Fastest)
Install [Userscripts for Safari](https://github.com/quoid/userscripts) (free on the Mac App Store) or Tampermonkey:
1. Open the Userscripts extension in Safari.
2. Add a new script and paste the contents of `extension/torrentclick.user.js`.
3. Save. Any `.torrent` link clicked in Safari will now stream directly to TorrentClick in RAM!

### Option 2: Native Safari Web Extension
The project includes a ready-to-build Xcode project created via `safari-web-extension-converter`:
1. Open `safari-extension-project/TorrentClickExtension/TorrentClickExtension.xcodeproj` in Xcode.
2. In Safari Preferences:
   - Enable **Develop → Allow Unsigned Extensions** (or sign with your Apple ID).
   - In Safari **Settings → Extensions**, enable **TorrentClickExtension**.
3. Hold **Option (⌥)** when clicking any torrent link if you ever want to bypass interception and download to disk normally.
