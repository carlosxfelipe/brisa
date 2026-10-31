//go:build darwin && cgo

package core

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#include "window_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"runtime"
	"unsafe"
)

var activeDarwinWindow *Window

func init() {
	runtime.LockOSThread()
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
