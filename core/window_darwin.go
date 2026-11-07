//go:build darwin && cgo

package core

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#include "window_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

var (
	activeDarwinWindow *Window
	darwinNotifyMu     sync.Mutex
)

func init() {
	if os.Getenv("BRISA_MACOS_NOTIFY_HELPER") == "1" {
		runDarwinNotificationHelperAndExit()
	}
	runtime.LockOSThread()
}

func runDarwinNotificationHelperAndExit() {
	runtime.LockOSThread()
	cTitle := C.CString(os.Getenv("BRISA_NOTIFY_TITLE"))
	cSubtitle := C.CString(os.Getenv("BRISA_NOTIFY_SUBTITLE"))
	cBody := C.CString(os.Getenv("BRISA_NOTIFY_BODY"))
	res := C.brisaRunNotificationHelper(cTitle, cSubtitle, cBody)
	C.free(unsafe.Pointer(cTitle))
	C.free(unsafe.Pointer(cSubtitle))
	C.free(unsafe.Pointer(cBody))
	if res == 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

//export brisaOnNativeWindowClosed
func brisaOnNativeWindowClosed() {
	if activeDarwinWindow != nil {
		_ = activeDarwinWindow.Close()
	}
}

func (w *Window) launch(targetURL string) error {
	if w.cfg.Headless {
		return nil
	}

	activeDarwinWindow = w

	cTitle := C.CString(w.cfg.Title)
	defer C.free(unsafe.Pointer(cTitle))

	cURL := C.CString(targetURL)
	defer C.free(unsafe.Pointer(cURL))

	var iconPtr unsafe.Pointer
	var iconLen C.int
	if w.app != nil && len(w.app.iconBytes) > 0 {
		iconPtr = unsafe.Pointer(&w.app.iconBytes[0])
		iconLen = C.int(len(w.app.iconBytes))
	}

	boolToInt := func(b bool) C.int {
		if b {
			return 1
		}
		return 0
	}

	C.brisaCreateWindow(
		cTitle,
		C.int(w.cfg.Width),
		C.int(w.cfg.Height),
		C.int(w.cfg.X),
		C.int(w.cfg.Y),
		boolToInt(w.cfg.Center),
		boolToInt(w.cfg.Fullscreen),
		boolToInt(w.cfg.Frameless),
		boolToInt(w.cfg.AlwaysOnTop),
		cURL,
		iconPtr,
		iconLen,
	)

	if w.cfg.MinWidth > 0 || w.cfg.MinHeight > 0 {
		w.setPlatformMinSize(w.cfg.MinWidth, w.cfg.MinHeight)
	}

	return nil
}

func (w *Window) runLoop(ctx context.Context) {
	if w.cfg.Headless {
		<-ctx.Done()
		return
	}

	go func() {
		select {
		case <-ctx.Done():
			C.brisaStopLoop()
		case <-w.closed:
			C.brisaStopLoop()
		}
	}()

	C.brisaRunLoop()
}

func (w *Window) setPlatformTitle(title string) {
	cStr := C.CString(title)
	defer C.free(unsafe.Pointer(cStr))
	C.brisaSetTitle(cStr)
}

func (w *Window) setPlatformSize(width, height int) {
	C.brisaSetSize(C.int(width), C.int(height))
}

func (w *Window) setPlatformMinSize(width, height int) {
	C.brisaSetMinSize(C.int(width), C.int(height))
}

func (w *Window) setPlatformPosition(x, y int) {
	C.brisaSetPosition(C.int(x), C.int(y))
}

func (w *Window) minimizePlatform() {
	C.brisaMinimize()
}

func (w *Window) maximizePlatform() {
	C.brisaMaximize()
}

func (w *Window) restorePlatform() {
	C.brisaRestore()
}

func (w *Window) setPlatformFullscreen(fs bool) {
	if fs {
		C.brisaSetFullscreen(1)
	} else {
		C.brisaSetFullscreen(0)
	}
}

func (w *Window) evalPlatform(js string) {
	cStr := C.CString(js)
	defer C.free(unsafe.Pointer(cStr))
	C.brisaEval(cStr)
}

func preparePlatformDesktop(app *App) {
	if app == nil || (app.window != nil && app.window.cfg.Headless) {
		return
	}
	if len(app.iconBytes) == 0 {
		return
	}
	go func() {
		_, _ = ensureDarwinNotifyBundle(app, app.iconBytes)
	}()
}

func notifyPlatform(app *App, opts NotificationOptions) bool {
	var iconBytes []byte
	if app != nil {
		iconBytes = app.iconBytes
	}
	if opts.Icon != "" && (app == nil || opts.Icon != app.iconPath) {
		var customBytes []byte
		if app != nil {
			customBytes, _ = resolveAppIcon(opts.Icon, app.config.Assets, app.AssetsFS)
		} else {
			customBytes, _ = resolveAppIcon(opts.Icon, nil, nil)
		}
		if len(customBytes) > 0 {
			iconBytes = customBytes
		}
	}

	if len(iconBytes) > 0 {
		if helperBin, err := ensureDarwinNotifyBundle(app, iconBytes); err == nil && helperBin != "" {
			cmd := exec.Command(helperBin)
			cmd.Env = append(os.Environ(),
				"BRISA_MACOS_NOTIFY_HELPER=1",
				"BRISA_NOTIFY_TITLE="+opts.Title,
				"BRISA_NOTIFY_SUBTITLE="+opts.Subtitle,
				"BRISA_NOTIFY_BODY="+opts.Body,
			)
			if err := cmd.Start(); err == nil {
				go func() { _ = cmd.Wait() }()
				return true
			}
		}
	}

	cTitle := C.CString(opts.Title)
	cSubtitle := C.CString(opts.Subtitle)
	cBody := C.CString(opts.Body)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cSubtitle))
	defer C.free(unsafe.Pointer(cBody))

	return C.brisaDeliverNotificationInProcess(cTitle, cSubtitle, cBody) != 0
}

func ensureDarwinNotifyBundle(app *App, iconBytes []byte) (string, error) {
	if app == nil {
		return "", fmt.Errorf("brisa darwin notify: app is nil")
	}

	darwinNotifyMu.Lock()
	defer darwinNotifyMu.Unlock()

	exePath, err := os.Executable()
	if err != nil || exePath == "" {
		return "", fmt.Errorf("brisa darwin notify: executable path unavailable: %w", err)
	}

	appID := strings.TrimSpace(app.config.AppID)
	appTitle := strings.TrimSpace(app.config.Title)
	appVersion := strings.TrimSpace(app.config.Version)

	icnsBytes := encodeAppleICNS(iconBytes)

	h := sha256.New()
	_, _ = io.WriteString(h, appID+"\x00"+appTitle+"\x00")
	_, _ = h.Write(icnsBytes)
	bundleHash := hex.EncodeToString(h.Sum(nil))[:12]

	baseCache, err := os.UserCacheDir()
	if err != nil || baseCache == "" {
		baseCache = os.TempDir()
	}
	bundleRoot := filepath.Join(baseCache, "brisa", "notify-bundles", bundleHash)
	if err := os.MkdirAll(bundleRoot, 0755); err != nil {
		bundleRoot = filepath.Join(os.TempDir(), "brisa-notify-bundles", bundleHash)
	}
	safeAppName := sanitizeMacBundleName(appTitle)
	bundleDir := filepath.Join(bundleRoot, safeAppName+".app")
	contentsDir := filepath.Join(bundleDir, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	resourcesDir := filepath.Join(contentsDir, "Resources")
	helperBin := filepath.Join(macOSDir, "brisa-notify")
	readyMarker := filepath.Join(bundleRoot, ".ready")

	if _, err := os.Stat(readyMarker); err == nil {
		if _, err := os.Stat(helperBin); err == nil {
			return helperBin, nil
		}
	}

	_ = os.RemoveAll(bundleDir)
	if err := os.MkdirAll(macOSDir, 0755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		return "", err
	}

	if len(icnsBytes) > 0 {
		if err := os.WriteFile(filepath.Join(resourcesDir, "app.icns"), icnsBytes, 0644); err != nil {
			return "", err
		}
	}

	notifyBundleID := fmt.Sprintf("%s.notify.%s", appID, bundleHash[:8])
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>%s</string>
    <key>CFBundleDisplayName</key>
    <string>%s</string>
    <key>CFBundleIdentifier</key>
    <string>%s</string>
    <key>CFBundleVersion</key>
    <string>%s</string>
    <key>CFBundleShortVersionString</key>
    <string>%s</string>
    <key>CFBundleExecutable</key>
    <string>brisa-notify</string>
    <key>CFBundleIconFile</key>
    <string>app.icns</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSUserNotificationAlertStyle</key>
    <string>banner</string>
</dict>
</plist>
`, xmlEscapePlist(appTitle), xmlEscapePlist(appTitle), xmlEscapePlist(notifyBundleID), xmlEscapePlist(appVersion), xmlEscapePlist(appVersion))

	if err := os.WriteFile(filepath.Join(contentsDir, "Info.plist"), []byte(plist), 0644); err != nil {
		return "", err
	}
	_ = os.WriteFile(filepath.Join(contentsDir, "PkgInfo"), []byte("APPL????"), 0644)

	if err := copyFileExecutable(exePath, helperBin); err != nil {
		return "", err
	}

	if cs, err := exec.LookPath("codesign"); err == nil {
		_ = exec.Command(cs, "--force", "--deep", "--sign", "-", bundleDir).Run()
	}

	const lsregisterPath = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	if _, err := os.Stat(lsregisterPath); err == nil {
		_ = exec.Command(lsregisterPath, "-f", bundleDir).Run()
	}

	_ = os.WriteFile(readyMarker, []byte("ok"), 0644)
	return helperBin, nil
}

func copyFileExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func encodeAppleICNS(iconBytes []byte) []byte {
	if len(iconBytes) == 0 {
		return nil
	}
	if len(iconBytes) >= 4 && string(iconBytes[:4]) == "icns" {
		return iconBytes
	}
	pngData := iconBytes
	if len(pngData) < 8 || !bytes.Equal(pngData[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		img, _, err := image.Decode(bytes.NewReader(iconBytes))
		if err != nil {
			return nil
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil
		}
		pngData = buf.Bytes()
	}

	chunkTypes := []string{"ic07", "ic08", "ic09", "ic10"}
	chunkLen := 8 + len(pngData)
	totalLen := 8 + len(chunkTypes)*chunkLen

	out := make([]byte, 0, totalLen)
	out = append(out, 'i', 'c', 'n', 's')
	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], uint32(totalLen))
	out = append(out, sizeBuf[:]...)

	for _, ct := range chunkTypes {
		out = append(out, ct[0], ct[1], ct[2], ct[3])
		binary.BigEndian.PutUint32(sizeBuf[:], uint32(chunkLen))
		out = append(out, sizeBuf[:]...)
		out = append(out, pngData...)
	}
	return out
}

func sanitizeMacBundleName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r == '/' || r == ':' || r == '\\' || r < 32 {
			continue
		}
		b.WriteRune(r)
	}
	res := strings.TrimSpace(b.String())
	if res == "" {
		return "Brisa"
	}
	return res
}

var plistEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&apos;",
)

func xmlEscapePlist(s string) string {
	return plistEscaper.Replace(s)
}
