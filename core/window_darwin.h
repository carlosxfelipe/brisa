//go:build darwin && cgo

#ifndef BRISA_WINDOW_DARWIN_H
#define BRISA_WINDOW_DARWIN_H

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

void brisaCreateWindow(const char* title, int width, int height, int x, int y, int center, int fullscreen, int frameless, int alwaysOnTop, const char* urlStr, const void* iconBytes, int iconLen);
void brisaRunLoop(void);
void brisaStopLoop(void);
void brisaSetTitle(const char* title);
void brisaSetSize(int width, int height);
void brisaSetMinSize(int minWidth, int minHeight);
void brisaSetPosition(int x, int y);
void brisaMinimize(void);
void brisaMaximize(void);
void brisaRestore(void);
void brisaSetFullscreen(int fullscreen);
void brisaEval(const char* js);

#endif // BRISA_WINDOW_DARWIN_H

