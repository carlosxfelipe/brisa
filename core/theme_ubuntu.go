package core

// ThemeUbuntu is the Canonical Ubuntu Yaru (GTK4 / Libadwaita) design system.
const ThemeUbuntu Theme = "ubuntu"

// RenderUbuntuCSS returns the built-in Ubuntu Yaru (GTK4 / Libadwaita) CSS kit.
// Light and Dark modes remain strictly governed by the operating system.
func RenderUbuntuCSS() string {
	return RenderThemeCSS(ThemeUbuntu)
}

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
