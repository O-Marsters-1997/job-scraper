// Package logger builds the process's slog.Logger and carries correlation
// attributes on a context.Context so they reach every log line without
// being passed explicitly (ADR 0013).
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
)

// New builds the process's slog.Logger. An unrecognised, non-empty format
// or level returns an error.
func New(w io.Writer, format, level string) (*slog.Logger, error) {
	lvl, err := parseLevel(level)
	if err != nil {
		return nil, err
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var base slog.Handler
	switch format {
	case "", "json":
		base = slog.NewJSONHandler(w, opts)
	case "text":
		base = slog.NewTextHandler(w, opts)
	default:
		return nil, fmt.Errorf("logger: unknown LOG_FORMAT %q", format)
	}
	return slog.New(ctxHandler{base}), nil
}

// NewFromEnv builds the process's logger from LOG_FORMAT and LOG_LEVEL,
// writing to standard output.
func NewFromEnv() (*slog.Logger, error) {
	return New(os.Stdout, os.Getenv("LOG_FORMAT"), os.Getenv("LOG_LEVEL"))
}

// MustFromEnv is NewFromEnv, printing the error and exiting the process
// instead of returning it.
func MustFromEnv() *slog.Logger {
	lg, err := NewFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return lg
}

func parseLevel(level string) (slog.Level, error) {
	if level == "" {
		return slog.LevelInfo, nil
	}
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return 0, fmt.Errorf("logger: unknown LOG_LEVEL %q", level)
	}
	return lvl, nil
}

type ctxAttrsKey struct{}

type ctxAttrMap map[string]slog.Value

// With returns a context carrying attrs, which every *Context call made
// with it attaches to its record. A later With call, or a field set at the
// log call site, wins over the same key here.
func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	existing, _ := ctx.Value(ctxAttrsKey{}).(ctxAttrMap)
	merged := make(ctxAttrMap, len(existing)+len(attrs))
	maps.Copy(merged, existing)
	for _, a := range attrs {
		merged[a.Key] = a.Value
	}
	return context.WithValue(ctx, ctxAttrsKey{}, merged)
}

func ctxAttrs(ctx context.Context) []slog.Attr {
	m, _ := ctx.Value(ctxAttrsKey{}).(ctxAttrMap)
	if len(m) == 0 {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(m))
	for k, v := range m {
		attrs = append(attrs, slog.Attr{Key: k, Value: v})
	}
	return attrs
}

func ctxAttr(ctx context.Context, key string) (slog.Value, bool) {
	m, _ := ctx.Value(ctxAttrsKey{}).(ctxAttrMap)
	v, ok := m[key]
	return v, ok
}

type ctxHandler struct {
	slog.Handler
}

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := ctxAttrsExcept(ctx, r)
	if len(attrs) == 0 {
		return h.Handler.Handle(ctx, r)
	}
	return h.Handler.WithAttrs(attrs).Handle(ctx, r)
}

// ctxAttrsExcept returns ctx's attributes, dropping any key r already sets
// itself — WithAttrs binds attrs as separate handler state, so without this
// a key on both ctx and r would print twice instead of r's value winning.
func ctxAttrsExcept(ctx context.Context, r slog.Record) []slog.Attr {
	attrs := ctxAttrs(ctx)
	if len(attrs) == 0 {
		return nil
	}
	own := make(map[string]bool, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		own[a.Key] = true
		return true
	})
	filtered := attrs[:0]
	for _, a := range attrs {
		if !own[a.Key] {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func (h ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ctxHandler{h.Handler.WithAttrs(attrs)}
}

func (h ctxHandler) WithGroup(name string) slog.Handler {
	return ctxHandler{h.Handler.WithGroup(name)}
}
