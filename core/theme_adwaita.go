package core

// ThemeAdwaita is the default GNOME 50 / GTK4 Libadwaita design system.
const ThemeAdwaita Theme = "adwaita"

// RenderAdwaitaCSS returns the built-in GNOME 50 / GTK4 Libadwaita CSS kit.
// Theme is strictly governed by the operating system.
func RenderAdwaitaCSS() string {
	return RenderThemeCSS(ThemeAdwaita)
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
