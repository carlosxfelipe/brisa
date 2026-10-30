package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
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
	w.cfg.Width = width
	w.cfg.Height = height
	w.state = WindowStateNormal
	w.mu.Unlock()

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

// SetPosition moves the desktop window to screen coordinates (x, y).
func (w *Window) SetPosition(x, y int) error {
	w.mu.Lock()
	w.cfg.X = x
	w.cfg.Y = y
	w.mu.Unlock()

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

func (w *Window) launch(targetURL string) error {
	if w.cfg.Headless {
		return nil
	}

	bin := w.cfg.BrowserPath
	if bin == "" {
		bin = LocateBrowser()
	}
	if bin == "" {
		return errors.New("brisa window: no compatible Chromium/Chrome/Edge/Brave browser found on system; install one or set WindowConfig.BrowserPath")
	}

	profileDir, err := os.MkdirTemp("", "brisa-profile-*")
	if err != nil {
		return fmt.Errorf("brisa window profile dir: %w", err)
	}

	args := []string{
		"--app=" + targetURL,
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-translate",
		"--disable-features=Translate,TranslateUI,BackForwardCache,AcceptCHFrame,MediaRouter,OptimizationHints",
		"--disable-extensions",
		"--disable-sync",
		"--disable-background-networking",
		"--remote-debugging-port=0",
		"--remote-allow-origins=*",
		fmt.Sprintf("--window-size=%d,%d", w.cfg.Width, w.cfg.Height),
	}
	if w.cfg.X != 0 || w.cfg.Y != 0 {
		args = append(args, fmt.Sprintf("--window-position=%d,%d", w.cfg.X, w.cfg.Y))
	}
	if w.cfg.Fullscreen {
		args = append(args, "--start-fullscreen")
	} else if w.cfg.Maximized {
		args = append(args, "--start-maximized")
	}
	args = append(args, w.cfg.CustomFlags...)

	cmd := exec.Command(bin, args...)
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(profileDir)
		return fmt.Errorf("brisa window start %q: %w", bin, err)
	}

	w.mu.Lock()
	w.cmd = cmd
	w.profile = profileDir
	w.mu.Unlock()

	go w.attachCDP(profileDir, targetURL)

	go func() {
		_ = cmd.Wait()
		_ = w.Close()
	}()

	return nil
}

func (w *Window) attachCDP(profileDir, targetURL string) {
	activePortFile := filepath.Join(profileDir, "DevToolsActivePort")
	deadline := time.Now().Add(4 * time.Second)
	var raw []byte
	for time.Now().Before(deadline) {
		select {
		case <-w.closed:
			return
		default:
		}
		if b, err := os.ReadFile(activePortFile); err == nil && len(b) > 0 {
			lines := strings.Split(strings.TrimSpace(string(b)), "\n")
			if len(lines) >= 2 {
				raw = b
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
	}
	if len(raw) == 0 {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	port := strings.TrimSpace(lines[0])
	wsPath := strings.TrimSpace(lines[1])
	wsURL := fmt.Sprintf("ws://127.0.0.1:%s%s", port, wsPath)

	client, err := dialCDP(wsURL, 2*time.Second)
	if err != nil {
		return
	}

	w.mu.Lock()
	w.cdp = client
	w.mu.Unlock()

	// Find the page target and attach session + windowId
	res, err := client.call("Target.getTargets", nil, "")
	if err != nil {
		return
	}
	var targets struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(res, &targets); err != nil {
		return
	}
	var pageTargetID string
	for _, t := range targets.TargetInfos {
		if t.Type == "page" {
			pageTargetID = t.TargetID
			if strings.HasPrefix(t.URL, targetURL) {
				break
			}
		}
	}
	if pageTargetID == "" {
		return
	}

	attachRes, err := client.call("Target.attachToTarget", map[string]any{
		"targetId": pageTargetID,
		"flatten":  true,
	}, "")
	if err == nil {
		var attached struct {
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(attachRes, &attached) == nil {
			w.mu.Lock()
			w.targetID = pageTargetID
			w.sessionID = attached.SessionID
			w.mu.Unlock()
		}
	}

	winRes, err := client.call("Browser.getWindowForTarget", map[string]any{
		"targetId": pageTargetID,
	}, "")
	if err == nil {
		var winInfo struct {
			WindowID int `json:"windowId"`
		}
		if json.Unmarshal(winRes, &winInfo) == nil {
			w.mu.Lock()
			w.windowID = winInfo.WindowID
			w.mu.Unlock()
		}
	}
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
