package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// multiHandler fan-outs slog records to multiple handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, r.Level) {
			_ = hh.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		handlers[i] = hh.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		handlers[i] = hh.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

var Default *slog.Logger

func init() {
	Default = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// DefaultPath returns the standard log file location alongside the config file.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "tw-forge.log"
	}
	return filepath.Join(dir, "total-war-mod-editor", "tw-forge.log")
}

// Init sets up the multi-handler logger. If the log file cannot be opened,
// falls back to console-only logging without returning an error.
func Init(logPath string) {
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)

	var handlers []slog.Handler

	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	handlers = append(handlers, consoleHandler)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		fileHandler := slog.NewTextHandler(file, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		handlers = append(handlers, fileHandler)
		// Session separator so it's easy to tell runs apart in the log file.
		_, _ = fmt.Fprintf(file, "\n--- session %s ---\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	Default = slog.New(&multiHandler{handlers: handlers})
}

// OpStart logs the beginning of a high-level user operation.
func OpStart(op, detail string) {
	Default.Info("op_start", "op", op, "detail", detail)
}

// OpDone logs the result of a high-level operation.
func OpDone(op string, err error) {
	if err != nil {
		Default.Error("op_done", "op", op, "error", err)
	} else {
		Default.Info("op_done", "op", op, "status", "ok")
	}
}

// FileWrite logs a write action on a specific file.
// action — "append", "insert", "rewrite", etc.
// target — unit type, building name, or any identifier of the changed entity.
// lines  — number of lines written/affected (0 if unknown).
func FileWrite(file, action, target string, lines int) {
	if lines > 0 {
		Default.Info("file_write", "file", file, "action", action, "target", target, "lines", lines)
	} else {
		Default.Info("file_write", "file", file, "action", action, "target", target)
	}
}

// FieldChanged logs a single field change. Goes to console only (Debug level).
func FieldChanged(entity, field, from, to string) {
	Default.Debug("field_changed", "entity", entity, "field", field, "from", from, "to", to)
}

// Info logs a general informational message.
func Info(msg string, args ...any) {
	Default.Info(msg, args...)
}

// Debug logs a debug-level message (console only).
func Debug(msg string, args ...any) {
	Default.Debug(msg, args...)
}

// Warn logs a warning.
func Warn(msg string, args ...any) {
	Default.Warn(msg, args...)
}

// Error logs an error.
func Error(msg string, args ...any) {
	Default.Error(msg, args...)
}
