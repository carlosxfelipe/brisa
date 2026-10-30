package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// HandlerFunc defines the low-level command handler function signature.
type HandlerFunc func(ctx *Context) error

// MiddlewareFunc defines a function that wraps a command execution chain.
type MiddlewareFunc func(ctx *Context) error

// Context encapsulates the state of a single IPC command invocation in Brisa.
type Context struct {
	stdCtx     context.Context
	app        *App
	command    string
	requestID  string
	rawPayload json.RawMessage
	httpReq    *http.Request
	httpWriter http.ResponseWriter
	handlers   []HandlerFunc
	index      int

	mu          sync.RWMutex
	store       map[string]any
	status      int
	result      any
	replied     bool
	errResponse *ErrorResponse
}

func newContext(stdCtx context.Context, app *App, command, requestID string, rawPayload json.RawMessage, handlers []HandlerFunc, r *http.Request, w http.ResponseWriter) *Context {
	if stdCtx == nil {
		stdCtx = context.Background()
	}
	return &Context{
		stdCtx:     stdCtx,
		app:        app,
		command:    command,
		requestID:  requestID,
		rawPayload: rawPayload,
		httpReq:    r,
		httpWriter: w,
		handlers:   handlers,
		index:      -1,
		status:     200,
	}
}

// Context returns the underlying standard library context.Context.
func (c *Context) Context() context.Context {
	if c.stdCtx == nil {
		return context.Background()
	}
	return c.stdCtx
}

// WithContext replaces the underlying context.Context (e.g., to attach deadlines or trace values).
func (c *Context) WithContext(ctx context.Context) {
	if ctx != nil {
		c.stdCtx = ctx
	}
}

// App returns the parent Brisa application instance.
func (c *Context) App() *App {
	return c.app
}

// Command returns the name of the invoked IPC command (e.g., "tasks.create").
func (c *Context) Command() string {
	return c.command
}

// RequestID returns the unique correlation ID for this command call.
func (c *Context) RequestID() string {
	return c.requestID
}

// SetRequestID sets the correlation ID for this command call.
func (c *Context) SetRequestID(id string) {
	c.requestID = id
}

// RawPayload returns the raw JSON input bytes passed to the command.
func (c *Context) RawPayload() json.RawMessage {
	return c.rawPayload
}

// BindJSON unmarshals the raw command payload into dest.
func (c *Context) BindJSON(dest any) error {
	if len(c.rawPayload) == 0 || string(c.rawPayload) == "null" {
		return nil
	}
	dec := json.NewDecoder(bytesReader(c.rawPayload))
	if err := dec.Decode(dest); err != nil {
		return NewError(400, "INVALID_PAYLOAD", fmt.Sprintf("invalid JSON payload: %v", err))
	}
	return nil
}

// Next executes the next middleware or handler in the command pipeline.
func (c *Context) Next() error {
	c.index++
	for c.index < len(c.handlers) {
		h := c.handlers[c.index]
		if err := h(c); err != nil {
			return err
		}
		c.index++
	}
	return nil
}

// Set stores a request-scoped key-value pair on the Context.
func (c *Context) Set(key string, val any) {
	c.mu.Lock()
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[key] = val
	c.mu.Unlock()
}

// Get retrieves a request-scoped value by key.
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.store == nil {
		return nil, false
	}
	v, ok := c.store[key]
	return v, ok
}

// MustGet retrieves a request-scoped value by key or panics if absent.
func (c *Context) MustGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic(fmt.Sprintf("brisa context: key %q does not exist", key))
	}
	return v
}

// Status sets the response status code for the command execution.
func (c *Context) Status(code int) *Context {
	c.status = code
	return c
}

// Reply records a successful structured result for the command.
func (c *Context) Reply(status int, data any) error {
	if status > 0 {
		c.status = status
	}
	c.result = data
	c.replied = true
	return nil
}

// Error records a structured error response for the command.
func (c *Context) Error(status int, code, message string) error {
	return NewError(status, code, message)
}

// Result returns the recorded command response data and status code.
func (c *Context) Result() (any, int, bool) {
	return c.result, c.status, c.replied
}

// Emit broadcasts a real-time event from Go to all connected desktop UI listeners.
func (c *Context) Emit(event string, data any) {
	if c.app != nil {
		c.app.Emit(event, data)
	}
}

// Window returns the desktop Window controller.
func (c *Context) Window() *Window {
	if c.app == nil {
		return nil
	}
	return c.app.Window()
}

// Dialog returns the native OS Dialog controller.
func (c *Context) Dialog() *Dialog {
	if c.app == nil {
		return nil
	}
	return c.app.Dialog()
}

// Clipboard returns the native OS Clipboard controller.
func (c *Context) Clipboard() *Clipboard {
	if c.app == nil {
		return nil
	}
	return c.app.Clipboard()
}

// Store returns the application's persistent local key-value store.
func (c *Context) Store() *Store {
	if c.app == nil {
		return nil
	}
	return c.app.Store()
}

// Notify triggers a desktop notification.
func (c *Context) Notify(opts NotificationOptions) error {
	if c.app == nil {
		return errors.New("brisa: app is nil")
	}
	return c.app.Notify(opts)
}

type rawSliceReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) *rawSliceReader {
	return &rawSliceReader{b: b}
}

func (r *rawSliceReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, errEOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

var errEOF = errors.New("EOF")
