package core

import (
	"bufio"
	"context"
	"os/exec"
	"runtime"
	"strings"
)

// Theme represents the desktop UI design system flavor.
type Theme string

// NormalizeTheme normalizes a Theme value, defaulting to ThemeAdwaita.
func NormalizeTheme(theme Theme) Theme {
	switch strings.ToLower(strings.TrimSpace(string(theme))) {
	case "ubuntu", "yaru":
		return ThemeUbuntu
	case "plasma", "kde", "breeze":
		return ThemePlasma
	case "macos", "darwin", "aqua", "mac":
		return ThemeMacOS
	case "windows", "win11", "fluent", "winui":
		return ThemeWindows
	default:
		return ThemeAdwaita
	}
}

// DetectSystemTheme queries the operating system and returns "dark" or "light".
func DetectSystemTheme() string {
	switch runtime.GOOS {
	case "linux":
		if path, err := exec.LookPath("gsettings"); err == nil {
			out, err := exec.Command(path, "get", "org.gnome.desktop.interface", "color-scheme").Output()
			if err == nil {
				return parseGnomeColorScheme(string(out))
			}
		}
	case "darwin":
		if path, err := exec.LookPath("defaults"); err == nil {
			out, err := exec.Command(path, "read", "-g", "AppleInterfaceStyle").Output()
			if err == nil && strings.Contains(strings.ToLower(string(out)), "dark") {
				return "dark"
			}
			return "light"
		}
	case "windows":
		if path, err := exec.LookPath("reg"); err == nil {
			out, err := exec.Command(path, "query", `HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "/v", "AppsUseLightTheme").Output()
			if err == nil && strings.Contains(string(out), "0x0") {
				return "dark"
			}
			return "light"
		}
	}
	return "dark"
}

func parseGnomeColorScheme(raw string) string {
	val := strings.ToLower(strings.TrimSpace(raw))
	if strings.Contains(val, "prefer-dark") {
		return "dark"
	}
	return "light"
}

func (a *App) startSystemThemeWatcher(ctx context.Context) {
	if runtime.GOOS != "linux" {
		return
	}
	gsettingsPath, err := exec.LookPath("gsettings")
	if err != nil {
		return
	}

	go func() {
		var cmd *exec.Cmd
		// Use stdbuf -oL so GLib flushes every line immediately (<1ms latency) over the pipe.
		if stdbufPath, err := exec.LookPath("stdbuf"); err == nil {
			cmd = exec.CommandContext(ctx, stdbufPath, "-oL", gsettingsPath, "monitor", "org.gnome.desktop.interface", "color-scheme")
		} else {
			cmd = exec.CommandContext(ctx, gsettingsPath, "monitor", "org.gnome.desktop.interface", "color-scheme")
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return
		}
		if err := cmd.Start(); err != nil {
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			next := parseGnomeColorScheme(scanner.Text())
			if a.window != nil && a.window.HasCDP() {
				_, _ = a.window.cdp.call("Emulation.setEmulatedMedia", map[string]any{
					"features": []map[string]string{
						{"name": "prefers-color-scheme", "value": next},
					},
				}, a.window.sessionID)
			}
			a.Emit("brisa:os:theme", Map{"theme": next})
		}
		_ = cmd.Wait()
	}()
}

// RenderThemeCSS returns the built-in desktop CSS kit for the requested Theme.
// Keeping all component classes 100% compatible across themes.
func RenderThemeCSS(theme Theme) string {
	var tokens string
	switch NormalizeTheme(theme) {
	case ThemeUbuntu:
		tokens = ubuntuThemeTokens
	case ThemePlasma:
		tokens = plasmaThemeTokens
	case ThemeMacOS:
		tokens = macosThemeTokens
	case ThemeWindows:
		tokens = windowsThemeTokens
	default:
		tokens = adwaitaThemeTokens
	}
	return tokens + sharedComponentCSS
}

const sharedComponentCSS = `
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body {
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.4;
  background-color: var(--window-bg-color);
  color: var(--window-fg-color);
  -webkit-font-smoothing: antialiased;
}

.adw-split-view {
  display: grid;
  grid-template-columns: 256px minmax(320px, 1fr);
  height: 100vh;
  width: 100vw;
  min-width: 580px;
}

.adw-sidebar {
  background-color: var(--sidebar-bg-color);
  border-right: 1px solid var(--sidebar-border-color);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.adw-sidebar-header {
  height: 46px;
  padding: 0 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--row-divider-color);
  font-weight: 700;
  font-size: 14px;
}

.adw-nav-list {
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  overflow-y: auto;
}

.adw-nav-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 12px;
  height: 38px;
  min-height: 38px;
  border-radius: var(--control-radius);
  color: var(--sidebar-fg-color);
  font-size: 14px;
  font-weight: 500;
  cursor: default;
}

.adw-nav-row:hover { background-color: var(--nav-hover-bg); }
.adw-nav-row.active { background-color: var(--nav-active-bg); font-weight: 600; }

.adw-content-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: var(--window-bg-color);
  overflow: hidden;
}

.adw-toolbar {
  height: 46px;
  padding: 0 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--row-divider-color);
  background-color: var(--window-bg-color);
  flex-shrink: 0;
}

.adw-page {
  flex: 1;
  overflow-y: auto;
  padding: 24px 20px 40px;
}

.adw-clamp {
  max-width: 680px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 26px;
}

.adw-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.adw-group-title { font-size: 14px; font-weight: 700; color: var(--window-fg-color); }
.adw-group-desc { font-size: 12.5px; color: var(--dim-label-color); margin-top: 2px; }

.boxed-list {
  background-color: var(--card-bg-color);
  border-radius: var(--card-radius);
  box-shadow: var(--card-shadow);
  overflow: hidden;
}

.adw-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  min-height: 56px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--row-divider-color);
}

.adw-row:first-child { border-top-left-radius: var(--card-radius); border-top-right-radius: var(--card-radius); }
.adw-row:last-child { border-bottom: none; border-bottom-left-radius: var(--card-radius); border-bottom-right-radius: var(--card-radius); }
.adw-row.activatable:hover { background-color: var(--card-hover-bg); }

.adw-row-text { flex: 1; min-width: 0; }
.adw-row-title { font-size: 14px; font-weight: 500; color: var(--window-fg-color); }
.adw-row-subtitle { font-size: 12px; color: var(--dim-label-color); margin-top: 2px; }
.adw-row-suffix { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }

.adw-entry-input, input:not([type="checkbox"]):not([type="radio"]) {
  background: transparent;
  border: none;
  outline: none;
  color: var(--window-fg-color);
  font-family: inherit;
  font-size: 14px;
}

.gtk-btn, button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  background-color: var(--button-bg);
  color: var(--window-fg-color);
  border: none;
  border-radius: var(--control-radius);
  height: 34px;
  min-height: 34px;
  min-width: 34px;
  padding: 0 14px;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
  flex-shrink: 0;
  cursor: default;
  text-decoration: none;
  outline: none;
}

.gtk-btn:focus-visible, button:focus-visible,
.gtk-dropdown:focus-visible, select:focus-visible {
  outline: none;
  box-shadow: var(--focus-ring);
}

.gtk-btn:hover, button:hover { background-color: var(--button-hover-bg); }
.gtk-btn:active, button:active { background-color: var(--button-active-bg); }

.gtk-btn.suggested-action, button.suggested-action {
  background-color: var(--accent-bg-color);
  color: var(--accent-fg-color);
}
.gtk-btn.suggested-action:hover, button.suggested-action:hover { background-color: var(--accent-hover-bg); }
.gtk-btn.suggested-action:active, button.suggested-action:active { background-color: var(--accent-active-bg); }

.gtk-btn.destructive-action, button.destructive-action {
  background-color: var(--destructive-bg-color);
  color: var(--destructive-color);
}
.gtk-btn.destructive-action:hover, button.destructive-action:hover { background-color: var(--destructive-hover-bg); }

.gtk-btn.flat { background-color: transparent; }
.gtk-btn.flat:hover { background-color: var(--button-bg); }
.gtk-btn.circular { border-radius: 999px; width: 34px; height: 34px; padding: 0; }
.gtk-btn.pill { border-radius: 999px; height: 38px; padding: 0 22px; }

.gtk-dropdown, select {
  appearance: none;
  -webkit-appearance: none;
  display: inline-flex;
  align-items: center;
  background-color: var(--button-bg);
  background-image: var(--select-chevron);
  background-repeat: no-repeat;
  background-position: right 10px center;
  color: var(--window-fg-color);
  border: none;
  border-radius: var(--control-radius);
  height: 34px;
  min-height: 34px;
  padding: 0 32px 0 14px;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  outline: none;
  cursor: default;
}

.gtk-dropdown:hover, select:hover { background-color: var(--button-hover-bg); }
.gtk-dropdown option, select option {
  background-color: var(--card-bg-color);
  color: var(--window-fg-color);
}

.gtk-dropdown-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.gtk-dropdown-btn {
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  background-color: var(--button-bg);
  color: var(--window-fg-color);
  border: none;
  border-radius: var(--control-radius);
  height: 34px;
  min-height: 34px;
  padding: 0 11px 0 14px;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  cursor: default;
  outline: none;
}

.gtk-dropdown-btn:hover { background-color: var(--button-hover-bg); }
.gtk-dropdown-btn:active, .gtk-dropdown-btn.open { background-color: var(--button-active-bg); }
.gtk-dropdown-btn:focus-visible { box-shadow: var(--focus-ring); }

.gtk-dropdown-chevron {
  width: 12px;
  height: 12px;
  opacity: 0.85;
  flex-shrink: 0;
}

.gtk-popover-menu {
  position: fixed;
  z-index: 9999;
  min-width: 148px;
  padding: 6px;
  border-radius: var(--card-radius);
  background-color: var(--popover-bg-color);
  color: var(--popover-fg-color);
  box-shadow: var(--popover-shadow);
  display: flex;
  flex-direction: column;
  gap: 2px;
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
  transform-origin: top right;
  pointer-events: none;
  transition: opacity 110ms ease, transform 110ms cubic-bezier(0.22, 1, 0.36, 1);
}

.gtk-popover-menu.open {
  opacity: 1;
  transform: scale(1) translateY(0);
  pointer-events: auto;
}

.gtk-popover-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  height: 34px;
  min-height: 34px;
  padding: 0 10px 0 12px;
  border: none;
  border-radius: var(--control-radius);
  background-color: transparent;
  color: var(--popover-fg-color);
  font-family: inherit;
  font-size: 14px;
  font-weight: 400;
  text-align: left;
  cursor: default;
  outline: none;
}

.gtk-popover-item:hover { background-color: var(--popover-hover-bg); }
.gtk-popover-item:active { background-color: var(--popover-active-bg); }

.gtk-popover-check {
  width: 15px;
  height: 15px;
  opacity: 0;
  flex-shrink: 0;
}

.gtk-popover-item.selected .gtk-popover-check {
  opacity: 1;
}

.adw-switch, .gtk-switch {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 44px;
  height: 24px;
  flex-shrink: 0;
  cursor: pointer;
  user-select: none;
  -webkit-user-select: none;
}

.adw-switch input, .gtk-switch input {
  opacity: 0;
  width: 0;
  height: 0;
  position: absolute;
  margin: 0;
}

.adw-switch-slider {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--button-bg);
  border: 1px solid var(--row-divider-color);
  border-radius: 999px;
  transition: background-color 180ms cubic-bezier(0.2, 0, 0, 1), border-color 180ms ease;
}

.adw-switch-slider::before {
  content: "";
  position: absolute;
  height: 18px;
  width: 18px;
  left: 2px;
  bottom: 2px;
  background-color: #ffffff;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.28);
  transition: transform 180ms cubic-bezier(0.2, 0, 0, 1), background-color 180ms ease;
}

.adw-switch:hover .adw-switch-slider {
  background-color: var(--button-hover-bg);
}

.adw-switch input:checked + .adw-switch-slider {
  background-color: var(--accent-bg-color);
  border-color: transparent;
}

.adw-switch input:checked + .adw-switch-slider::before {
  transform: translateX(20px);
}

.adw-switch input:focus-visible + .adw-switch-slider {
  box-shadow: var(--focus-ring);
}

.adw-switch input:disabled + .adw-switch-slider {
  opacity: 0.45;
  cursor: not-allowed;
}

/* Checkbox & Radio Buttons */
input[type="checkbox"], input[type="radio"] {
  accent-color: var(--accent-bg-color);
  cursor: pointer;
  width: 17px;
  height: 17px;
  flex-shrink: 0;
}

.adw-check-label, .adw-radio-label {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 500;
  color: var(--window-fg-color);
  cursor: pointer;
  user-select: none;
}

/* Range / Slider */
input[type="range"] {
  --range-progress: 50%;
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 24px;
  background: transparent;
  outline: none;
  cursor: pointer;
  margin: 0;
}

input[type="range"]::-webkit-slider-runnable-track {
  width: 100%;
  height: 6px;
  border-radius: 999px;
  background: linear-gradient(
    to right,
    var(--accent-bg-color) 0%,
    var(--accent-bg-color) var(--range-progress, 50%),
    rgba(128, 128, 128, 0.28) var(--range-progress, 50%),
    rgba(128, 128, 128, 0.28) 100%
  ) no-repeat;
  background-color: rgba(128, 128, 128, 0.28);
  background-size: 100% 100%;
  border: 1px solid var(--row-divider-color);
}

input[type="range"]:hover::-webkit-slider-runnable-track {
  border-color: rgba(128, 128, 128, 0.4);
}

input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  margin-top: -6px;
  border-radius: 50%;
  background-color: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.18);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.18), 0 2px 6px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  transition: transform 120ms ease;
}

input[type="range"]:active::-webkit-slider-thumb {
  transform: scale(1.12);
  box-shadow: 0 1px 5px rgba(0, 0, 0, 0.32), 0 0 0 4px var(--accent-subtle-bg, rgba(53, 132, 228, 0.2));
}

input[type="range"]:focus-visible::-webkit-slider-thumb {
  box-shadow: var(--focus-ring);
}

input[type="range"]::-moz-range-track {
  width: 100%;
  height: 6px;
  border-radius: 999px;
  background-color: rgba(128, 128, 128, 0.28);
  border: 1px solid var(--row-divider-color);
}

input[type="range"]::-moz-range-progress {
  background-color: var(--accent-bg-color);
  height: 6px;
  border-radius: 999px;
}

input[type="range"]::-moz-range-thumb {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background-color: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.18);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.18), 0 2px 6px rgba(0, 0, 0, 0.08);
  cursor: pointer;
}
`
