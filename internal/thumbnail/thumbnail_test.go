package thumbnail

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestRenderHalfBlocksShape(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), 128, 255})
		}
	}

	cols, rows := 24, 12
	out := renderHalfBlocks(img, cols, rows)
	lines := strings.Split(out, "\n")
	if len(lines) != rows {
		t.Fatalf("expected %d lines, got %d", rows, len(lines))
	}
	for i, l := range lines {
		if w := lipglossWidth(l); w != cols {
			t.Fatalf("line %d width = %d, want %d", i, w, cols)
		}
		if !strings.Contains(l, "\x1b[") {
			t.Fatalf("line %d missing ANSI escapes", i)
		}
	}
}

func TestRenderHalfBlocksAspect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if x < 16 {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				img.Set(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}
	out := renderHalfBlocks(img, 20, 10)
	lines := strings.Split(out, "\n")
	if len(lines) != 10 {
		t.Fatalf("expected 10 rows, got %d", len(lines))
	}
}

func TestRenderRejectsEmptyURL(t *testing.T) {
	if _, err := Render("", 20); err == nil {
		t.Fatal("expected error for empty url")
	}
}

// lipglossWidth counts visible cells, stripping ANSI sequences.
func lipglossWidth(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		n++
	}
	return n
}
