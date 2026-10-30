<div align="center">
	<img src="logo.svg" width="90px" alt="Brisa Framework">
</div>

<h1 align="center">Brisa Framework 🌬️</h1>

<div align="center">

[![Go Version](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Pattern](https://img.shields.io/badge/pattern-REPR%20IPC-blue)](https://deviq.com/design-patterns/repr-design-pattern/)
[![UI Kit](https://img.shields.io/badge/UI-GNOME%2050%20%2F%20GTK4%20Libadwaita-3584e4)](core/adwaita.go)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-0-success)](go.mod)
[![CGO](https://img.shields.io/badge/CGO-disabled%20(100%25%20Pure%20Go)-purple)](go.mod)

</div>

**Brisa** is an ultra-lightweight desktop application micro-framework for Go with **zero external dependencies** and **100% pure Go compilation (`CGO_ENABLED=0`)**.

Built by the creator of [Gota 💧](https://github.com/carlosxfelipe/gota), Brisa brings the **REPR Design Pattern (Request-Endpoint-Response)** to desktop IPC and ships with a built-in **GNOME 50 / GTK4 Libadwaita UI Kit** (`/brisa/adwaita.css`) that synchronizes instantaneously with your operating system's Light and Dark mode—without bundling 150MB+ of Electron/Node.js or requiring CGO/GTK C headers.

---

## ✨ Key Features

- **Zero External Dependencies & `CGO_ENABLED=0`**: Built 100% on the Go standard library (`net/http`, `embed`, `crypto/aes`, `crypto/sha1`, `reflect`). Cross-compiles cleanly with standard `go build`.
- **GNOME 50 / GTK4 Libadwaita UI Kit (`/brisa/adwaita.css`)**: Automatically injected into your app with native-grade `AdwNavigationSplitView`, `AdwPreferencesGroup`, `.boxed-list`, `AdwActionRow`, `AdwEntryRow`, `AdwComboRow`, and uniform 34px GTK4 buttons (`.gtk-btn`, `.suggested-action`, `.destructive-action`, `.flat`, `.circular`, `.pill`).
- **OS-Governed Light & Dark Mode (<1ms Latency)**: Strictly governed by the operating system—synchronizes in real time with GNOME (`stdbuf -oL gsettings monitor org.gnome.desktop.interface color-scheme`), macOS (`AppleInterfaceStyle`), Windows Registry, and `@media (prefers-color-scheme)`.
- **REPR Desktop IPC Bridge**: Typed command registration (`brisa.Bind`, `brisa.BindEndpoint`, `brisa.BindService`) with automatic JSON payload binding and declarative struct tag validation (`validate:"required,min=3,oneof=..."`).
- **Auto-Generated JS & TypeScript SDK**: Serves `/brisa/sdk.js` at runtime (automatically injected into `index.html`) exposing `brisa.call.<group>.<command>()`, `brisa.invoke()`, `brisa.batch()`, and generates `.d.ts` definitions at `/brisa/types.d.ts`.
- **Built-in Interactive IPC Inspector**: Inspired by GNOME D-Spy and Gota's Scalar Docs, Brisa includes a zero-dependency interactive Desktop IPC Playground & Live SSE Event Monitor at `/brisa/inspector`.
- **Encrypted Local Store**: Built-in persistent key-value and JSON configuration store (`ctx.Store()`) backed by atomic file writes and optional **AES-256-GCM encryption** at rest.
- **Native OS Integration**: Cross-platform Window control via a pure-Go Chrome DevTools Protocol (CDP) WebSocket client (resize, move, minimize, maximize, fullscreen, title, JS eval), File/Directory/Message Dialogs, Clipboard, and Desktop Notifications.

> 📖 Looking for in-depth architecture patterns, GTK4 CSS classes, validation rules, and AI coding guidance? See [llms.txt](llms.txt).

---

## ⚡ Architecture: Why Brisa Instead of Electron?

Traditional desktop web frameworks like **Electron** bundle an entire copy of Chromium and Node.js inside every application binary (~150 MB to 250 MB per app). Other Go desktop frameworks rely on **CGO** (`webkit2gtk-4.1` C headers, `gcc`, complex cross-compilation toolchains) and heavy external dependency trees.

**Brisa takes the Deno Desktop (WebUI / Gluon) and Lorca approach—engineered 100% on the Go standard library:**

1. **No Bundled Browser Engine (~5–8 MB Single Binary)**: Your compiled binary contains only your Go code and embedded UI assets (`embed.FS`).
2. **Reuses the OS Browser Engine in Chromeless App-Mode**: At startup, `brisa.LocateBrowser()` automatically finds an existing Chromium-based engine already installed on the user's machine (Chrome, Chromium, Brave, Edge, or Vivaldi across Linux, macOS, and Windows) and launches an isolated, chromeless desktop window (`--app`) with extensions, sync, translation popups, and background networking stripped out.
3. **Pure-Go Chrome DevTools Protocol (CDP) Bridge**: Instead of CGO bindings, Brisa connects to the window using a built-in, zero-dependency RFC 6455 WebSocket client (`core/cdp.go`) to control window geometry (`SetSize`, `SetPosition`, `Minimize`, `Maximize`, `SetFullscreen`, `SetTitle`, `Eval`, `Close`) natively from Go.

| Feature | Electron | CGO WebView Frameworks | **Brisa 🌬️** |
| :--- | :--- | :--- | :--- |
| **Binary Size** | ~150 MB – 250 MB | ~15 MB – 40 MB | **~5 MB – 8 MB** |
| **External `go.mod` Deps** | N/A (Node/npm) | Dozens of packages | **0 (100% Go stdlib)** |
| **Requires CGO / C Compiler** | N/A | Yes (`gcc`, GTK/WebKit headers) | **No (`CGO_ENABLED=0`)** |
| **Browser Engine** | Bundled per app | OS WebKit / WebView2 (via C FFI) | **Reuses OS Browser (`--app` + CDP)** |
| **IPC Architecture** | Untyped channels | Custom / reflection bindings | **Typed REPR + Auto Validation** |
| **Built-in UI Kit** | None | None | **GNOME 50 / GTK4 Libadwaita** |

---

## 📦 Installation

Initialize your Go module:

```bash
go mod init mydesktopapp
```

Install Brisa:

```bash
go get github.com/carlosxfelipe/brisa
```

Requires Go 1.26 or newer.

---

## 🚀 Quickstart (REPR Desktop Pattern + GTK4 / Libadwaita UI)

Create a `main.go` file:

```go
package main

import (
	"log"

	"github.com/carlosxfelipe/brisa"
)

// 1. Request DTO with declarative validation
type GreetRequest struct {
	Name string `json:"name" doc:"User name" example:"Carlos" validate:"required,min=2"`
}

// 2. Response DTO
type GreetResponse struct {
	Message string `json:"message" example:"Hello, Carlos! Welcome to Brisa 🌬️"`
}

func main() {
	app := brisa.New(brisa.Config{
		AppID: "com.example.hello",
		Title: "Hello Brisa",
		Theme: brisa.ThemeUbuntu, // or brisa.ThemeAdwaita (default GNOME Libadwaita)
		Window: brisa.WindowConfig{
			Width:  900,
			Height: 580,
		},
		IndexHTML: `<!DOCTYPE html>
<html lang="en" translate="no">
<body>
  <main class="adw-page">
    <div class="adw-clamp">
      <div class="adw-group">
        <div class="adw-group-title">Welcome (GTK4 / Libadwaita)</div>
        <div class="boxed-list">
          <div class="adw-row">
            <div class="adw-row-text">
              <div class="adw-row-subtitle">Your name</div>
              <input id="name" class="adw-entry-input" placeholder="Enter your name..." />
            </div>
            <div class="adw-row-suffix">
              <button class="gtk-btn suggested-action" onclick="sayHello()">Greet</button>
            </div>
          </div>
          <div class="adw-row">
            <div class="adw-row-text">
              <div class="adw-row-title" id="out">Waiting for IPC call...</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </main>
  <script>
    async function sayHello() {
      const name = document.getElementById("name").value;
      const res = await brisa.call.users.greet({ name });
      document.getElementById("out").textContent = res.message;
    }
  </script>
</body>
</html>`,
	})

	// 3. REPR Desktop Command
	brisa.Bind(app, "users.greet", func(ctx *brisa.Context, req GreetRequest) (GreetResponse, error) {
		return GreetResponse{
			Message: "Hello, " + req.Name + "! Welcome to Brisa 🌬️",
		}, nil
	}).Summary("Greet the desktop user")

	log.Fatal(app.Run())
}
```

Run your desktop application:

```bash
go run main.go
```

---

## 🏛️ Three Ways to Bind Go Logic to the Desktop UI

### 1. Functional REPR Command (`brisa.Bind`)

```go
brisa.Bind(app, "tasks.create", func(ctx *brisa.Context, req CreateTaskRequest) (Task, error) {
	// req is already bound and validated!
	return Task{ID: "tsk_1", Title: req.Title}, nil
}).Summary("Create task").Tags("Tasks")
```

### 2. Struct-Based REPR Endpoint (`brisa.BindEndpoint`)

```go
type CreateTaskEndpoint struct{}

func (e *CreateTaskEndpoint) Handle(ctx *brisa.Context, req CreateTaskRequest) (Task, error) {
	return Task{ID: "tsk_1", Title: req.Title}, nil
}

brisa.BindEndpoint(app, "tasks.create", &CreateTaskEndpoint{})
```

### 3. Automatic Service Reflection (`brisa.BindService`)

```go
type SystemService struct{}

func (s *SystemService) Info() (SystemInfoResponse, error) {
	return SystemInfoResponse{OS: runtime.GOOS}, nil
}

// Exposes "system.info" in Go and `await brisa.call.system.info()` in JS!
brisa.BindService(app, "system", &SystemService{})
```

---

## 🎨 GNOME Adwaita & Ubuntu Yaru UI Kit (`/brisa/adwaita.css`)

By default, Brisa automatically injects `/brisa/adwaita.css` and `/brisa/sdk.js` into your `index.html`. All HTML layout and component classes (`.adw-split-view`, `.boxed-list`, `.adw-row`, `.gtk-btn`, `.gtk-dropdown`) work identically across both built-in themes—just set `Theme` in `brisa.Config`:

```go
app := brisa.New(brisa.Config{
	Title: "My Desktop App",
	// Choose your desktop UI design system in Go code:
	// - brisa.ThemeAdwaita: GNOME 50 / GTK4 Libadwaita (Adwaita Blue #3584e4, Cantarell, 9px/12px radii)
	// - brisa.ThemeUbuntu:  Canonical Ubuntu Yaru GTK4 (Ubuntu Orange #E95420, Ubuntu Sans, 6px/8px radii)
	Theme: brisa.ThemeUbuntu,
})
```

- **OS-Governed Light & Dark Mode**: Light and Dark modes are strictly controlled by the operating system (`org.gnome.desktop.interface color-scheme` on GNOME/Ubuntu, `AppleInterfaceStyle` on macOS, `AppsUseLightTheme` on Windows), repainting in the exact same frame as the native window headerbar.
- **Custom CSS Mode**: Set `DisableAdwaitaCSS: true` in `brisa.Config` if you prefer to bring your own stylesheet from scratch.

---

## 🔐 Encrypted Local Persistent Store

Brisa includes a zero-dependency persistent key-value and JSON store saved in the user's OS config directory (`os.UserConfigDir()/<AppID>/store.json`), with optional **AES-256-GCM encryption** at rest:

```go
app := brisa.New(brisa.Config{
	AppID: "com.mycompany.vault",
	Store: brisa.StoreOptions{
		EncryptionKey: "my-32-byte-secret-passphrase",
	},
})

// Inside any command handler:
_ = ctx.Store().Set("user.preferences", map[string]any{"notifications": true})
```

---

## 🧹 Code Formatting

```bash
go fmt ./...
```

## 🧪 Running Tests

```bash
CGO_ENABLED=0 go test ./... -v
```

## 🏃 Running the Studio Showcase

```bash
go run examples/main.go
```

---

## 📄 License

MIT © [Carlos Felipe](https://github.com/carlosxfelipe)
