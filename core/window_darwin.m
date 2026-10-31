//go:build darwin && cgo

#import "window_darwin.h"

extern void brisaOnNativeWindowClosed(void);

@interface BrisaNativeWindowDelegate : NSObject <NSWindowDelegate, WKUIDelegate, WKNavigationDelegate>
@end

@implementation BrisaNativeWindowDelegate
- (void)windowWillClose:(NSNotification *)notification {
    brisaOnNativeWindowClosed();
}

- (WKWebView *)webView:(WKWebView *)webView
createWebViewWithConfiguration:(WKWebViewConfiguration *)configuration
   forNavigationAction:(WKNavigationAction *)navigationAction
        windowFeatures:(WKWindowFeatures *)windowFeatures {
    NSURL *url = navigationAction.request.URL;
    if (url) {
        [[NSWorkspace sharedWorkspace] openURL:url];
    }
    return nil;
}

- (void)webView:(WKWebView *)webView
decidePolicyForNavigationAction:(WKNavigationAction *)navigationAction
decisionHandler:(void (^)(WKNavigationActionPolicy))decisionHandler {
    if (navigationAction.targetFrame == nil) {
        NSURL *url = navigationAction.request.URL;
        if (url) {
            [[NSWorkspace sharedWorkspace] openURL:url];
        }
        decisionHandler(WKNavigationActionPolicyCancel);
        return;
    }
    decisionHandler(WKNavigationActionPolicyAllow);
}
@end

static NSWindow *gBrisaWindow = nil;
static WKWebView *gBrisaWebView = nil;
static BrisaNativeWindowDelegate *gBrisaDelegate = nil;

void brisaCreateWindow(const char* title, int width, int height, int x, int y, int center, int fullscreen, int frameless, int alwaysOnTop, const char* urlStr, const void* iconBytes, int iconLen) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];

        if (iconBytes && iconLen > 0) {
            NSData *data = [NSData dataWithBytes:iconBytes length:iconLen];
            NSImage *img = [[NSImage alloc] initWithData:data];
            if (img) {
                [NSApp setApplicationIconImage:img];
            }
        }

        NSRect frame = NSMakeRect(x, y, width, height);
        NSWindowStyleMask style = NSWindowStyleMaskTitled |
                                  NSWindowStyleMaskClosable |
                                  NSWindowStyleMaskMiniaturizable |
                                  NSWindowStyleMaskResizable;
        if (frameless) {
            style = NSWindowStyleMaskBorderless | NSWindowStyleMaskResizable;
        }

        gBrisaWindow = [[NSWindow alloc] initWithContentRect:frame
                                                   styleMask:style
                                                     backing:NSBackingStoreBuffered
                                                       defer:NO];

        if (title) {
            [gBrisaWindow setTitle:[NSString stringWithUTF8String:title]];
        }
        if (center) {
            [gBrisaWindow center];
        }
        if (alwaysOnTop) {
            [gBrisaWindow setLevel:NSFloatingWindowLevel];
        }

        gBrisaDelegate = [[BrisaNativeWindowDelegate alloc] init];
        [gBrisaWindow setDelegate:gBrisaDelegate];

        WKWebViewConfiguration *config = [[WKWebViewConfiguration alloc] init];
        [config.preferences setValue:@YES forKey:@"developerExtrasEnabled"];

        gBrisaWebView = [[WKWebView alloc] initWithFrame:frame configuration:config];
        [gBrisaWebView setUIDelegate:gBrisaDelegate];
        [gBrisaWebView setNavigationDelegate:gBrisaDelegate];

        if (@available(macOS 13.3, *)) {
            [gBrisaWebView setInspectable:YES];
        }

        [gBrisaWindow setContentView:gBrisaWebView];

        if (urlStr) {
            NSURL *url = [NSURL URLWithString:[NSString stringWithUTF8String:urlStr]];
            [gBrisaWebView loadRequest:[NSURLRequest requestWithURL:url]];
        }

        if (fullscreen) {
            [gBrisaWindow toggleFullScreen:nil];
        }

        [gBrisaWindow makeKeyAndOrderFront:nil];
        [NSApp activateIgnoringOtherApps:YES];
    }
}

void brisaRunLoop(void) {
    @autoreleasepool {
        [NSApp run];
    }
}

void brisaStopLoop(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            [gBrisaWindow close];
            gBrisaWindow = nil;
        }
        [NSApp stop:nil];
        NSEvent* event = [NSEvent otherEventWithType:NSEventTypeApplicationDefined
                                            location:NSMakePoint(0, 0)
                                       modifierFlags:0
                                           timestamp:0
                                        windowNumber:0
                                             context:nil
                                             subtype:0
                                               data1:0
                                               data2:0];
        [NSApp postEvent:event atStart:YES];
    });
}

void brisaSetTitle(const char* title) {
    if (!title) return;
    NSString *nsTitle = [NSString stringWithUTF8String:title];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            [gBrisaWindow setTitle:nsTitle];
        }
    });
}

void brisaSetSize(int width, int height) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            [gBrisaWindow setContentSize:NSMakeSize(width, height)];
        }
    });
}

void brisaSetPosition(int x, int y) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            NSScreen *screen = [gBrisaWindow screen];
            if (screen) {
                NSRect screenFrame = [screen frame];
                CGFloat flippedY = screenFrame.size.height - y;
                [gBrisaWindow setFrameTopLeftPoint:NSMakePoint(x, flippedY)];
            }
        }
    });
}

void brisaMinimize(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            [gBrisaWindow miniaturize:nil];
        }
    });
}

void brisaMaximize(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            [gBrisaWindow zoom:nil];
        }
    });
}

void brisaRestore(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            if ([gBrisaWindow isMiniaturized]) {
                [gBrisaWindow deminiaturize:nil];
            } else if ([gBrisaWindow isZoomed]) {
                [gBrisaWindow zoom:nil];
            }
        }
    });
}

void brisaSetFullscreen(int fullscreen) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWindow) {
            BOOL isFS = (([gBrisaWindow styleMask] & NSWindowStyleMaskFullScreen) != 0);
            if ((fullscreen && !isFS) || (!fullscreen && isFS)) {
                [gBrisaWindow toggleFullScreen:nil];
            }
        }
    });
}

void brisaEval(const char* js) {
    if (!js) return;
    NSString *nsJS = [NSString stringWithUTF8String:js];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (gBrisaWebView) {
            [gBrisaWebView evaluateJavaScript:nsJS completionHandler:nil];
        }
    });
}

