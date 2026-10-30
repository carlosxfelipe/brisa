package core

import (
	"bufio"
	"context"
	"os/exec"
	"runtime"
	"strings"
)

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

// RenderAdwaitaCSS returns the built-in GNOME 50 / GTK4 Libadwaita CSS kit served at /brisa/adwaita.css.
// Theme is strictly governed by the operating system.
func RenderAdwaitaCSS() string {
	return `/* Brisa Built-in UI Kit — GNOME 50 / GTK4 / Libadwaita 1.6+ Design System (OS-governed) */
:root {
  color-scheme: light dark;
  --window-bg-color: #222226;
  --window-fg-color: #ffffff;
  --view-bg-color: #1d1d20;
  --view-fg-color: #ffffff;
  --headerbar-bg-color: #2e2e32;
  --headerbar-fg-color: #ffffff;
  --headerbar-border-color: rgba(0, 0, 0, 0.48);
  --sidebar-bg-color: #2a2a2e;
  --sidebar-fg-color: #ffffff;
  --sidebar-border-color: rgba(0, 0, 0, 0.42);
  --card-bg-color: #2e2e32;
  --card-hover-bg: #343439;
  --card-border-color: rgba(255, 255, 255, 0.06);
  --row-divider-color: rgba(255, 255, 255, 0.07);
  --dim-label-color: rgba(255, 255, 255, 0.56);
  --button-bg: rgba(255, 255, 255, 0.10);
  --button-hover-bg: rgba(255, 255, 255, 0.15);
  --button-active-bg: rgba(255, 255, 255, 0.22);
  --nav-active-bg: rgba(255, 255, 255, 0.12);
  --nav-hover-bg: rgba(255, 255, 255, 0.06);
  --accent-bg-color: #3584e4;
  --accent-hover-bg: #438de6;
  --accent-active-bg: #2a76d2;
  --accent-fg-color: #ffffff;
  --accent-color: #78aeed;
  --destructive-bg-color: rgba(224, 27, 36, 0.16);
  --destructive-hover-bg: rgba(224, 27, 36, 0.26);
  --destructive-color: #ff7b63;
  --success-color: #57e389;
  --warning-color: #f8e45c;
  --focus-ring: 0 0 0 2px rgba(120, 174, 237, 0.55);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.24), 0 0 0 1px var(--card-border-color);
  --toast-bg: rgba(18, 18, 21, 0.94);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23ffffff' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}

@media (prefers-color-scheme: light) {
  :root:not([data-os-theme="dark"]) {
    --window-bg-color: #fafafb;
    --window-fg-color: rgba(0, 0, 0, 0.85);
    --view-bg-color: #ffffff;
    --view-fg-color: rgba(0, 0, 0, 0.85);
    --headerbar-bg-color: #ebebed;
    --headerbar-fg-color: rgba(0, 0, 0, 0.85);
    --headerbar-border-color: rgba(0, 0, 0, 0.14);
    --sidebar-bg-color: #ebebed;
    --sidebar-fg-color: rgba(0, 0, 0, 0.85);
    --sidebar-border-color: rgba(0, 0, 0, 0.12);
    --card-bg-color: #ffffff;
    --card-hover-bg: #f6f6f8;
    --card-border-color: rgba(0, 0, 0, 0.08);
    --row-divider-color: rgba(0, 0, 0, 0.08);
    --dim-label-color: rgba(0, 0, 0, 0.55);
    --button-bg: rgba(0, 0, 0, 0.07);
    --button-hover-bg: rgba(0, 0, 0, 0.11);
    --button-active-bg: rgba(0, 0, 0, 0.16);
    --nav-active-bg: rgba(0, 0, 0, 0.09);
    --nav-hover-bg: rgba(0, 0, 0, 0.05);
    --accent-bg-color: #3584e4;
    --accent-hover-bg: #2a76d2;
    --accent-active-bg: #1c65bd;
    --accent-fg-color: #ffffff;
    --accent-color: #1c71d8;
    --destructive-bg-color: rgba(224, 27, 36, 0.10);
    --destructive-hover-bg: rgba(224, 27, 36, 0.18);
    --destructive-color: #c01c28;
    --success-color: #26a269;
    --warning-color: #b5835a;
    --focus-ring: 0 0 0 2px rgba(53, 132, 228, 0.45);
    --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
    --toast-bg: rgba(34, 34, 38, 0.94);
    --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23222226' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
  }
}

:root[data-os-theme="light"] {
  --window-bg-color: #fafafb;
  --window-fg-color: rgba(0, 0, 0, 0.85);
  --view-bg-color: #ffffff;
  --view-fg-color: rgba(0, 0, 0, 0.85);
  --headerbar-bg-color: #ebebed;
  --headerbar-fg-color: rgba(0, 0, 0, 0.85);
  --headerbar-border-color: rgba(0, 0, 0, 0.14);
  --sidebar-bg-color: #ebebed;
  --sidebar-fg-color: rgba(0, 0, 0, 0.85);
  --sidebar-border-color: rgba(0, 0, 0, 0.12);
  --card-bg-color: #ffffff;
  --card-hover-bg: #f6f6f8;
  --card-border-color: rgba(0, 0, 0, 0.08);
  --row-divider-color: rgba(0, 0, 0, 0.08);
  --dim-label-color: rgba(0, 0, 0, 0.55);
  --button-bg: rgba(0, 0, 0, 0.07);
  --button-hover-bg: rgba(0, 0, 0, 0.11);
  --button-active-bg: rgba(0, 0, 0, 0.16);
  --nav-active-bg: rgba(0, 0, 0, 0.09);
  --nav-hover-bg: rgba(0, 0, 0, 0.05);
  --accent-bg-color: #3584e4;
  --accent-hover-bg: #2a76d2;
  --accent-active-bg: #1c65bd;
  --accent-fg-color: #ffffff;
  --accent-color: #1c71d8;
  --destructive-bg-color: rgba(224, 27, 36, 0.10);
  --destructive-hover-bg: rgba(224, 27, 36, 0.18);
  --destructive-color: #c01c28;
  --success-color: #26a269;
  --warning-color: #b5835a;
  --focus-ring: 0 0 0 2px rgba(53, 132, 228, 0.45);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
  --toast-bg: rgba(34, 34, 38, 0.94);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23222226' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body {
  font-family: "Cantarell", "Inter", system-ui, -apple-system, sans-serif;
  font-size: 14px;
  line-height: 1.4;
  background-color: var(--window-bg-color);
  color: var(--window-fg-color);
  -webkit-font-smoothing: antialiased;
}

.adw-split-view {
  display: grid;
  grid-template-columns: 256px 1fr;
  height: 100vh;
  width: 100vw;
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
  border-radius: 9px;
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
  border-radius: 12px;
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

.adw-row:last-child { border-bottom: none; }
.adw-row.activatable:hover { background-color: var(--card-hover-bg); }
.adw-row:focus-within { box-shadow: inset var(--focus-ring); }

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

/* Unified GTK4 / Libadwaita 1.6+ Button & Dropdown Geometry (34px height, 9px radius) */
.gtk-btn, button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  background-color: var(--button-bg);
  color: var(--window-fg-color);
  border: none;
  border-radius: 9px;
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
  border-radius: 9px;
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
`
}
