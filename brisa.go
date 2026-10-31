package brisa

import (
	"time"

	"github.com/carlosxfelipe/brisa/core"
	"github.com/carlosxfelipe/brisa/validation"
)

// Re-exported Core & Validation Types
type (
	App                  = core.App
	Config               = core.Config
	Theme                = core.Theme
	WindowConfig         = core.WindowConfig
	Window               = core.Window
	WindowState          = core.WindowState
	Context              = core.Context
	Command              = core.Command
	CommandGroup         = core.CommandGroup
	CommandDoc           = core.CommandDoc
	CommandSchema        = core.CommandSchema
	FieldSchema          = core.FieldSchema
	InspectorSpec        = core.InspectorSpec
	DispatchResult       = core.DispatchResult
	HandlerFunc          = core.HandlerFunc
	MiddlewareFunc       = core.MiddlewareFunc
	EventHandler         = core.EventHandler
	Map                  = core.Map
	Empty                = core.Empty
	RPCError             = core.RPCError
	ErrorResponse        = core.ErrorResponse
	ValidationError      = validation.ValidationError
	Validator            = validation.Validator
	ContextValidator     = validation.ContextValidator
	Store                = core.Store
	StoreOptions         = core.StoreOptions
	Dialog               = core.Dialog
	FileFilter           = core.FileFilter
	FileDialogOptions    = core.FileDialogOptions
	MessageDialogOptions = core.MessageDialogOptions
	NotificationOptions  = core.NotificationOptions
	Clipboard            = core.Clipboard
	RequestIDOptions     = core.RequestIDOptions
	SlogLoggerOptions    = core.SlogLoggerOptions
	RateLimiterOptions   = core.RateLimiterOptions
)

// Re-exported WindowState & Theme Constants
const (
	WindowStateNormal     = core.WindowStateNormal
	WindowStateMinimized  = core.WindowStateMinimized
	WindowStateMaximized  = core.WindowStateMaximized
	WindowStateFullscreen = core.WindowStateFullscreen

	ThemeAdwaita = core.ThemeAdwaita
	ThemeUbuntu  = core.ThemeUbuntu
	ThemePlasma  = core.ThemePlasma
	ThemeMacOS   = core.ThemeMacOS
	ThemeWindows = core.ThemeWindows
)

// New initializes and returns a new Brisa desktop application instance.
func New(cfg Config) *App {
	if cfg.SilenceTagWarnings {
		validation.SilenceTagWarnings = true
	}
	return core.New(cfg)
}

// NewError creates a structured RPCError with a status code, machine-readable code, and message.
func NewError(status int, code, message string) *RPCError {
	return core.NewError(status, code, message)
}

// SilenceTagWarnings enables or disables startup warning logs for suspicious validation tags.
func SilenceTagWarnings(silence bool) {
	validation.SilenceTagWarnings = silence
}

// LocateBrowser finds an installed Chromium/Chrome/Edge/Brave binary for app-mode desktop rendering.
func LocateBrowser() string {
	return core.LocateBrowser()
}

// OpenExternal opens a URL or file path in the operating system's default application.
func OpenExternal(target string) error {
	return core.OpenExternal(target)
}

// Built-in Middleware Helpers

// Recover recovers from panics inside IPC command handlers and returns a 500 INTERNAL_PANIC error.
func Recover() MiddlewareFunc {
	return core.Recover()
}

// Logger logs IPC command execution latency and status using the standard log package.
func Logger() MiddlewareFunc {
	return core.Logger()
}

// SlogLogger logs IPC command execution using structured log/slog.
func SlogLogger(opts ...SlogLoggerOptions) MiddlewareFunc {
	return core.SlogLogger(opts...)
}

// RequestID ensures every IPC command invocation has a correlation ID.
func RequestID(opts ...RequestIDOptions) MiddlewareFunc {
	return core.RequestID(opts...)
}

// Timeout enforces an execution deadline on IPC commands.
func Timeout(d time.Duration) MiddlewareFunc {
	return core.Timeout(d)
}

// RateLimiter limits the rate of IPC command calls per time window.
func RateLimiter(opts RateLimiterOptions) MiddlewareFunc {
	return core.RateLimiter(opts)
}
