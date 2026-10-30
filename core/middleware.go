package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"
)

// RequestIDOptions configures the RequestID middleware.
type RequestIDOptions struct {
	Generator func() string
}

// SlogLoggerOptions configures the structured slog command logger middleware.
type SlogLoggerOptions struct {
	Logger *slog.Logger
}

// RateLimiterOptions configures the command rate-limiting middleware.
type RateLimiterOptions struct {
	Max    int
	Window time.Duration
}

// Recover returns a middleware that recovers from panics during IPC command execution
// and converts them into a structured 500 INTERNAL_PANIC RPCError.
func Recover() MiddlewareFunc {
	return func(ctx *Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = NewError(500, "INTERNAL_PANIC", fmt.Sprintf("panic recovered in command %q: %v", ctx.Command(), r))
			}
		}()
		return ctx.Next()
	}
}

// Logger returns a standard log middleware that logs IPC command execution latency and status.
func Logger() MiddlewareFunc {
	return func(ctx *Context) error {
		start := time.Now()
		err := ctx.Next()
		dur := time.Since(start)
		status := ctx.status
		if err != nil {
			var rpcErr *RPCError
			if ok := asRPCError(err, &rpcErr); ok && rpcErr.Status != 0 {
				status = rpcErr.Status
			} else {
				status = 500
			}
		}
		log.Printf("[BRISA] cmd=%s status=%d duration=%s req_id=%s", ctx.Command(), status, dur, ctx.RequestID())
		return err
	}
}

// SlogLogger returns a structured log/slog middleware for IPC commands.
func SlogLogger(opts ...SlogLoggerOptions) MiddlewareFunc {
	var logger *slog.Logger
	if len(opts) > 0 && opts[0].Logger != nil {
		logger = opts[0].Logger
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return func(ctx *Context) error {
		start := time.Now()
		err := ctx.Next()
		dur := time.Since(start)
		status := ctx.status
		if err != nil {
			var rpcErr *RPCError
			if ok := asRPCError(err, &rpcErr); ok && rpcErr.Status != 0 {
				status = rpcErr.Status
			} else {
				status = 500
			}
		}
		attrs := []any{
			slog.String("command", ctx.Command()),
			slog.Int("status", status),
			slog.Duration("duration", dur),
			slog.String("request_id", ctx.RequestID()),
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
			logger.ErrorContext(ctx.Context(), "brisa command failed", attrs...)
		} else {
			logger.InfoContext(ctx.Context(), "brisa command completed", attrs...)
		}
		return err
	}
}

// RequestID ensures every IPC command invocation has a unique RequestID.
func RequestID(opts ...RequestIDOptions) MiddlewareFunc {
	gen := defaultRequestID
	if len(opts) > 0 && opts[0].Generator != nil {
		gen = opts[0].Generator
	}
	return func(ctx *Context) error {
		if ctx.RequestID() == "" {
			ctx.SetRequestID(gen())
		}
		return ctx.Next()
	}
}

// Timeout attaches a context deadline to the command execution and returns 408 COMMAND_TIMEOUT if exceeded.
func Timeout(d time.Duration) MiddlewareFunc {
	return func(ctx *Context) error {
		if d <= 0 {
			return ctx.Next()
		}
		c, cancel := context.WithTimeout(ctx.Context(), d)
		defer cancel()
		ctx.WithContext(c)

		err := ctx.Next()
		if c.Err() == context.DeadlineExceeded {
			return NewError(408, "COMMAND_TIMEOUT", fmt.Sprintf("command %q timed out after %s", ctx.Command(), d))
		}
		return err
	}
}

// RateLimiter limits how many times a given IPC command can be called within a time window.
func RateLimiter(opts RateLimiterOptions) MiddlewareFunc {
	maxCalls := opts.Max
	if maxCalls <= 0 {
		maxCalls = 60
	}
	window := opts.Window
	if window <= 0 {
		window = time.Minute
	}

	type bucket struct {
		count   int
		resetAt time.Time
	}
	var (
		mu      sync.Mutex
		buckets = make(map[string]*bucket)
	)

	return func(ctx *Context) error {
		now := time.Now()
		key := ctx.Command()

		mu.Lock()
		b, ok := buckets[key]
		if !ok || now.After(b.resetAt) {
			b = &bucket{count: 0, resetAt: now.Add(window)}
			buckets[key] = b
		}
		b.count++
		current := b.count
		mu.Unlock()

		if current > maxCalls {
			return NewError(429, "RATE_LIMIT_EXCEEDED", fmt.Sprintf("rate limit exceeded for command %q", key))
		}
		return ctx.Next()
	}
}

func defaultRequestID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return "req_" + hex.EncodeToString(b[:])
}

func asRPCError(err error, target **RPCError) bool {
	if err == nil {
		return false
	}
	if r, ok := err.(*RPCError); ok {
		*target = r
		return true
	}
	return false
}
