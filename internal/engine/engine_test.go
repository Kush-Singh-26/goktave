package engine

import (
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/Kush-Singh-26/goktave/internal/logger"
)

func TestMain(m *testing.M) {
	logger.L = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	os.Exit(m.Run())
}
