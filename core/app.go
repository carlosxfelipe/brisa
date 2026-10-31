package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Config holds the configuration for a Brisa desktop application.
type Config struct {
	AppID              string
	Title              string
	Version            string
	Description        string
	Icon               string
	Addr               string
	Assets             fs.FS
	AssetsSubDir       string
	IndexHTML          string
	DevServerURL       string
	InspectorURL       string
	Theme              Theme
	DisableAdwaitaCSS  bool
	DisableCSRFToken   bool
	SilenceTagWarnings bool
	Window             WindowConfig
	Store              StoreOptions
}

// Command represents a registered IPC command in Brisa and provides fluent metadata methods.
type Command struct {
	name        string
	group       string
	handler     HandlerFunc
	middlewares []MiddlewareFunc
	doc         CommandDoc
	internal    bool
}

// Summary sets a short summary for the command in the IPC Inspector and TypeScript SDK.
func (c *Command) Summary(s string) *Command {
	c.doc.Summary = s
	return c
}

// Description sets a detailed description for the command.
func (c *Command) Description(d string) *Command {
	c.doc.Description = d
	return c
}

// Tags attaches one or more category tags to the command.
func (c *Command) Tags(tags ...string) *Command {
	c.doc.Tags = append(c.doc.Tags, tags...)
	return c
}

// Deprecated marks the command as deprecated in the IPC schema.
func (c *Command) Deprecated() *Command {
	c.doc.Deprecated = true
	return c
}

// Middleware attaches command-specific middlewares that run after global/group middlewares.
func (c *Command) Middleware(mw ...MiddlewareFunc) *Command {
	c.middlewares = append(c.middlewares, mw...)
	return c
}

// Doc returns the CommandDoc metadata for this command.
func (c *Command) Doc() CommandDoc {
	return c.doc
}

// CommandGroup groups related IPC commands under a common namespace prefix (e.g., "tasks") and shared middlewares.
type CommandGroup struct {
	prefix      string
	app         *App
	middlewares []MiddlewareFunc
	tags        []string
}

// Group creates a nested CommandGroup with an additional dot-separated namespace prefix.
func (g *CommandGroup) Group(prefix string, mw ...MiddlewareFunc) *CommandGroup {
	clean := strings.Trim(prefix, ".")
	combined := g.prefix
	if clean != "" {
		if combined != "" {
			combined += "." + clean
		} else {
			combined = clean
		}
	}
	mergedMW := make([]MiddlewareFunc, 0, len(g.middlewares)+len(mw))
	mergedMW = append(mergedMW, g.middlewares...)
	mergedMW = append(mergedMW, mw...)
	return &CommandGroup{
		prefix:      combined,
		app:         g.app,
		middlewares: mergedMW,
		tags:        append([]string(nil), g.tags...),
	}
}

// Use appends middlewares to the CommandGroup.
func (g *CommandGroup) Use(mw ...MiddlewareFunc) *CommandGroup {
	g.middlewares = append(g.middlewares, mw...)
	return g
}

// Tags sets default tags inherited by all commands registered in this group.
func (g *CommandGroup) Tags(tags ...string) *CommandGroup {
	g.tags = append(g.tags, tags...)
	return g
}

// RegisterCommand registers a low-level command handler within this group.
func (g *CommandGroup) RegisterCommand(name string, handler HandlerFunc, doc CommandDoc) *Command {
	clean := strings.Trim(name, ".")
	fullName := clean
	if g.prefix != "" {
		if clean != "" {
			fullName = g.prefix + "." + clean
		} else {
			fullName = g.prefix
		}
	}
	doc.Group = g.prefix
	if len(g.tags) > 0 {
		doc.Tags = append(append([]string(nil), g.tags...), doc.Tags...)
	}
	cmd := g.app.RegisterCommand(fullName, handler, doc)
	if len(g.middlewares) > 0 {
		cmd.middlewares = append(append([]MiddlewareFunc(nil), g.middlewares...), cmd.middlewares...)
	}
	return cmd
}

// DispatchResult holds the result of an IPC command execution.
type DispatchResult struct {
	Status    int            `json:"status"`
	Data      any            `json:"data,omitempty"`
	Error     *ErrorResponse `json:"error,omitempty"`
	RequestID string         `json:"requestId,omitempty"`
}

// EventHandler is a callback invoked when an event is emitted on the Brisa event bus.
type EventHandler func(data any)

type sseSubscriber struct {
	ch chan string
}

// App is the core Brisa desktop application instance.
type App struct {
	mu          sync.RWMutex
	config      Config
	token       string
	commands    map[string]*Command
	cmdOrder    []string
	middlewares []MiddlewareFunc

	window    *Window
	dialog    *Dialog
	clipboard *Clipboard
	store     *Store

	eventMu       sync.RWMutex
	eventHandlers map[string][]EventHandler
	sseMu         sync.RWMutex
	sseClients    map[*sseSubscriber]struct{}

	startupHooks     []func(ctx context.Context) error
	readyHooks       []func(app *App)
	beforeCloseHooks []func(app *App) bool
	shutdownHooks    []func(app *App)

	httpServer *http.Server
	listener   net.Listener
	baseURL    string
	AssetsFS   fs.FS
	iconBytes  []byte
	iconPath   string
}

// New creates and initializes a new Brisa desktop application.
func New(cfg Config) *App {
	if cfg.AppID == "" {
		cfg.AppID = "com.brisa.app"
	}
	if cfg.Title == "" {
		cfg.Title = "Brisa Desktop App"
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:0"
	}
	if cfg.InspectorURL == "" {
		cfg.InspectorURL = "/brisa/inspector"
	}
	cfg.Theme = NormalizeTheme(cfg.Theme)
	if cfg.Window.Title == "" {
		cfg.Window.Title = cfg.Title
	}

	var tokenBytes [24]byte
	_, _ = rand.Read(tokenBytes[:])
	token := hex.EncodeToString(tokenBytes[:])

	rawAssets := cfg.Assets
	assets := cfg.Assets
	if assets != nil && cfg.AssetsSubDir != "" {
		if sub, err := fs.Sub(assets, cfg.AssetsSubDir); err == nil {
			assets = sub
		}
	}

	iconBytes, iconPath := resolveAppIcon(cfg.Icon, rawAssets, assets)

	store, err := NewStore(cfg.AppID, cfg.Store)
	if err != nil {
		// Fall back to in-memory store if disk path is unavailable
		store, _ = NewStore(cfg.AppID, StoreOptions{MemoryOnly: true, EncryptionKey: cfg.Store.EncryptionKey})
	}

	app := &App{
		config:        cfg,
		token:         token,
		commands:      make(map[string]*Command),
		eventHandlers: make(map[string][]EventHandler),
		sseClients:    make(map[*sseSubscriber]struct{}),
		store:         store,
		AssetsFS:      assets,
		iconBytes:     iconBytes,
		iconPath:      iconPath,
	}
	app.window = newWindow(app, cfg.Window)
	app.dialog = &Dialog{app: app}
	app.clipboard = &Clipboard{app: app}

	app.registerBuiltinCommands()
	return app
}

// Config returns a copy of the application configuration.
func (a *App) Config() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// Theme returns the active desktop UI theme.
func (a *App) Theme() Theme {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Theme
}

// SetTheme updates the active desktop UI theme in Go.
func (a *App) SetTheme(theme Theme) {
	a.mu.Lock()
	a.config.Theme = NormalizeTheme(theme)
	a.mu.Unlock()
}

// Token returns the per-session security token used to authenticate IPC requests.
func (a *App) Token() string {
	return a.token
}

// BaseURL returns the bound loopback URL (e.g. "http://127.0.0.1:54321") once started.
func (a *App) BaseURL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.baseURL
}

// Window returns the desktop Window controller.
func (a *App) Window() *Window {
	return a.window
}

// Dialog returns the native OS Dialog controller.
func (a *App) Dialog() *Dialog {
	return a.dialog
}

// Clipboard returns the native OS Clipboard controller.
func (a *App) Clipboard() *Clipboard {
	return a.clipboard
}

// Store returns the persistent local key-value store.
func (a *App) Store() *Store {
	return a.store
}

// Notify sends a native desktop notification and emits a WebUI notification event.
func (a *App) Notify(opts NotificationOptions) error {
	return Notify(a, opts)
}

// Use registers one or more global IPC command middlewares.
func (a *App) Use(mw ...MiddlewareFunc) *App {
	a.mu.Lock()
	a.middlewares = append(a.middlewares, mw...)
	a.mu.Unlock()
	return a
}

// Group creates a new CommandGroup with the given dot-separated namespace prefix.
func (a *App) Group(prefix string, mw ...MiddlewareFunc) *CommandGroup {
	return &CommandGroup{
		prefix:      strings.Trim(prefix, "."),
		app:         a,
		middlewares: append([]MiddlewareFunc(nil), mw...),
	}
}

// RegisterCommand registers a command handler with the application.
func (a *App) RegisterCommand(name string, handler HandlerFunc, doc CommandDoc) *Command {
	name = strings.TrimSpace(strings.Trim(name, "."))
	if name == "" {
		panic("brisa: command name cannot be empty")
	}
	doc.Name = name
	cmd := &Command{
		name:     name,
		group:    doc.Group,
		handler:  handler,
		doc:      doc,
		internal: strings.HasPrefix(name, "$brisa."),
	}

	a.mu.Lock()
	if _, exists := a.commands[name]; !exists {
		a.cmdOrder = append(a.cmdOrder, name)
	}
	a.commands[name] = cmd
	a.mu.Unlock()
	return cmd
}

// OnStartup registers a hook executed before the loopback server and window start.
func (a *App) OnStartup(fn func(ctx context.Context) error) *App {
	a.mu.Lock()
	a.startupHooks = append(a.startupHooks, fn)
	a.mu.Unlock()
	return a
}

// OnReady registers a hook executed once the desktop window is launched and ready.
func (a *App) OnReady(fn func(app *App)) *App {
	a.mu.Lock()
	a.readyHooks = append(a.readyHooks, fn)
	a.mu.Unlock()
	return a
}

// OnBeforeClose registers a hook called before shutting down. Returning false cancels the close.
func (a *App) OnBeforeClose(fn func(app *App) bool) *App {
	a.mu.Lock()
	a.beforeCloseHooks = append(a.beforeCloseHooks, fn)
	a.mu.Unlock()
	return a
}

// OnShutdown registers a cleanup hook called when the application exits.
func (a *App) OnShutdown(fn func(app *App)) *App {
	a.mu.Lock()
	a.shutdownHooks = append(a.shutdownHooks, fn)
	a.mu.Unlock()
	return a
}

// On subscribes a Go callback to an event topic on the bidirectional event bus.
func (a *App) On(event string, handler EventHandler) {
	a.eventMu.Lock()
	a.eventHandlers[event] = append(a.eventHandlers[event], handler)
	a.eventMu.Unlock()
}

// Emit broadcasts an event with arbitrary JSON-serializable data to both Go subscribers
// and all connected desktop frontend windows via Server-Sent Events.
func (a *App) Emit(event string, data any) {
	a.eventMu.RLock()
	handlers := append([]EventHandler(nil), a.eventHandlers[event]...)
	a.eventMu.RUnlock()

	for _, fn := range handlers {
		func() {
			defer func() { _ = recover() }()
			fn(data)
		}()
	}

	payload, err := json.Marshal(Map{
		"event":     event,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return
	}
	msg := "data: " + string(payload) + "\n\n"

	a.sseMu.RLock()
	for sub := range a.sseClients {
		select {
		case sub.ch <- msg:
		default:
		}
	}
	a.sseMu.RUnlock()
}

// Dispatch invokes a registered IPC command directly in Go (useful for tests and internal orchestration).
func (a *App) Dispatch(ctx context.Context, command string, payload any) (*DispatchResult, error) {
	var raw json.RawMessage
	switch v := payload.(type) {
	case nil:
		raw = json.RawMessage(`{}`)
	case json.RawMessage:
		raw = v
	case []byte:
		raw = json.RawMessage(v)
	case string:
		if v == "" {
			raw = json.RawMessage(`{}`)
		} else {
			raw = json.RawMessage(v)
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("brisa dispatch marshal payload: %w", err)
		}
		raw = b
	}

	return a.executeCommand(ctx, command, "", raw, nil, nil), nil
}

func (a *App) executeCommand(stdCtx context.Context, command, reqID string, rawPayload json.RawMessage, r *http.Request, w http.ResponseWriter) *DispatchResult {
	if reqID == "" {
		reqID = defaultRequestID()
	}

	a.mu.RLock()
	cmd, exists := a.commands[command]
	globalMW := append([]MiddlewareFunc(nil), a.middlewares...)
	a.mu.RUnlock()

	if !exists {
		return &DispatchResult{
			Status:    404,
			RequestID: reqID,
			Error: &ErrorResponse{
				Code:      "COMMAND_NOT_FOUND",
				Message:   fmt.Sprintf("unknown command %q", command),
				Status:    404,
				Command:   command,
				RequestID: reqID,
			},
		}
	}

	chain := make([]HandlerFunc, 0, len(globalMW)+len(cmd.middlewares)+1)
	for _, mw := range globalMW {
		m := mw
		chain = append(chain, HandlerFunc(m))
	}
	for _, mw := range cmd.middlewares {
		m := mw
		chain = append(chain, HandlerFunc(m))
	}
	chain = append(chain, cmd.handler)

	c := newContext(stdCtx, a, command, reqID, rawPayload, chain, r, w)
	err := c.Next()
	if err != nil {
		errResp := normalizeError(err, command, reqID)
		return &DispatchResult{
			Status:    errResp.Status,
			RequestID: c.RequestID(),
			Error:     errResp,
		}
	}

	resData, status, _ := c.Result()
	if status == 0 {
		status = 200
	}
	return &DispatchResult{
		Status:    status,
		Data:      resData,
		RequestID: c.RequestID(),
	}
}

func normalizeError(err error, command, reqID string) *ErrorResponse {
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		status := rpcErr.Status
		if status == 0 {
			status = 400
		}
		code := rpcErr.Code
		if code == "" {
			code = "BRISA_ERROR"
		}
		return &ErrorResponse{
			Code:      code,
			Message:   rpcErr.Message,
			Status:    status,
			Command:   command,
			RequestID: reqID,
			Errors:    rpcErr.Errors,
			Details:   rpcErr.Details,
		}
	}

	// Support ValidationError or custom error types implementing HTTPStatus() and ValidationErrors()
	type validationErrCarrier interface {
		HTTPStatus() int
		FieldErrors() map[string][]string
	}
	var ve validationErrCarrier
	if errors.As(err, &ve) {
		return &ErrorResponse{
			Code:      "VALIDATION_FAILED",
			Message:   "validation failed",
			Status:    ve.HTTPStatus(),
			Command:   command,
			RequestID: reqID,
			Errors:    ve.FieldErrors(),
		}
	}

	return &ErrorResponse{
		Code:      "INTERNAL_ERROR",
		Message:   err.Error(),
		Status:    500,
		Command:   command,
		RequestID: reqID,
	}
}

// InspectorSpec builds the introspective schema for all user-registered commands.
func (a *App) InspectorSpec() InspectorSpec {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var schemas []CommandSchema
	for _, name := range a.cmdOrder {
		cmd := a.commands[name]
		if cmd.internal {
			continue
		}
		schemas = append(schemas, CommandSchema{
			Name:        cmd.name,
			Group:       cmd.group,
			Summary:     cmd.doc.Summary,
			Description: cmd.doc.Description,
			Tags:        cmd.doc.Tags,
			Deprecated:  cmd.doc.Deprecated,
			Input:       BuildTypeSchema(cmd.doc.RequestType),
			Output:      BuildTypeSchema(cmd.doc.ReplyType),
		})
	}

	return InspectorSpec{
		AppID:       a.config.AppID,
		Title:       a.config.Title,
		Version:     a.config.Version,
		Description: a.config.Description,
		Commands:    schemas,
	}
}

// ServeHTTP implements http.Handler for the Brisa loopback server and httptest.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Security headers for desktop webview isolation
	if !a.isAllowedHost(r.Host) {
		http.Error(w, "Forbidden host", http.StatusForbidden)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	switch r.URL.Path {
	case "/brisa/adwaita.css", "/brisa/theme.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = io.WriteString(w, RenderThemeCSS(a.Theme()))
		return

	case "/brisa/ubuntu.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = io.WriteString(w, RenderUbuntuCSS())
		return

	case "/brisa/plasma.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = io.WriteString(w, RenderPlasmaCSS())
		return

	case "/brisa/macos.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = io.WriteString(w, RenderMacOSCSS())
		return

	case "/brisa/windows.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = io.WriteString(w, RenderWindowsCSS())
		return

	case "/brisa/icon.png", "/favicon.ico":
		if len(a.iconBytes) > 0 {
			w.Header().Set("Content-Type", http.DetectContentType(a.iconBytes))
			_, _ = w.Write(a.iconBytes)
			return
		}

	case "/brisa/manifest.json":
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(Map{
			"name":       a.config.Title,
			"short_name": a.config.Title,
			"start_url":  "/",
			"display":    "standalone",
			"icons": []Map{
				{
					"src":     "/brisa/icon.png",
					"sizes":   "512x512",
					"type":    "image/png",
					"purpose": "any maskable",
				},
			},
		})
		return

	case "/brisa/sdk.js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, a.GenerateJSSDK())
		return

	case "/brisa/schema.json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(a.InspectorSpec())
		return

	case "/brisa/types.d.ts":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, a.GenerateTypeScriptSDK())
		return

	case a.config.InspectorURL:
		if a.config.InspectorURL != "-" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, a.RenderInspectorHTML())
			return
		}

	case "/brisa/rpc":
		a.handleRPC(w, r)
		return

	case "/brisa/rpc/batch":
		a.handleRPCBatch(w, r)
		return

	case "/brisa/events":
		a.handleSSE(w, r)
		return
	}

	// DevServer proxy when configured
	if a.config.DevServerURL != "" {
		if target, err := url.Parse(a.config.DevServerURL); err == nil {
			proxy := httputil.NewSingleHostReverseProxy(target)
			proxy.ServeHTTP(w, r)
			return
		}
	}

	a.serveStaticAsset(w, r)
}

func (a *App) checkSecurityToken(r *http.Request) bool {
	if a.config.DisableCSRFToken {
		return true
	}
	tok := r.Header.Get("X-Brisa-Token")
	if tok == "" {
		tok = r.URL.Query().Get("token")
	}
	if tok == "" || a.token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tok), []byte(a.token)) == 1
}

// isAllowedHost only accepts localhost, loopback IPs (127.0.0.1, ::1), or an explicitly configured Addr host.
// This blocks DNS rebinding, where the browser sends "Host: attacker.example".
func (a *App) isAllowedHost(hostport string) bool {
	if a.config.DisableCSRFToken {
		return true
	}
	host := extractHost(hostport)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	if cfgHost := extractHost(a.config.Addr); cfgHost != "" && cfgHost != "0.0.0.0" && cfgHost != "::" {
		if strings.EqualFold(host, cfgHost) {
			return true
		}
	}
	return false
}

func extractHost(hostport string) string {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	return strings.Trim(host, "[]")
}

type rpcRequestEnvelope struct {
	Command   string          `json:"command"`
	Payload   json.RawMessage `json:"payload"`
	RequestID string          `json:"requestId,omitempty"`
}

func (a *App) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Map{
			"error": ErrorResponse{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "POST required for /brisa/rpc",
				Status:  http.StatusMethodNotAllowed,
			},
		})
		return
	}
	if !a.checkSecurityToken(r) {
		writeJSON(w, http.StatusForbidden, Map{
			"error": ErrorResponse{
				Code:    "INVALID_TOKEN",
				Message: "missing or invalid X-Brisa-Token header",
				Status:  http.StatusForbidden,
			},
		})
		return
	}

	var env rpcRequestEnvelope
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Map{
			"error": ErrorResponse{
				Code:    "READ_ERROR",
				Message: err.Error(),
				Status:  http.StatusBadRequest,
			},
		})
		return
	}
	if err := json.Unmarshal(body, &env); err != nil || strings.TrimSpace(env.Command) == "" {
		writeJSON(w, http.StatusBadRequest, Map{
			"error": ErrorResponse{
				Code:    "INVALID_RPC_ENVELOPE",
				Message: "request body must be JSON with a non-empty 'command' field",
				Status:  http.StatusBadRequest,
			},
		})
		return
	}

	res := a.executeCommand(r.Context(), env.Command, env.RequestID, env.Payload, r, w)
	if res.Error != nil {
		writeJSON(w, res.Status, Map{
			"error":     res.Error,
			"requestId": res.RequestID,
		})
		return
	}
	writeJSON(w, res.Status, Map{
		"data":      res.Data,
		"requestId": res.RequestID,
	})
}

func (a *App) handleRPCBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Map{"error": "POST required"})
		return
	}
	if !a.checkSecurityToken(r) {
		writeJSON(w, http.StatusForbidden, Map{"error": "invalid X-Brisa-Token"})
		return
	}

	var calls []rpcRequestEnvelope
	if err := json.NewDecoder(io.LimitReader(r.Body, 32<<20)).Decode(&calls); err != nil {
		writeJSON(w, http.StatusBadRequest, Map{"error": "invalid batch JSON array"})
		return
	}

	results := make([]*DispatchResult, len(calls))
	for i, c := range calls {
		results[i] = a.executeCommand(r.Context(), c.Command, c.RequestID, c.Payload, r, w)
	}
	writeJSON(w, http.StatusOK, results)
}

func (a *App) handleSSE(w http.ResponseWriter, r *http.Request) {
	if !a.checkSecurityToken(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub := &sseSubscriber{ch: make(chan string, 64)}
	a.sseMu.Lock()
	a.sseClients[sub] = struct{}{}
	a.sseMu.Unlock()

	defer func() {
		a.sseMu.Lock()
		delete(a.sseClients, sub)
		a.sseMu.Unlock()
	}()

	_, _ = io.WriteString(w, ": brisa-sse-connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-sub.ch:
			if _, err := io.WriteString(w, msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (a *App) serveStaticAsset(w http.ResponseWriter, r *http.Request) {
	cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if cleanPath == "" || cleanPath == "." {
		cleanPath = "index.html"
	}

	if a.AssetsFS != nil {
		data, err := fs.ReadFile(a.AssetsFS, cleanPath)
		if err != nil && cleanPath != "index.html" {
			// SPA fallback to index.html
			if indexData, idxErr := fs.ReadFile(a.AssetsFS, "index.html"); idxErr == nil {
				cleanPath = "index.html"
				data = indexData
				err = nil
			}
		}
		if err == nil {
			if cleanPath == "index.html" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(a.injectBrisaRuntime(data))
				return
			}
			ctype := mime.TypeByExtension(path.Ext(cleanPath))
			if ctype == "" {
				ctype = http.DetectContentType(data)
			}
			w.Header().Set("Content-Type", ctype)
			_, _ = w.Write(data)
			return
		}
	}

	if a.config.IndexHTML != "" && (cleanPath == "index.html" || !strings.Contains(cleanPath, ".")) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(a.injectBrisaRuntime([]byte(a.config.IndexHTML)))
		return
	}

	http.NotFound(w, r)
}

func (a *App) injectBrisaRuntime(htmlBytes []byte) []byte {
	if bytes.Contains(htmlBytes, []byte("/brisa/sdk.js")) {
		return htmlBytes
	}
	var snippet strings.Builder
	snippet.WriteString(`<meta name="google" content="notranslate">`)
	snippet.WriteString(`<link rel="manifest" href="/brisa/manifest.json">`)
	if len(a.iconBytes) > 0 && !bytes.Contains(htmlBytes, []byte(`rel="icon"`)) {
		snippet.WriteString(`<link rel="icon" type="image/png" sizes="512x512" href="/brisa/icon.png"><link rel="apple-touch-icon" href="/brisa/icon.png">`)
	}
	if !a.config.DisableAdwaitaCSS &&
		!bytes.Contains(htmlBytes, []byte("/brisa/adwaita.css")) &&
		!bytes.Contains(htmlBytes, []byte("/brisa/ubuntu.css")) &&
		!bytes.Contains(htmlBytes, []byte("/brisa/theme.css")) {
		snippet.WriteString(`<link rel="stylesheet" href="/brisa/adwaita.css">`)
	}
	snippet.WriteString(`<script src="/brisa/sdk.js"></script>`)
	scriptTag := []byte(snippet.String())

	if idx := bytes.Index(bytes.ToLower(htmlBytes), []byte("</head>")); idx != -1 {
		var out bytes.Buffer
		out.Write(htmlBytes[:idx])
		out.Write(scriptTag)
		out.Write(htmlBytes[idx:])
		return out.Bytes()
	}
	var out bytes.Buffer
	out.Write(scriptTag)
	out.Write(htmlBytes)
	return out.Bytes()
}

func resolveAppIcon(icon string, rawAssets, subAssets fs.FS) ([]byte, string) {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return nil, ""
	}
	clean := strings.TrimPrefix(path.Clean(strings.ReplaceAll(icon, "\\", "/")), "/")
	for _, fsys := range []fs.FS{rawAssets, subAssets} {
		if fsys != nil {
			if data, err := fs.ReadFile(fsys, clean); err == nil && len(data) > 0 {
				if _, statErr := os.Stat(icon); statErr == nil {
					return data, icon
				}
				return data, ""
			}
		}
	}
	for _, candidate := range []string{icon, path.Join("examples", clean)} {
		if data, err := os.ReadFile(candidate); err == nil && len(data) > 0 {
			return data, candidate
		}
	}
	return nil, ""
}

// Start starts the internal loopback server and launches the desktop window without blocking.
func (a *App) Start(ctx context.Context) error {
	for _, fn := range a.startupHooks {
		if err := fn(ctx); err != nil {
			return err
		}
	}

	ln, err := net.Listen("tcp", a.config.Addr)
	if err != nil {
		return fmt.Errorf("brisa listen %s: %w", a.config.Addr, err)
	}

	srv := &http.Server{
		Handler:           a,
		ReadHeaderTimeout: 10 * time.Second,
	}

	a.mu.Lock()
	a.listener = ln
	a.httpServer = srv
	a.baseURL = "http://" + ln.Addr().String()
	baseURL := a.baseURL
	a.mu.Unlock()

	go func() {
		_ = srv.Serve(ln)
	}()

	if err := a.window.launch(baseURL); err != nil {
		_ = srv.Close()
		return err
	}

	ensureLinuxDesktopEntry(a.config, a.iconBytes, a.iconPath)
	setX11WindowIcon(a.config.AppID, a.iconBytes)

	a.startSystemThemeWatcher(ctx)

	for _, fn := range a.readyHooks {
		fn(a)
	}
	return nil
}

// Run starts the Brisa desktop application, opens the native/app-mode window,
// and blocks until the window is closed or an OS termination signal is received.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Start(ctx); err != nil {
		return err
	}

	if a.config.Window.Headless {
		<-ctx.Done()
	} else {
		a.window.runLoop(ctx)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return a.Shutdown(shutdownCtx)
}

// Shutdown gracefully closes the desktop window, runs shutdown hooks, and stops the loopback server.
func (a *App) Shutdown(ctx context.Context) error {
	for _, fn := range a.beforeCloseHooks {
		if !fn(a) {
			return nil
		}
	}

	_ = a.window.Close()

	a.mu.RLock()
	srv := a.httpServer
	a.mu.RUnlock()

	var err error
	if srv != nil {
		err = srv.Shutdown(ctx)
	}

	for _, fn := range a.shutdownHooks {
		fn(a)
	}
	return err
}

func (a *App) registerBuiltinCommands() {
	// Events
	a.RegisterCommand("$brisa.events.emit", func(ctx *Context) error {
		var req struct {
			Event string `json:"event"`
			Data  any    `json:"data"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		a.Emit(req.Event, req.Data)
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})

	// Window
	a.RegisterCommand("$brisa.window.setTitle", func(ctx *Context) error {
		var req struct {
			Title string `json:"title"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		_ = a.window.SetTitle(req.Title)
		return ctx.Reply(200, Map{"title": req.Title})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.setSize", func(ctx *Context) error {
		var req struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		if err := a.window.SetSize(req.Width, req.Height); err != nil {
			return NewError(400, "INVALID_WINDOW_SIZE", err.Error())
		}
		return ctx.Reply(200, Map{"width": req.Width, "height": req.Height})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.setPosition", func(ctx *Context) error {
		var req struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		_ = a.window.SetPosition(req.X, req.Y)
		return ctx.Reply(200, Map{"x": req.X, "y": req.Y})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.minimize", func(ctx *Context) error {
		_ = a.window.Minimize()
		return ctx.Reply(200, Map{"state": a.window.State()})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.maximize", func(ctx *Context) error {
		_ = a.window.Maximize()
		return ctx.Reply(200, Map{"state": a.window.State()})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.restore", func(ctx *Context) error {
		_ = a.window.Restore()
		return ctx.Reply(200, Map{"state": a.window.State()})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.setFullscreen", func(ctx *Context) error {
		var req struct {
			Fullscreen bool `json:"fullscreen"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		_ = a.window.SetFullscreen(req.Fullscreen)
		return ctx.Reply(200, Map{"state": a.window.State()})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.close", func(ctx *Context) error {
		go func() { _ = a.window.Close() }()
		return ctx.Reply(200, Map{"closing": true})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.window.state", func(ctx *Context) error {
		w, h := a.window.Size()
		x, y := a.window.Position()
		return ctx.Reply(200, Map{
			"title":  a.window.Title(),
			"width":  w,
			"height": h,
			"x":      x,
			"y":      y,
			"state":  a.window.State(),
		})
	}, CommandDoc{})

	// Dialog
	a.RegisterCommand("$brisa.dialog.message", func(ctx *Context) error {
		var opts MessageDialogOptions
		if err := ctx.BindJSON(&opts); err != nil {
			return err
		}
		_ = a.dialog.Message(opts)
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.dialog.confirm", func(ctx *Context) error {
		var req struct {
			Title   string `json:"title"`
			Message string `json:"message"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		confirmed, _ := a.dialog.Confirm(req.Title, req.Message)
		return ctx.Reply(200, Map{"confirmed": confirmed})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.dialog.openFile", func(ctx *Context) error {
		var opts FileDialogOptions
		if err := ctx.BindJSON(&opts); err != nil {
			return err
		}
		file, _ := a.dialog.OpenFile(opts)
		return ctx.Reply(200, Map{"path": file})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.dialog.openMultipleFiles", func(ctx *Context) error {
		var opts FileDialogOptions
		if err := ctx.BindJSON(&opts); err != nil {
			return err
		}
		files, _ := a.dialog.OpenMultipleFiles(opts)
		return ctx.Reply(200, Map{"paths": files})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.dialog.openDirectory", func(ctx *Context) error {
		var opts FileDialogOptions
		if err := ctx.BindJSON(&opts); err != nil {
			return err
		}
		dir, _ := a.dialog.OpenDirectory(opts)
		return ctx.Reply(200, Map{"path": dir})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.dialog.saveFile", func(ctx *Context) error {
		var opts FileDialogOptions
		if err := ctx.BindJSON(&opts); err != nil {
			return err
		}
		path, _ := a.dialog.SaveFile(opts)
		return ctx.Reply(200, Map{"path": path})
	}, CommandDoc{})

	// Clipboard
	a.RegisterCommand("$brisa.clipboard.writeText", func(ctx *Context) error {
		var req struct {
			Text string `json:"text"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		if err := a.clipboard.WriteText(req.Text); err != nil {
			return err
		}
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.clipboard.readText", func(ctx *Context) error {
		text, err := a.clipboard.ReadText()
		if err != nil {
			return err
		}
		return ctx.Reply(200, Map{"text": text})
	}, CommandDoc{})

	// Store
	a.RegisterCommand("$brisa.store.get", func(ctx *Context) error {
		var req struct {
			Key string `json:"key"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		val, exists := a.store.GetAny(req.Key)
		return ctx.Reply(200, Map{"key": req.Key, "value": val, "exists": exists})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.store.set", func(ctx *Context) error {
		var req struct {
			Key   string `json:"key"`
			Value any    `json:"value"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		if err := a.store.Set(req.Key, req.Value); err != nil {
			return err
		}
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.store.delete", func(ctx *Context) error {
		var req struct {
			Key string `json:"key"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		if err := a.store.Delete(req.Key); err != nil {
			return err
		}
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.store.keys", func(ctx *Context) error {
		return ctx.Reply(200, Map{"keys": a.store.Keys()})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.store.all", func(ctx *Context) error {
		return ctx.Reply(200, a.store.All())
	}, CommandDoc{})

	a.RegisterCommand("$brisa.store.clear", func(ctx *Context) error {
		if err := a.store.Clear(); err != nil {
			return err
		}
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})

	// Notify & OpenExternal
	a.RegisterCommand("$brisa.notify", func(ctx *Context) error {
		var opts NotificationOptions
		if err := ctx.BindJSON(&opts); err != nil {
			return err
		}
		_ = a.Notify(opts)
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})

	a.RegisterCommand("$brisa.openExternal", func(ctx *Context) error {
		var req struct {
			Target string `json:"target"`
		}
		if err := ctx.BindJSON(&req); err != nil {
			return err
		}
		if err := OpenExternal(req.Target); err != nil {
			return NewError(400, "OPEN_EXTERNAL_FAILED", err.Error())
		}
		return ctx.Reply(200, Map{"ok": true})
	}, CommandDoc{})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
