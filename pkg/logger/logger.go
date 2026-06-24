package logger

import (
    "log/slog"
    "os"

    "github.com/webhook-platform/pkg/config"
)

func New(cfg *config.Config) *slog.Logger {
    var level slog.Level
    switch cfg.LogLevel {
    case "debug":
        level = slog.LevelDebug
    case "warn":
        level = slog.LevelWarn
    case "error":
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }

    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level:     level,
        AddSource: cfg.Environment == "development",
    })

    logger := slog.New(handler)
    slog.SetDefault(logger)

    return logger
}