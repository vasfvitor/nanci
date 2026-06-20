package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	logpkg "github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
)

const (
	desktopLogFileName   = "nanci-desktop.log"
	wailsLogFileName     = "wails.log"
	logFileMaxBytes      = 2 * 1024 * 1024
	logFileMaxBackups    = 3
	maxBufferedLogEvents = 1000
)

type desktopLogEvent struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Attrs string `json:"attrs"`
	Raw   string `json:"raw"`
}

func desktopLogDir() (string, error) {
	dataDir, err := paths.DataDir()
	if err != nil {
		return "", fmt.Errorf("data dir: %w", err)
	}
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", fmt.Errorf("ensure log dir: %w", err)
	}
	return logDir, nil
}

type rotatingFileWriter struct {
	basePath    string
	maxBytes    int64
	maxBackups  int
	mu          sync.Mutex
	file        *os.File
	currentSize int64
}

func newRotatingFileWriter(basePath string, maxBytes int64, maxBackups int) (*rotatingFileWriter, error) {
	w := &rotatingFileWriter{
		basePath:   basePath,
		maxBytes:   maxBytes,
		maxBackups: maxBackups,
	}
	if err := w.openCurrent(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.openCurrent(); err != nil {
			return 0, err
		}
	}

	if w.currentSize > 0 && w.currentSize+int64(len(p)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.currentSize += int64(n)
	return n, err
}

func (w *rotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.currentSize = 0
	return err
}

func (w *rotatingFileWriter) openCurrent() error {
	file, err := os.OpenFile(w.basePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	w.file = file
	w.currentSize = info.Size()
	return nil
}

func (w *rotatingFileWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close log file: %w", err)
		}
		w.file = nil
		w.currentSize = 0
	}

	if w.maxBackups > 0 {
		oldestBackup := rotatedLogPath(w.basePath, w.maxBackups)
		if err := os.Remove(oldestBackup); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove oldest backup: %w", err)
		}

		for i := w.maxBackups - 1; i >= 1; i-- {
			src := rotatedLogPath(w.basePath, i)
			dst := rotatedLogPath(w.basePath, i+1)
			if err := os.Rename(src, dst); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("shift log backup %d: %w", i, err)
			}
		}

		if err := os.Rename(w.basePath, rotatedLogPath(w.basePath, 1)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("rotate current log: %w", err)
		}
	} else {
		if err := os.Remove(w.basePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove current log: %w", err)
		}
	}

	return w.openCurrent()
}

func rotatedLogPath(basePath string, index int) string {
	return fmt.Sprintf("%s.%d", basePath, index)
}

func collectRotatedLogPaths(basePath string, maxBackups int) []string {
	paths := make([]string, 0, maxBackups+1)
	for i := maxBackups; i >= 1; i-- {
		paths = append(paths, rotatedLogPath(basePath, i))
	}
	return append(paths, basePath)
}

type desktopLogHandler struct {
	base slog.Handler
	emit func(event desktopLogEvent)
}

func newDesktopLogHandler(output io.Writer, level slog.Leveler, emit func(event desktopLogEvent)) slog.Handler {
	return &desktopLogHandler{
		base: slog.NewTextHandler(output, &slog.HandlerOptions{Level: level}),
		emit: emit,
	}
}

func (h *desktopLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *desktopLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if err := h.base.Handle(ctx, record); err != nil {
		return err
	}

	if h.emit != nil {
		h.emit(newDesktopLogEvent(record))
	}

	return nil
}

func (h *desktopLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &desktopLogHandler{
		base: h.base.WithAttrs(attrs),
		emit: h.emit,
	}
}

func (h *desktopLogHandler) WithGroup(name string) slog.Handler {
	return &desktopLogHandler{
		base: h.base.WithGroup(name),
		emit: h.emit,
	}
}

func newDesktopLogEvent(record slog.Record) desktopLogEvent {
	eventTime := record.Time
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}

	var attrs []string
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s=%s", attr.Key, attr.Value.String()))
		return true
	})

	level := strings.ToUpper(record.Level.String())
	attrText := strings.Join(attrs, " ")
	raw := fmt.Sprintf("time=%s level=%s msg=%q", eventTime.Format(time.RFC3339), level, record.Message)
	if attrText != "" {
		raw += " " + attrText
	}

	return desktopLogEvent{
		Time:  eventTime.Format(time.RFC3339),
		Level: level,
		Msg:   record.Message,
		Attrs: attrText,
		Raw:   raw,
	}
}

func newDesktopLogger(ctx context.Context, level slog.Leveler, basePath string) (*slog.Logger, *rotatingFileWriter, error) {
	writer, err := newRotatingFileWriter(basePath, logFileMaxBytes, logFileMaxBackups)
	if err != nil {
		return nil, nil, err
	}

	handler := newDesktopLogHandler(writer, level, func(event desktopLogEvent) {
		if ctx != nil {
			runtime.EventsEmit(ctx, "backend-log", event)
		}
	})

	return slog.New(handler), writer, nil
}

func resolveDesktopBaseLevel(trace bool) slog.Level {
	if trace {
		return logpkg.LevelTrace
	}
	return slog.LevelDebug
}
