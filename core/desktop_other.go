//go:build !linux

package core

// ensureLinuxDesktopEntry is a no-op on non-Linux platforms.
func ensureLinuxDesktopEntry(cfg Config, iconBytes []byte, iconPath string) {}

// setX11WindowIcon is a no-op on non-Linux platforms.
func setX11WindowIcon(appID string, iconBytes []byte) {}
