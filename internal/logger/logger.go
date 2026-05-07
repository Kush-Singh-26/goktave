package logger

import (
	"log/slog"
	"os"
)

var L *slog.Logger

func Init() (func(), error) {
	f, err := os.OpenFile("goktave.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	L = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return func() {
		f.Close()
	}, nil
}
