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

const (
	// ThemeAdwaita is the default GNOME 50 / GTK4 Libadwaita design system.
	ThemeAdwaita Theme = "adwaita"
	// ThemeUbuntu is the Canonical Ubuntu Yaru (GTK4 / Libadwaita) design system.
	ThemeUbuntu Theme = "ubuntu"
)

// NormalizeTheme normalizes a Theme value, defaulting to ThemeAdwaita.
func NormalizeTheme(theme Theme) Theme {
	switch strings.ToLower(strings.TrimSpace(string(theme))) {
	case "ubuntu", "yaru":
		return ThemeUbuntu
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

// RenderAdwaitaCSS returns the built-in GNOME 50 / GTK4 Libadwaita CSS kit.
// Theme is strictly governed by the operating system.
func RenderAdwaitaCSS() string {
	return RenderThemeCSS(ThemeAdwaita)
}

// RenderUbuntuCSS returns the built-in Ubuntu Yaru (GTK4 / Libadwaita) CSS kit.
// Light and Dark modes remain strictly governed by the operating system.
func RenderUbuntuCSS() string {
	return RenderThemeCSS(ThemeUbuntu)
}

// RenderThemeCSS returns the built-in desktop CSS kit for the requested Theme
// (ThemeAdwaita or ThemeUbuntu), keeping all component classes 100% compatible.
func RenderThemeCSS(theme Theme) string {
	tokens := adwaitaThemeTokens
	if NormalizeTheme(theme) == ThemeUbuntu {
		tokens = ubuntuThemeTokens
	}
	return tokens + sharedComponentCSS
}

const adwaitaThemeTokens = `/* Brisa Built-in UI Kit — GNOME 50 / GTK4 / Libadwaita 1.6+ Design System (OS-governed) */
:root {
  color-scheme: light dark;
  --font-sans: "Cantarell", "Inter", system-ui, -apple-system, sans-serif;
  --control-radius: 9px;
  --card-radius: 12px;
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
  --accent-subtle-bg: rgba(53, 132, 228, 0.18);
  --destructive-bg-color: rgba(224, 27, 36, 0.16);
  --destructive-hover-bg: rgba(224, 27, 36, 0.26);
  --destructive-color: #ff7b63;
  --info-bg-color: rgba(53, 132, 228, 0.18);
  --info-color: #78aeed;
  --success-bg-color: rgba(87, 227, 137, 0.15);
  --success-color: #57e389;
  --warning-color: #f8e45c;
  --focus-ring: 0 0 0 2px rgba(120, 174, 237, 0.55);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.24), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #36363a;
  --popover-fg-color: #ffffff;
  --popover-border-color: rgba(255, 255, 255, 0.09);
  --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.45), 0 1px 3px rgba(0, 0, 0, 0.28), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(255, 255, 255, 0.08);
  --popover-active-bg: rgba(255, 255, 255, 0.14);
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
    --accent-subtle-bg: rgba(53, 132, 228, 0.14);
    --destructive-bg-color: rgba(224, 27, 36, 0.10);
    --destructive-hover-bg: rgba(224, 27, 36, 0.18);
    --destructive-color: #c01c28;
    --info-bg-color: rgba(53, 132, 228, 0.14);
    --info-color: #1c71d8;
    --success-bg-color: rgba(38, 162, 105, 0.14);
    --success-color: #26a269;
    --warning-color: #b5835a;
    --focus-ring: 0 0 0 2px rgba(53, 132, 228, 0.45);
    --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
    --popover-bg-color: #ffffff;
    --popover-fg-color: rgba(0, 0, 0, 0.85);
    --popover-border-color: rgba(0, 0, 0, 0.10);
    --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.14), 0 1px 3px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
    --popover-hover-bg: rgba(0, 0, 0, 0.06);
    --popover-active-bg: rgba(0, 0, 0, 0.11);
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
  --accent-subtle-bg: rgba(53, 132, 228, 0.14);
  --destructive-bg-color: rgba(224, 27, 36, 0.10);
  --destructive-hover-bg: rgba(224, 27, 36, 0.18);
  --destructive-color: #c01c28;
  --info-bg-color: rgba(53, 132, 228, 0.14);
  --info-color: #1c71d8;
  --success-bg-color: rgba(38, 162, 105, 0.14);
  --success-color: #26a269;
  --warning-color: #b5835a;
  --focus-ring: 0 0 0 2px rgba(53, 132, 228, 0.45);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #ffffff;
  --popover-fg-color: rgba(0, 0, 0, 0.85);
  --popover-border-color: rgba(0, 0, 0, 0.10);
  --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.14), 0 1px 3px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(0, 0, 0, 0.06);
  --popover-active-bg: rgba(0, 0, 0, 0.11);
  --toast-bg: rgba(34, 34, 38, 0.94);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23222226' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}
`

const ubuntuThemeTokens = `/* Brisa Built-in UI Kit — Canonical Ubuntu Yaru / GTK4 / Libadwaita Design System (OS-governed) */
:root {
  color-scheme: light dark;
  --font-sans: "Ubuntu Sans", "Ubuntu", "Cantarell", "Inter", system-ui, -apple-system, sans-serif;
  --control-radius: 6px;
  --card-radius: 8px;
  --window-bg-color: #2c2c2c;
  --window-fg-color: #ffffff;
  --view-bg-color: #222222;
  --view-fg-color: #ffffff;
  --headerbar-bg-color: #242424;
  --headerbar-fg-color: #ffffff;
  --headerbar-border-color: rgba(0, 0, 0, 0.52);
  --sidebar-bg-color: #262626;
  --sidebar-fg-color: #ffffff;
  --sidebar-border-color: rgba(0, 0, 0, 0.45);
  --card-bg-color: #343434;
  --card-hover-bg: #3c3c3c;
  --card-border-color: rgba(255, 255, 255, 0.07);
  --row-divider-color: rgba(255, 255, 255, 0.08);
  --dim-label-color: rgba(255, 255, 255, 0.58);
  --button-bg: rgba(255, 255, 255, 0.09);
  --button-hover-bg: rgba(255, 255, 255, 0.14);
  --button-active-bg: rgba(255, 255, 255, 0.20);
  --nav-active-bg: rgba(233, 84, 32, 0.22);
  --nav-hover-bg: rgba(255, 255, 255, 0.06);
  --accent-bg-color: #e95420;
  --accent-hover-bg: #eb6536;
  --accent-active-bg: #d34615;
  --accent-fg-color: #ffffff;
  --accent-color: #f07746;
  --accent-subtle-bg: rgba(233, 84, 32, 0.18);
  --destructive-bg-color: rgba(224, 27, 36, 0.16);
  --destructive-hover-bg: rgba(224, 27, 36, 0.26);
  --destructive-color: #ff7b63;
  --info-bg-color: rgba(53, 132, 228, 0.18);
  --info-color: #78aeed;
  --success-bg-color: rgba(71, 212, 90, 0.15);
  --success-color: #47d45a;
  --warning-color: #f99b11;
  --focus-ring: 0 0 0 2px rgba(233, 84, 32, 0.55);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.28), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #383838;
  --popover-fg-color: #ffffff;
  --popover-border-color: rgba(255, 255, 255, 0.09);
  --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.48), 0 1px 3px rgba(0, 0, 0, 0.30), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(255, 255, 255, 0.08);
  --popover-active-bg: rgba(255, 255, 255, 0.14);
  --toast-bg: rgba(24, 24, 24, 0.95);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23ffffff' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}

@media (prefers-color-scheme: light) {
  :root:not([data-os-theme="dark"]) {
    --window-bg-color: #fafafa;
    --window-fg-color: #2c2c2c;
    --view-bg-color: #ffffff;
    --view-fg-color: #2c2c2c;
    --headerbar-bg-color: #f0f0f0;
    --headerbar-fg-color: #2c2c2c;
    --headerbar-border-color: rgba(0, 0, 0, 0.12);
    --sidebar-bg-color: #f2f2f2;
    --sidebar-fg-color: #2c2c2c;
    --sidebar-border-color: rgba(0, 0, 0, 0.11);
    --card-bg-color: #ffffff;
    --card-hover-bg: #f5f5f5;
    --card-border-color: rgba(0, 0, 0, 0.09);
    --row-divider-color: rgba(0, 0, 0, 0.08);
    --dim-label-color: rgba(0, 0, 0, 0.56);
    --button-bg: rgba(0, 0, 0, 0.065);
    --button-hover-bg: rgba(0, 0, 0, 0.105);
    --button-active-bg: rgba(0, 0, 0, 0.15);
    --nav-active-bg: rgba(233, 84, 32, 0.14);
    --nav-hover-bg: rgba(0, 0, 0, 0.05);
    --accent-bg-color: #e95420;
    --accent-hover-bg: #d34615;
    --accent-active-bg: #b83b10;
    --accent-fg-color: #ffffff;
    --accent-color: #c34113;
    --accent-subtle-bg: rgba(233, 84, 32, 0.14);
    --destructive-bg-color: rgba(199, 22, 43, 0.10);
    --destructive-hover-bg: rgba(199, 22, 43, 0.18);
    --destructive-color: #c7162b;
    --info-bg-color: rgba(53, 132, 228, 0.14);
    --info-color: #1c71d8;
    --success-bg-color: rgba(14, 132, 32, 0.14);
    --success-color: #0e8420;
    --warning-color: #c46c00;
    --focus-ring: 0 0 0 2px rgba(233, 84, 32, 0.45);
    --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
    --popover-bg-color: #ffffff;
    --popover-fg-color: #2c2c2c;
    --popover-border-color: rgba(0, 0, 0, 0.10);
    --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.14), 0 1px 3px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
    --popover-hover-bg: rgba(0, 0, 0, 0.06);
    --popover-active-bg: rgba(0, 0, 0, 0.11);
    --toast-bg: rgba(38, 38, 38, 0.95);
    --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%232c2c2c' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
  }
}

:root[data-os-theme="light"] {
  --window-bg-color: #fafafa;
  --window-fg-color: #2c2c2c;
  --view-bg-color: #ffffff;
  --view-fg-color: #2c2c2c;
  --headerbar-bg-color: #f0f0f0;
  --headerbar-fg-color: #2c2c2c;
  --headerbar-border-color: rgba(0, 0, 0, 0.12);
  --sidebar-bg-color: #f2f2f2;
  --sidebar-fg-color: #2c2c2c;
  --sidebar-border-color: rgba(0, 0, 0, 0.11);
  --card-bg-color: #ffffff;
  --card-hover-bg: #f5f5f5;
  --card-border-color: rgba(0, 0, 0, 0.09);
  --row-divider-color: rgba(0, 0, 0, 0.08);
  --dim-label-color: rgba(0, 0, 0, 0.56);
  --button-bg: rgba(0, 0, 0, 0.065);
  --button-hover-bg: rgba(0, 0, 0, 0.105);
  --button-active-bg: rgba(0, 0, 0, 0.15);
  --nav-active-bg: rgba(233, 84, 32, 0.14);
  --nav-hover-bg: rgba(0, 0, 0, 0.05);
  --accent-bg-color: #e95420;
  --accent-hover-bg: #d34615;
  --accent-active-bg: #b83b10;
  --accent-fg-color: #ffffff;
  --accent-color: #c34113;
  --accent-subtle-bg: rgba(233, 84, 32, 0.14);
  --destructive-bg-color: rgba(199, 22, 43, 0.10);
  --destructive-hover-bg: rgba(199, 22, 43, 0.18);
  --destructive-color: #c7162b;
  --info-bg-color: rgba(53, 132, 228, 0.14);
  --info-color: #1c71d8;
  --success-bg-color: rgba(14, 132, 32, 0.14);
  --success-color: #0e8420;
  --warning-color: #c46c00;
  --focus-ring: 0 0 0 2px rgba(233, 84, 32, 0.45);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #ffffff;
  --popover-fg-color: #2c2c2c;
  --popover-border-color: rgba(0, 0, 0, 0.10);
  --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.14), 0 1px 3px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(0, 0, 0, 0.06);
  --popover-active-bg: rgba(0, 0, 0, 0.11);
  --toast-bg: rgba(38, 38, 38, 0.95);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%232c2c2c' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}
`

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

/* Unified GTK4 / Libadwaita 1.6+ Button & Dropdown Geometry (34px height) */
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

/* GTK4 / Libadwaita GtkDropDown & GtkPopover Menu */
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
`
