package core

// ThemeWindows is the Microsoft Windows 11 (Fluent / WinUI 3) design system.
const ThemeWindows Theme = "windows"

// RenderWindowsCSS returns the built-in Windows 11 (Fluent) CSS kit.
// Light and Dark modes remain strictly governed by the operating system.
func RenderWindowsCSS() string {
	return RenderThemeCSS(ThemeWindows)
}

const windowsThemeTokens = `:root {
  color-scheme: light dark;
  --font-sans: "Segoe UI Variable", "Segoe UI", "Inter", system-ui, -apple-system, sans-serif;
  --control-radius: 4px;
  --card-radius: 8px;
  --window-bg-color: #202020;
  --window-fg-color: #ffffff;
  --view-bg-color: #191919;
  --view-fg-color: #ffffff;
  --headerbar-bg-color: #202020;
  --headerbar-fg-color: #ffffff;
  --headerbar-border-color: rgba(255, 255, 255, 0.08);
  --sidebar-bg-color: #202020;
  --sidebar-fg-color: #ffffff;
  --sidebar-border-color: rgba(255, 255, 255, 0.07);
  --card-bg-color: #2b2b2b;
  --card-hover-bg: #323232;
  --card-border-color: rgba(255, 255, 255, 0.08);
  --row-divider-color: rgba(255, 255, 255, 0.08);
  --dim-label-color: rgba(255, 255, 255, 0.60);
  --button-bg: rgba(255, 255, 255, 0.07);
  --button-hover-bg: rgba(255, 255, 255, 0.11);
  --button-active-bg: rgba(255, 255, 255, 0.05);
  --nav-active-bg: rgba(96, 205, 255, 0.16);
  --nav-hover-bg: rgba(255, 255, 255, 0.06);
  --accent-bg-color: #0067c0;
  --accent-hover-bg: #1975c5;
  --accent-active-bg: #005ba9;
  --accent-fg-color: #ffffff;
  --accent-color: #60cdff;
  --accent-subtle-bg: rgba(96, 205, 255, 0.16);
  --destructive-bg-color: rgba(196, 43, 28, 0.22);
  --destructive-hover-bg: rgba(196, 43, 28, 0.32);
  --destructive-color: #ff99a4;
  --info-bg-color: rgba(96, 205, 255, 0.16);
  --info-color: #60cdff;
  --success-bg-color: rgba(108, 203, 95, 0.16);
  --success-color: #6ccb5f;
  --warning-color: #fce100;
  --focus-ring: 0 0 0 2px rgba(96, 205, 255, 0.60);
  --card-shadow: 0 2px 4px rgba(0, 0, 0, 0.26), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #2c2c2c;
  --popover-fg-color: #ffffff;
  --popover-border-color: rgba(255, 255, 255, 0.10);
  --popover-shadow: 0 8px 24px rgba(0, 0, 0, 0.48), 0 2px 6px rgba(0, 0, 0, 0.28), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(255, 255, 255, 0.08);
  --popover-active-bg: rgba(255, 255, 255, 0.12);
  --toast-bg: rgba(32, 32, 32, 0.96);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23ffffff' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}

@media (prefers-color-scheme: light) {
  :root:not([data-os-theme="dark"]) {
    --window-bg-color: #f3f3f3;
    --window-fg-color: #1b1b1b;
    --view-bg-color: #fbfbfb;
    --view-fg-color: #1b1b1b;
    --headerbar-bg-color: #eeeeee;
    --headerbar-fg-color: #1b1b1b;
    --headerbar-border-color: rgba(0, 0, 0, 0.08);
    --sidebar-bg-color: #f0f0f0;
    --sidebar-fg-color: #1b1b1b;
    --sidebar-border-color: rgba(0, 0, 0, 0.07);
    --card-bg-color: #fbfbfb;
    --card-hover-bg: #f5f5f5;
    --card-border-color: rgba(0, 0, 0, 0.07);
    --row-divider-color: rgba(0, 0, 0, 0.07);
    --dim-label-color: rgba(0, 0, 0, 0.60);
    --button-bg: rgba(0, 0, 0, 0.05);
    --button-hover-bg: rgba(0, 0, 0, 0.08);
    --button-active-bg: rgba(0, 0, 0, 0.12);
    --nav-active-bg: rgba(0, 103, 192, 0.12);
    --nav-hover-bg: rgba(0, 0, 0, 0.04);
    --accent-bg-color: #0067c0;
    --accent-hover-bg: #005ba9;
    --accent-active-bg: #004e92;
    --accent-fg-color: #ffffff;
    --accent-color: #0067c0;
    --accent-subtle-bg: rgba(0, 103, 192, 0.12);
    --destructive-bg-color: rgba(196, 43, 28, 0.10);
    --destructive-hover-bg: rgba(196, 43, 28, 0.18);
    --destructive-color: #c42b1c;
    --info-bg-color: rgba(0, 103, 192, 0.12);
    --info-color: #0067c0;
    --success-bg-color: rgba(15, 123, 15, 0.12);
    --success-color: #0f7b0f;
    --warning-color: #9d5d00;
    --focus-ring: 0 0 0 2px rgba(0, 103, 192, 0.50);
    --card-shadow: 0 2px 4px rgba(0, 0, 0, 0.04), 0 0 0 1px var(--card-border-color);
    --popover-bg-color: #fcfcfc;
    --popover-fg-color: #1b1b1b;
    --popover-border-color: rgba(0, 0, 0, 0.09);
    --popover-shadow: 0 8px 24px rgba(0, 0, 0, 0.14), 0 2px 6px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--popover-border-color);
    --popover-hover-bg: rgba(0, 0, 0, 0.05);
    --popover-active-bg: rgba(0, 0, 0, 0.09);
    --toast-bg: rgba(27, 27, 27, 0.95);
    --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%231b1b1b' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
  }
}

:root[data-os-theme="light"] {
  --window-bg-color: #f3f3f3;
  --window-fg-color: #1b1b1b;
  --view-bg-color: #fbfbfb;
  --view-fg-color: #1b1b1b;
  --headerbar-bg-color: #eeeeee;
  --headerbar-fg-color: #1b1b1b;
  --headerbar-border-color: rgba(0, 0, 0, 0.08);
  --sidebar-bg-color: #f0f0f0;
  --sidebar-fg-color: #1b1b1b;
  --sidebar-border-color: rgba(0, 0, 0, 0.07);
  --card-bg-color: #fbfbfb;
  --card-hover-bg: #f5f5f5;
  --card-border-color: rgba(0, 0, 0, 0.07);
  --row-divider-color: rgba(0, 0, 0, 0.07);
  --dim-label-color: rgba(0, 0, 0, 0.60);
  --button-bg: rgba(0, 0, 0, 0.05);
  --button-hover-bg: rgba(0, 0, 0, 0.08);
  --button-active-bg: rgba(0, 0, 0, 0.12);
  --nav-active-bg: rgba(0, 103, 192, 0.12);
  --nav-hover-bg: rgba(0, 0, 0, 0.04);
  --accent-bg-color: #0067c0;
  --accent-hover-bg: #005ba9;
  --accent-active-bg: #004e92;
  --accent-fg-color: #ffffff;
  --accent-color: #0067c0;
  --accent-subtle-bg: rgba(0, 103, 192, 0.12);
  --destructive-bg-color: rgba(196, 43, 28, 0.10);
  --destructive-hover-bg: rgba(196, 43, 28, 0.18);
  --destructive-color: #c42b1c;
  --info-bg-color: rgba(0, 103, 192, 0.12);
  --info-color: #0067c0;
  --success-bg-color: rgba(15, 123, 15, 0.12);
  --success-color: #0f7b0f;
  --warning-color: #9d5d00;
  --focus-ring: 0 0 0 2px rgba(0, 103, 192, 0.50);
  --card-shadow: 0 2px 4px rgba(0, 0, 0, 0.04), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #fcfcfc;
  --popover-fg-color: #1b1b1b;
  --popover-border-color: rgba(0, 0, 0, 0.09);
  --popover-shadow: 0 8px 24px rgba(0, 0, 0, 0.14), 0 2px 6px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(0, 0, 0, 0.05);
  --popover-active-bg: rgba(0, 0, 0, 0.09);
  --toast-bg: rgba(27, 27, 27, 0.95);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%231b1b1b' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}
`
