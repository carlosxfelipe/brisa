package core

// ThemePlasma is the KDE Plasma (Breeze / Kirigami) design system.
const ThemePlasma Theme = "plasma"

// RenderPlasmaCSS returns the built-in KDE Plasma (Breeze) CSS kit.
// Light and Dark modes remain strictly governed by the operating system.
func RenderPlasmaCSS() string {
	return RenderThemeCSS(ThemePlasma)
}

const plasmaThemeTokens = `:root {
  color-scheme: light dark;
  --font-sans: "Noto Sans", "Inter", system-ui, -apple-system, sans-serif;
  --control-radius: 5px;
  --card-radius: 6px;
  --window-bg-color: #202326;
  --window-fg-color: #fcfcfc;
  --view-bg-color: #1b1e20;
  --view-fg-color: #fcfcfc;
  --headerbar-bg-color: #292c30;
  --headerbar-fg-color: #fcfcfc;
  --headerbar-border-color: rgba(0, 0, 0, 0.55);
  --sidebar-bg-color: #232629;
  --sidebar-fg-color: #fcfcfc;
  --sidebar-border-color: rgba(255, 255, 255, 0.08);
  --card-bg-color: #292c30;
  --card-hover-bg: #31363b;
  --card-border-color: rgba(255, 255, 255, 0.10);
  --row-divider-color: rgba(255, 255, 255, 0.08);
  --dim-label-color: #a1a9b1;
  --button-bg: #31363b;
  --button-hover-bg: #3b4045;
  --button-active-bg: #454a4f;
  --nav-active-bg: rgba(61, 174, 233, 0.24);
  --nav-hover-bg: rgba(61, 174, 233, 0.10);
  --accent-bg-color: #3daee9;
  --accent-hover-bg: #50b8ec;
  --accent-active-bg: #1d99f3;
  --accent-fg-color: #fcfcfc;
  --accent-color: #3daee9;
  --accent-subtle-bg: rgba(61, 174, 233, 0.18);
  --destructive-bg-color: rgba(218, 68, 83, 0.18);
  --destructive-hover-bg: rgba(218, 68, 83, 0.28);
  --destructive-color: #f44f5f;
  --info-bg-color: rgba(61, 174, 233, 0.18);
  --info-color: #3daee9;
  --success-bg-color: rgba(39, 174, 96, 0.18);
  --success-color: #2ecc71;
  --warning-color: #fdbc4b;
  --focus-ring: 0 0 0 2px rgba(61, 174, 233, 0.55);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.30), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #292c30;
  --popover-fg-color: #fcfcfc;
  --popover-border-color: rgba(255, 255, 255, 0.12);
  --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.50), 0 1px 3px rgba(0, 0, 0, 0.32), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(61, 174, 233, 0.16);
  --popover-active-bg: rgba(61, 174, 233, 0.26);
  --toast-bg: rgba(27, 30, 32, 0.95);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23fcfcfc' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}

@media (prefers-color-scheme: light) {
  :root:not([data-os-theme="dark"]) {
    --window-bg-color: #eff0f1;
    --window-fg-color: #232629;
    --view-bg-color: #fcfcfc;
    --view-fg-color: #232629;
    --headerbar-bg-color: #dee0e2;
    --headerbar-fg-color: #232629;
    --headerbar-border-color: rgba(0, 0, 0, 0.14);
    --sidebar-bg-color: #e3e5e7;
    --sidebar-fg-color: #232629;
    --sidebar-border-color: rgba(0, 0, 0, 0.12);
    --card-bg-color: #fcfcfc;
    --card-hover-bg: #f3f4f5;
    --card-border-color: rgba(0, 0, 0, 0.11);
    --row-divider-color: rgba(0, 0, 0, 0.08);
    --dim-label-color: #7f8c8d;
    --button-bg: #e3e5e7;
    --button-hover-bg: #d6d9dc;
    --button-active-bg: #c8ccd0;
    --nav-active-bg: rgba(61, 174, 233, 0.20);
    --nav-hover-bg: rgba(61, 174, 233, 0.09);
    --accent-bg-color: #3daee9;
    --accent-hover-bg: #1d99f3;
    --accent-active-bg: #1177c8;
    --accent-fg-color: #fcfcfc;
    --accent-color: #006699;
    --accent-subtle-bg: rgba(61, 174, 233, 0.15);
    --destructive-bg-color: rgba(218, 68, 83, 0.12);
    --destructive-hover-bg: rgba(218, 68, 83, 0.20);
    --destructive-color: #da4453;
    --info-bg-color: rgba(61, 174, 233, 0.15);
    --info-color: #006699;
    --success-bg-color: rgba(39, 174, 96, 0.14);
    --success-color: #27ae60;
    --warning-color: #f67400;
    --focus-ring: 0 0 0 2px rgba(61, 174, 233, 0.45);
    --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.05), 0 0 0 1px var(--card-border-color);
    --popover-bg-color: #fcfcfc;
    --popover-fg-color: #232629;
    --popover-border-color: rgba(0, 0, 0, 0.13);
    --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.14), 0 1px 3px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
    --popover-hover-bg: rgba(61, 174, 233, 0.14);
    --popover-active-bg: rgba(61, 174, 233, 0.22);
    --toast-bg: rgba(35, 38, 41, 0.94);
    --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23232629' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
  }
}

:root[data-os-theme="light"] {
  --window-bg-color: #eff0f1;
  --window-fg-color: #232629;
  --view-bg-color: #fcfcfc;
  --view-fg-color: #232629;
  --headerbar-bg-color: #dee0e2;
  --headerbar-fg-color: #232629;
  --headerbar-border-color: rgba(0, 0, 0, 0.14);
  --sidebar-bg-color: #e3e5e7;
  --sidebar-fg-color: #232629;
  --sidebar-border-color: rgba(0, 0, 0, 0.12);
  --card-bg-color: #fcfcfc;
  --card-hover-bg: #f3f4f5;
  --card-border-color: rgba(0, 0, 0, 0.11);
  --row-divider-color: rgba(0, 0, 0, 0.08);
  --dim-label-color: #7f8c8d;
  --button-bg: #e3e5e7;
  --button-hover-bg: #d6d9dc;
  --button-active-bg: #c8ccd0;
  --nav-active-bg: rgba(61, 174, 233, 0.20);
  --nav-hover-bg: rgba(61, 174, 233, 0.09);
  --accent-bg-color: #3daee9;
  --accent-hover-bg: #1d99f3;
  --accent-active-bg: #1177c8;
  --accent-fg-color: #fcfcfc;
  --accent-color: #006699;
  --accent-subtle-bg: rgba(61, 174, 233, 0.15);
  --destructive-bg-color: rgba(218, 68, 83, 0.12);
  --destructive-hover-bg: rgba(218, 68, 83, 0.20);
  --destructive-color: #da4453;
  --info-bg-color: rgba(61, 174, 233, 0.15);
  --info-color: #006699;
  --success-bg-color: rgba(39, 174, 96, 0.14);
  --success-color: #27ae60;
  --warning-color: #f67400;
  --focus-ring: 0 0 0 2px rgba(61, 174, 233, 0.45);
  --card-shadow: 0 1px 2px rgba(0, 0, 0, 0.05), 0 0 0 1px var(--card-border-color);
  --popover-bg-color: #fcfcfc;
  --popover-fg-color: #232629;
  --popover-border-color: rgba(0, 0, 0, 0.13);
  --popover-shadow: 0 6px 20px rgba(0, 0, 0, 0.14), 0 1px 3px rgba(0, 0, 0, 0.08), 0 0 0 1px var(--popover-border-color);
  --popover-hover-bg: rgba(61, 174, 233, 0.14);
  --popover-active-bg: rgba(61, 174, 233, 0.22);
  --toast-bg: rgba(35, 38, 41, 0.94);
  --select-chevron: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 16 16' fill='none' stroke='%23232629' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M4 6l4 4 4-4'/%3E%3C/svg%3E");
}
`
