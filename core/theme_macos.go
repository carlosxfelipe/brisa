package core

// ThemeMacOS is the Apple macOS (Aqua) design system.
const ThemeMacOS Theme = "macos"

// RenderMacOSCSS returns the built-in macOS CSS kit.
// Light and Dark modes remain strictly governed by the operating system.
func RenderMacOSCSS() string {
	return RenderThemeCSS(ThemeMacOS)
}

const macosThemeTokens = `:root {
  color-scheme: light dark;
  --font-sans: -apple-system, BlinkMacSystemFont, "SF Pro Text", "SF Pro Display", "Inter", system-ui, sans-serif;
  --control-radius: 8px;
  --card-radius: 11px;
  --window-bg-color: #1c1c1e;
  --window-fg-color: #f5f5f7;
  --view-bg-color: #161618;
  --view-fg-color: #f5f5f7;
  --headerbar-bg-color: #242426;
  --headerbar-fg-color: #f5f5f7;
  --headerbar-border-color: rgba(255, 255, 255, 0.10);
  --sidebar-bg-color: #202022;
  --sidebar-fg-color: #f5f5f7;
  --sidebar-border-color: rgba(255, 255, 255, 0.09);
  --card-bg-color: #262629;
  --card-hover-bg: #2e2e32;
  --card-border-color: rgba(255, 255, 255, 0.10);
  --row-divider-color: rgba(255, 255, 255, 0.08);
  --dim-label-color: rgba(235, 235, 245, 0.60);
  --button-bg: rgba(255, 255, 255, 0.11);
  --button-hover-bg: rgba(255, 255, 255, 0.16);
  --button-active-bg: rgba(255, 255, 255, 0.22);
  --nav-active-bg: rgba(10, 132, 255, 0.24);
  --nav-hover-bg: rgba(255, 255, 255, 0.07);
  --accent-bg-color: #0a84ff;
  --accent-hover-bg: #2994ff;
  --accent-active-bg: #0070e0;
  --accent-fg-color: #ffffff;
  --accent-color: #0a84ff;
  --accent-subtle-bg: rgba(10, 132, 255, 0.18);
  --destructive-bg-color: rgba(255, 69, 58, 0.18);
  --destructive-hover-bg: rgba(255, 69, 58, 0.28);
  --destructive-color: #ff453a;
  --info-bg-color: rgba(10, 132, 255, 0.18);
  --info-color: #0a84ff;
  --success-bg-color: rgba(48, 209, 88, 0.18);
  --success-color: #30d158;
  --warning-color: #ff9f0a;
  --focus-ring: 0 0 0 3px rgba(10, 132, 255, 0.50);
  --card-shadow: 0 1px 3px rgba(0, 0, 0, 0.32), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #2c2c2e;
  --popover-fg-color: #f5f5f7;
  --popover-border-color: rgba(255, 255, 255, 0.14);
  --popover-shadow: 0 10px 28px rgba(0, 0, 0, 0.52), 0 2px 6px rgba(0, 0, 0, 0.30), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(10, 132, 255, 0.22);
  --popover-active-bg: rgba(10, 132, 255, 0.32);
  --toast-bg: rgba(28, 28, 30, 0.94);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23f5f5f7' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}

@media (prefers-color-scheme: light) {
  :root:not([data-os-theme="dark"]) {
    --window-bg-color: #f5f5f7;
    --window-fg-color: #1d1d1f;
    --view-bg-color: #ffffff;
    --view-fg-color: #1d1d1f;
    --headerbar-bg-color: #ededf0;
    --headerbar-fg-color: #1d1d1f;
    --headerbar-border-color: rgba(0, 0, 0, 0.10);
    --sidebar-bg-color: #ebebf0;
    --sidebar-fg-color: #1d1d1f;
    --sidebar-border-color: rgba(0, 0, 0, 0.09);
    --card-bg-color: #ffffff;
    --card-hover-bg: #f8f8fa;
    --card-border-color: rgba(0, 0, 0, 0.08);
    --row-divider-color: rgba(60, 60, 67, 0.12);
    --dim-label-color: rgba(60, 60, 67, 0.60);
    --button-bg: rgba(0, 0, 0, 0.06);
    --button-hover-bg: rgba(0, 0, 0, 0.10);
    --button-active-bg: rgba(0, 0, 0, 0.15);
    --nav-active-bg: rgba(0, 122, 255, 0.15);
    --nav-hover-bg: rgba(0, 0, 0, 0.05);
    --accent-bg-color: #007aff;
    --accent-hover-bg: #006ee6;
    --accent-active-bg: #0062cc;
    --accent-fg-color: #ffffff;
    --accent-color: #007aff;
    --accent-subtle-bg: rgba(0, 122, 255, 0.14);
    --destructive-bg-color: rgba(255, 59, 48, 0.12);
    --destructive-hover-bg: rgba(255, 59, 48, 0.20);
    --destructive-color: #ff3b30;
    --info-bg-color: rgba(0, 122, 255, 0.14);
    --info-color: #007aff;
    --success-bg-color: rgba(52, 199, 89, 0.14);
    --success-color: #28a745;
    --warning-color: #ff9500;
    --focus-ring: 0 0 0 3px rgba(0, 122, 255, 0.42);
    --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
    --popover-bg-color: #ffffff;
    --popover-fg-color: #1d1d1f;
    --popover-border-color: rgba(0, 0, 0, 0.11);
    --popover-shadow: 0 10px 28px rgba(0, 0, 0, 0.15), 0 2px 6px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
    --popover-hover-bg: rgba(0, 122, 255, 0.14);
    --popover-active-bg: rgba(0, 122, 255, 0.22);
    --toast-bg: rgba(29, 29, 31, 0.94);
    --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%231d1d1f' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
  }
}

:root[data-os-theme="light"] {
  --window-bg-color: #f5f5f7;
  --window-fg-color: #1d1d1f;
  --view-bg-color: #ffffff;
  --view-fg-color: #1d1d1f;
  --headerbar-bg-color: #ededf0;
  --headerbar-fg-color: #1d1d1f;
  --headerbar-border-color: rgba(0, 0, 0, 0.10);
  --sidebar-bg-color: #ebebf0;
  --sidebar-fg-color: #1d1d1f;
  --sidebar-border-color: rgba(0, 0, 0, 0.09);
  --card-bg-color: #ffffff;
  --card-hover-bg: #f8f8fa;
  --card-border-color: rgba(0, 0, 0, 0.08);
  --row-divider-color: rgba(60, 60, 67, 0.12);
  --dim-label-color: rgba(60, 60, 67, 0.60);
  --button-bg: rgba(0, 0, 0, 0.06);
  --button-hover-bg: rgba(0, 0, 0, 0.10);
  --button-active-bg: rgba(0, 0, 0, 0.15);
  --nav-active-bg: rgba(0, 122, 255, 0.15);
  --nav-hover-bg: rgba(0, 0, 0, 0.05);
  --accent-bg-color: #007aff;
  --accent-hover-bg: #006ee6;
  --accent-active-bg: #0062cc;
  --accent-fg-color: #ffffff;
  --accent-color: #007aff;
  --accent-subtle-bg: rgba(0, 122, 255, 0.14);
  --destructive-bg-color: rgba(255, 59, 48, 0.12);
  --destructive-hover-bg: rgba(255, 59, 48, 0.20);
  --destructive-color: #ff3b30;
  --info-bg-color: rgba(0, 122, 255, 0.14);
  --info-color: #007aff;
  --success-bg-color: rgba(52, 199, 89, 0.14);
  --success-color: #28a745;
  --warning-color: #ff9500;
  --focus-ring: 0 0 0 3px rgba(0, 122, 255, 0.42);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #ffffff;
  --popover-fg-color: #1d1d1f;
  --popover-border-color: rgba(0, 0, 0, 0.11);
  --popover-shadow: 0 10px 28px rgba(0, 0, 0, 0.15), 0 2px 6px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(0, 122, 255, 0.14);
  --popover-active-bg: rgba(0, 122, 255, 0.22);
  --toast-bg: rgba(29, 29, 31, 0.94);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%231d1d1f' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}
`
