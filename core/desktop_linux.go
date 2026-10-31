package core

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ensureLinuxDesktopEntry ensures that on Linux systems running Wayland or X11,
// a desktop entry file exists under ~/.local/share/applications/ matching cfg.AppID,
// so that window managers and docks (GNOME Shell, Ubuntu Dock, KDE Plasma, etc.)
// can associate the window's WM_CLASS / app_id with the configured desktop icon and metadata.
func ensureLinuxDesktopEntry(cfg Config, iconBytes []byte, iconPath string) {
	if runtime.GOOS != "linux" {
		return
	}
	appID := strings.TrimSpace(cfg.AppID)
	if appID == "" {
		return
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	appsDir := filepath.Join(dataHome, "applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return
	}

	resolvedIconPath := iconPath
	if resolvedIconPath == "" || !filepath.IsAbs(resolvedIconPath) {
		if len(iconBytes) > 0 {
			iconsDir := filepath.Join(dataHome, "icons", "hicolor", "512x512", "apps")
			_ = os.MkdirAll(iconsDir, 0755)
			targetIcon := filepath.Join(iconsDir, appID+".png")
			_ = os.WriteFile(targetIcon, iconBytes, 0644)
			resolvedIconPath = targetIcon
		} else if iconPath != "" {
			if abs, err := filepath.Abs(iconPath); err == nil {
				resolvedIconPath = abs
			}
		}
	}

	execPath, err := os.Executable()
	if err != nil || execPath == "" {
		execPath = "/usr/bin/env true"
	}

	title := cfg.Title
	if title == "" {
		title = appID
	}
	desc := cfg.Description
	if desc == "" {
		desc = title
	}

	desktopFile := filepath.Join(appsDir, appID+".desktop")

	var sb strings.Builder
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Version=1.0\n")
	sb.WriteString("Type=Application\n")
	sb.WriteString(fmt.Sprintf("Name=%s\n", title))
	sb.WriteString(fmt.Sprintf("Comment=%s\n", desc))
	if resolvedIconPath != "" {
		sb.WriteString(fmt.Sprintf("Icon=%s\n", resolvedIconPath))
	}
	sb.WriteString(fmt.Sprintf("Exec=%s\n", execPath))
	sb.WriteString("Terminal=false\n")
	sb.WriteString(fmt.Sprintf("StartupWMClass=%s\n", appID))
	sb.WriteString("StartupNotify=true\n")
	sb.WriteString("Categories=Utility;\n")

	_ = os.WriteFile(desktopFile, []byte(sb.String()), 0644)

	if upd, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command(upd, appsDir).Run()
	}
}

// setX11WindowIcon connects directly to the local X11 display (if running under X11)
// and sets the _NET_WM_ICON property on the client window with the supplied PNG/JPEG icon bytes,
// ensuring the window manager, alt-tab switcher, and taskbar display the custom icon immediately.
func setX11WindowIcon(appID string, iconBytes []byte) {
	if runtime.GOOS != "linux" || len(iconBytes) == 0 {
		return
	}
	display := os.Getenv("DISPLAY")
	if display == "" {
		return
	}

	img, _, err := image.Decode(bytes.NewReader(iconBytes))
	if err != nil {
		return
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return
	}

	// Limit to max 128x128 for X11 property payload if overly huge
	if w > 128 || h > 128 {
		img = downscaleImage(img, 64, 64)
		bounds = img.Bounds()
		w, h = bounds.Dx(), bounds.Dy()
	}

	pixels := make([]uint32, 2+w*h)
	pixels[0] = uint32(w)
	pixels[1] = uint32(h)
	idx := 2
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// image.RGBA returns alpha-premultiplied 16-bit values [0..0xffff]
			// Convert to 8-bit ARGB
			a8 := uint32(a >> 8)
			r8 := uint32(r >> 8)
			g8 := uint32(g >> 8)
			b8 := uint32(b >> 8)
			pixels[idx] = (a8 << 24) | (r8 << 16) | (g8 << 8) | b8
			idx++
		}
	}

	go func() {
		// Poll briefly to find the created window matching appID
		for i := 0; i < 20; i++ {
			time.Sleep(100 * time.Millisecond)
			wid := findX11WindowID(appID)
			if wid != 0 {
				_ = applyX11IconProperty(display, wid, pixels)
				return
			}
		}
	}()
}

func downscaleImage(src image.Image, targetW, targetH int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := bounds.Min.X + (x*srcW)/targetW
			srcY := bounds.Min.Y + (y*srcH)/targetH
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func findX11WindowID(appID string) uint32 {
	xprop, err := exec.LookPath("xprop")
	if err != nil {
		return 0
	}
	out, err := exec.Command(xprop, "-root", "_NET_CLIENT_LIST").Output()
	if err != nil {
		return 0
	}
	parts := strings.Split(string(out), "#")
	if len(parts) < 2 {
		return 0
	}
	wids := strings.Split(parts[1], ",")
	for _, raw := range wids {
		wStr := strings.TrimSpace(raw)
		if wStr == "" {
			continue
		}
		info, err := exec.Command(xprop, "-id", wStr, "WM_CLASS").Output()
		if err == nil && strings.Contains(string(info), appID) {
			var wid uint32
			_, _ = fmt.Sscanf(wStr, "0x%x", &wid)
			if wid == 0 {
				_, _ = fmt.Sscanf(wStr, "%d", &wid)
			}
			return wid
		}
	}
	return 0
}

func applyX11IconProperty(display string, wid uint32, pixels []uint32) error {
	dispNum := "0"
	if idx := strings.Index(display, ":"); idx != -1 {
		dispNum = strings.Split(display[idx+1:], ".")[0]
	}

	socketPath := "/tmp/.X11-unix/X" + dispNum
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	authName, authCookie := getXAuthority()
	pad := func(n int) int { return (4 - (n % 4)) % 4 }

	nameLen := len(authName)
	cookieLen := len(authCookie)
	setupReq := make([]byte, 12+nameLen+pad(nameLen)+cookieLen+pad(cookieLen))
	setupReq[0] = 0x6c // Little-endian 'l'
	setupReq[2] = 11   // Protocol major 11
	setupReq[6] = byte(nameLen)
	setupReq[7] = byte(nameLen >> 8)
	setupReq[8] = byte(cookieLen)
	setupReq[9] = byte(cookieLen >> 8)

	offset := 12
	copy(setupReq[offset:], authName)
	offset += nameLen + pad(nameLen)
	copy(setupReq[offset:], authCookie)

	if _, err := conn.Write(setupReq); err != nil {
		return err
	}

	replyHdr := make([]byte, 8)
	if _, err := conn.Read(replyHdr); err != nil || replyHdr[0] != 1 {
		return fmt.Errorf("x11 setup failed")
	}
	addLen := uint16(replyHdr[6]) | (uint16(replyHdr[7]) << 8)
	discard := make([]byte, int(addLen)*4)
	if _, err := conn.Read(discard); err != nil {
		return err
	}

	// InternAtom "_NET_WM_ICON" (Opcode 16)
	atomName := []byte("_NET_WM_ICON")
	aLen := len(atomName)
	reqWords := 2 + (aLen+pad(aLen))/4
	atomReq := make([]byte, reqWords*4)
	atomReq[0] = 16 // InternAtom
	atomReq[2] = byte(reqWords)
	atomReq[3] = byte(reqWords >> 8)
	atomReq[4] = byte(aLen)
	atomReq[5] = byte(aLen >> 8)
	copy(atomReq[8:], atomName)

	if _, err := conn.Write(atomReq); err != nil {
		return err
	}
	atomReply := make([]byte, 32)
	if _, err := conn.Read(atomReply); err != nil {
		return err
	}
	atomID := uint32(atomReply[8]) | (uint32(atomReply[9]) << 8) | (uint32(atomReply[10]) << 16) | (uint32(atomReply[11]) << 24)
	if atomID == 0 {
		return fmt.Errorf("intern atom failed")
	}

	// ChangeProperty (Opcode 18)
	// mode = 0 (Replace), format = 32, type = 6 (CARDINAL)
	totalWords := 6 + len(pixels)
	cpReq := make([]byte, totalWords*4)
	cpReq[0] = 18 // ChangeProperty
	cpReq[1] = 0  // Replace
	cpReq[2] = byte(totalWords)
	cpReq[3] = byte(totalWords >> 8)

	// Window ID
	cpReq[4] = byte(wid)
	cpReq[5] = byte(wid >> 8)
	cpReq[6] = byte(wid >> 16)
	cpReq[7] = byte(wid >> 24)

	// Property atom
	cpReq[8] = byte(atomID)
	cpReq[9] = byte(atomID >> 8)
	cpReq[10] = byte(atomID >> 16)
	cpReq[11] = byte(atomID >> 24)

	// Type atom (6 = CARDINAL)
	cpReq[12] = 6

	// Format = 32
	cpReq[16] = 32

	// Length of data in format units (number of 32-bit elements)
	numElems := uint32(len(pixels))
	cpReq[20] = byte(numElems)
	cpReq[21] = byte(numElems >> 8)
	cpReq[22] = byte(numElems >> 16)
	cpReq[23] = byte(numElems >> 24)

	for i, px := range pixels {
		pOff := 24 + i*4
		cpReq[pOff] = byte(px)
		cpReq[pOff+1] = byte(px >> 8)
		cpReq[pOff+2] = byte(px >> 16)
		cpReq[pOff+3] = byte(px >> 24)
	}

	if _, err := conn.Write(cpReq); err != nil {
		return err
	}

	// Flush and sync via GetInputFocus (Opcode 43)
	syncReq := []byte{43, 0, 1, 0}
	_, _ = conn.Write(syncReq)
	syncReply := make([]byte, 32)
	_, _ = conn.Read(syncReply)

	return nil
}

func getXAuthority() ([]byte, []byte) {
	path := os.Getenv("XAUTHORITY")
	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, ".Xauthority")
		}
	}
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	i := 0
	for i+4 <= len(data) {
		i += 2 // family
		alen := int(data[i])<<8 | int(data[i+1])
		i += 2 + alen
		if i+2 > len(data) {
			break
		}
		nlen := int(data[i])<<8 | int(data[i+1])
		i += 2 + nlen
		if i+2 > len(data) {
			break
		}
		klen := int(data[i])<<8 | int(data[i+1])
		i += 2
		if i+klen > len(data) {
			break
		}
		name := data[i : i+klen]
		i += klen
		if i+2 > len(data) {
			break
		}
		dlen := int(data[i])<<8 | int(data[i+1])
		i += 2
		if i+dlen > len(data) {
			break
		}
		cookie := data[i : i+dlen]
		i += dlen
		if string(name) == "MIT-MAGIC-COOKIE-1" {
			return name, cookie
		}
	}
	return nil, nil
}
