package main

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var enabledScopes map[string]struct{}

func init() {
	enabledScopes = make(map[string]struct{})

	envStr := os.Getenv("LOG_SCOPES")

	if envStr == "" {
		return
	}

	for scope := range strings.SplitSeq(envStr, ",") {
		enabledScopes[scope] = struct{}{}
	}
}

func NewLogger(scope string) *slog.Logger {
	var handler slog.Handler

	if _, ok := enabledScopes[scope]; ok {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(io.Discard, nil)
	}

	return slog.New(handler).With(slog.String("scope", scope))
}
