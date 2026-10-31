package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// WindowState represents the visual state of a desktop window.
type WindowState string

const (
	WindowStateNormal     WindowState = "normal"
	WindowStateMinimized  WindowState = "minimized"
	WindowStateMaximized  WindowState = "maximized"
	WindowStateFullscreen WindowState = "fullscreen"
)

// WindowConfig holds desktop window appearance and geometry configuration.
type WindowConfig struct {
	Title       string
	Width       int
	Height      int
	MinWidth    int
	MinHeight   int
	X           int
	Y           int
	Center      bool
	Fullscreen  bool
	Maximized   bool
	Frameless   bool
	AlwaysOnTop bool
	Headless    bool
	BrowserPath string
	CustomFlags []string
}

// Window represents and controls the active desktop application window.
type Window struct {
	mu        sync.RWMutex
	app       *App
	cfg       WindowConfig
	state     WindowState
	cmd       *exec.Cmd
	profile   string
	cdp       *cdpClient
	sessionID string
	targetID  string
	windowID  int
	closed    chan struct{}
	closeOnce sync.Once
}

func newWindow(app *App, cfg WindowConfig) *Window {
	if cfg.Width <= 0 {
		cfg.Width = 1080
	}
	if cfg.Height <= 0 {
		cfg.Height = 720
	}
	if cfg.MinWidth == 0 {
		if cfg.Width > 0 && cfg.Width < 680 {
			cfg.MinWidth = cfg.Width
		} else {
			cfg.MinWidth = 680
		}
	} else if cfg.MinWidth < 0 {
		cfg.MinWidth = 0
	}
	if cfg.MinHeight == 0 {
		if cfg.Height > 0 && cfg.Height < 480 {
			cfg.MinHeight = cfg.Height
		} else {
			cfg.MinHeight = 480
		}
	} else if cfg.MinHeight < 0 {
		cfg.MinHeight = 0
	}
	if cfg.MinWidth > 0 && cfg.Width < cfg.MinWidth {
		cfg.Width = cfg.MinWidth
	}
	if cfg.MinHeight > 0 && cfg.Height < cfg.MinHeight {
		cfg.Height = cfg.MinHeight
	}
	if cfg.Title == "" && app != nil {
		cfg.Title = app.config.Title
	}
	state := WindowStateNormal
	if cfg.Fullscreen {
		state = WindowStateFullscreen
	} else if cfg.Maximized {
		state = WindowStateMaximized
	}
	return &Window{
		app:    app,
		cfg:    cfg,
		state:  state,
		closed: make(chan struct{}),
	}
}

// HasCDP reports whether an active Chrome DevTools Protocol connection is attached.
func (w *Window) HasCDP() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cdp != nil && !w.cdp.closed.Load()
}

// Title returns the current window title.
func (w *Window) Title() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cfg.Title
}

// Size returns the current configured window width and height.
func (w *Window) Size() (int, int) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cfg.Width, w.cfg.Height
}

// Position returns the current configured window X and Y coordinates.
func (w *Window) Position() (int, int) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cfg.X, w.cfg.Y
}

// State returns the current WindowState ("normal", "minimized", "maximized", "fullscreen").
func (w *Window) State() WindowState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state
}

// SetTitle updates the desktop window title dynamically.
func (w *Window) SetTitle(title string) error {
	w.mu.Lock()
	w.cfg.Title = title
	w.mu.Unlock()

	w.setPlatformTitle(title)

	if w.HasCDP() {
		_, _ = w.Eval(fmt.Sprintf("document.title = %q", title))
	}
	if w.app != nil {
		w.app.Emit("brisa:window:title", Map{"title": title})
	}
	return nil
}

// SetSize resizes the desktop window to width x height pixels.
func (w *Window) SetSize(width, height int) error {
	if width <= 0 || height <= 0 {
		return errors.New("brisa window: width and height must be positive")
	}
	w.mu.Lock()
	if w.cfg.MinWidth > 0 && width < w.cfg.MinWidth {
		width = w.cfg.MinWidth
	}
	if w.cfg.MinHeight > 0 && height < w.cfg.MinHeight {
		height = w.cfg.MinHeight
	}
	w.cfg.Width = width
	w.cfg.Height = height
	w.state = WindowStateNormal
	w.mu.Unlock()

	w.setPlatformSize(width, height)

	if err := w.setCDPBounds(map[string]any{
		"windowState": "normal",
		"width":       width,
		"height":      height,
	}); err == nil {
		return nil
	}
	if w.app != nil {
		w.app.Emit("brisa:window:size", Map{"width": width, "height": height})
	}
	return nil
}

// SetMinSize updates the minimum allowable dimensions of the desktop window.
func (w *Window) SetMinSize(width, height int) error {
	if width < 0 || height < 0 {
		return errors.New("brisa window: min width and min height must be non-negative")
	}
	w.mu.Lock()
	w.cfg.MinWidth = width
	w.cfg.MinHeight = height
	w.mu.Unlock()

	w.setPlatformMinSize(width, height)
	return nil
}

// SetPosition moves the desktop window to screen coordinates (x, y).
func (w *Window) SetPosition(x, y int) error {
	w.mu.Lock()
	w.cfg.X = x
	w.cfg.Y = y
	w.mu.Unlock()

	w.setPlatformPosition(x, y)

	if err := w.setCDPBounds(map[string]any{
		"windowState": "normal",
		"left":        x,
		"top":         y,
	}); err == nil {
		return nil
	}
	if w.app != nil {
		w.app.Emit("brisa:window:position", Map{"x": x, "y": y})
	}
	return nil
}

// Minimize minimizes the window to the system taskbar/dock.
func (w *Window) Minimize() error {
	w.mu.Lock()
	w.state = WindowStateMinimized
	w.mu.Unlock()

	w.minimizePlatform()

	if err := w.setCDPBounds(map[string]any{"windowState": "minimized"}); err == nil {
		return nil
	}
	if w.app != nil {
		w.app.Emit("brisa:window:state", Map{"state": string(WindowStateMinimized)})
	}
	return nil
}

// Maximize maximizes the window on the current screen.
func (w *Window) Maximize() error {
	w.mu.Lock()
	w.state = WindowStateMaximized
	w.mu.Unlock()

	w.maximizePlatform()

	if err := w.setCDPBounds(map[string]any{"windowState": "maximized"}); err == nil {
		return nil
	}
	if w.app != nil {
		w.app.Emit("brisa:window:state", Map{"state": string(WindowStateMaximized)})
	}
	return nil
}

// Restore restores the window from minimized/maximized/fullscreen back to normal state.
func (w *Window) Restore() error {
	w.mu.Lock()
	w.state = WindowStateNormal
	w.mu.Unlock()

	w.restorePlatform()

	if err := w.setCDPBounds(map[string]any{"windowState": "normal"}); err == nil {
		return nil
	}
	if w.app != nil {
		w.app.Emit("brisa:window:state", Map{"state": string(WindowStateNormal)})
	}
	return nil
}

// SetFullscreen toggles fullscreen mode on the desktop window.
func (w *Window) SetFullscreen(fullscreen bool) error {
	state := WindowStateNormal
	if fullscreen {
		state = WindowStateFullscreen
	}
	w.mu.Lock()
	w.cfg.Fullscreen = fullscreen
	w.state = state
	w.mu.Unlock()

	w.setPlatformFullscreen(fullscreen)

	if err := w.setCDPBounds(map[string]any{"windowState": string(state)}); err == nil {
		return nil
	}
	if w.app != nil {
		w.app.Emit("brisa:window:state", Map{"state": string(state)})
	}
	return nil
}

// Eval evaluates a JavaScript expression in the desktop window context and returns the result.
func (w *Window) Eval(expression string) (any, error) {
	w.evalPlatform(expression)
	w.mu.RLock()
	cdp := w.cdp
	sessionID := w.sessionID
	w.mu.RUnlock()

	if cdp != nil && !cdp.closed.Load() && sessionID != "" {
		raw, err := cdp.call("Runtime.evaluate", map[string]any{
			"expression":    expression,
			"returnByValue": true,
			"awaitPromise":  true,
		}, sessionID)
		if err != nil {
			return nil, err
		}
		var evalRes struct {
			Result struct {
				Value any `json:"value"`
			} `json:"result"`
			ExceptionDetails *struct {
				Text string `json:"text"`
			} `json:"exceptionDetails"`
		}
		if err := json.Unmarshal(raw, &evalRes); err != nil {
			return nil, err
		}
		if evalRes.ExceptionDetails != nil {
			return nil, fmt.Errorf("js evaluation error: %s", evalRes.ExceptionDetails.Text)
		}
		return evalRes.Result.Value, nil
	}

	if w.app != nil {
		w.app.Emit("brisa:window:eval", Map{"expression": expression})
		return nil, nil
	}
	return nil, errors.New("brisa window: no active window runtime attached")
}

// Close closes the desktop window and terminates the browser app process if running.
func (w *Window) Close() error {
	w.closeOnce.Do(func() {
		w.mu.Lock()
		cdp := w.cdp
		cmd := w.cmd
		profile := w.profile
		w.mu.Unlock()

		if w.app != nil {
			w.app.Emit("brisa:window:close", Map{})
		}
		if cdp != nil {
			_, _ = cdp.call("Browser.close", nil, "")
			cdp.close()
		}
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		if profile != "" {
			_ = os.RemoveAll(profile)
		}
		close(w.closed)
	})
	return nil
}

// Done returns a channel that is closed when the desktop window is closed.
func (w *Window) Done() <-chan struct{} {
	return w.closed
}

func (w *Window) setCDPBounds(bounds map[string]any) error {
	w.mu.RLock()
	cdp := w.cdp
	winID := w.windowID
	w.mu.RUnlock()

	if cdp == nil || cdp.closed.Load() || winID == 0 {
		return errors.New("cdp window ID unavailable")
	}
	_, err := cdp.call("Browser.setWindowBounds", map[string]any{
		"windowId": winID,
		"bounds":   bounds,
	}, "")
	return err
}

// LocateBrowser searches standard OS paths for an installed Chromium-based browser capable of app-mode rendering.
func LocateBrowser() string {
	var candidates []string
	switch runtime.GOOS {
	case "linux":
		candidates = []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
			"brave-browser",
			"microsoft-edge",
			"microsoft-edge-stable",
			"vivaldi",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Vivaldi.app/Contents/MacOS/Vivaldi",
		}
	case "windows":
		var prefixes []string
		for _, env := range []string{"LOCALAPPDATA", "PROGRAMFILES", "PROGRAMFILES(X86)"} {
			if v := os.Getenv(env); v != "" {
				prefixes = append(prefixes, v)
			}
		}
		suffixes := []string{
			`Google\Chrome\Application\chrome.exe`,
			`Microsoft\Edge\Application\msedge.exe`,
			`BraveSoftware\Brave-Browser\Application\brave.exe`,
			`Chromium\Application\chrome.exe`,
		}
		for _, p := range prefixes {
			for _, s := range suffixes {
				candidates = append(candidates, filepath.Join(p, s))
			}
		}
	}

	for _, c := range candidates {
		if filepath.IsAbs(c) {
			if info, err := os.Stat(c); err == nil && !info.IsDir() {
				return c
			}
		} else if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}
