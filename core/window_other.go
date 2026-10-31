//go:build !darwin || !cgo

package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
	if w.app != nil && w.app.config.AppID != "" {
		args = append(args, "--class="+w.app.config.AppID)
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

func (w *Window) runLoop(ctx context.Context) {
	select {
	case <-w.closed:
	case <-ctx.Done():
	}
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

func (w *Window) setPlatformTitle(title string)     {}
func (w *Window) setPlatformSize(width, height int) {}
func (w *Window) setPlatformPosition(x, y int)      {}
func (w *Window) minimizePlatform()                 {}
func (w *Window) maximizePlatform()                 {}
func (w *Window) restorePlatform()                  {}
func (w *Window) setPlatformFullscreen(fs bool)     {}
func (w *Window) evalPlatform(js string)            {}
