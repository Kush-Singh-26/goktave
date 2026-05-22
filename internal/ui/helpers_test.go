package ui

import (
	"testing"
	"time"
)

func TestParseLRC(t *testing.T) {
	input := `[00:12.50]First line
[01:05.00]Second line
Not a timed line`

	lines := parseLRC(input)
	if len(lines) != 2 {
		t.Fatalf("parseLRC() len = %d; want 2", len(lines))
	}

	if lines[0].Text != "First line" {
		t.Errorf("line 0 text = %q", lines[0].Text)
	}
	want0 := 12*time.Second + 500*time.Millisecond
	if lines[0].Time != want0 {
		t.Errorf("line 0 time = %v; want %v", lines[0].Time, want0)
	}

	want1 := time.Minute + 5*time.Second
	if lines[1].Time != want1 {
		t.Errorf("line 1 time = %v; want %v", lines[1].Time, want1)
	}
}

func TestParseLRC_centiseconds(t *testing.T) {
	lines := parseLRC("[00:01.25]Hi")
	if len(lines) != 1 {
		t.Fatalf("len = %d", len(lines))
	}
	if lines[0].Time != 1250*time.Millisecond {
		t.Errorf("time = %v; want 1250ms", lines[0].Time)
	}
}

func TestParseLRC_plainText(t *testing.T) {
	if got := parseLRC("No timestamps here"); len(got) != 0 {
		t.Fatalf("expected no synced lines, got %d", len(got))
	}
}

func TestActiveLyricIndex(t *testing.T) {
	lines := []SyncedLine{
		{Time: 0, Text: "a"},
		{Time: 5 * time.Second, Text: "b"},
		{Time: 10 * time.Second, Text: "c"},
	}

	idx := activeLyricIndex(lines, 6*time.Second)
	if idx != 1 {
		t.Errorf("activeLyricIndex at 6s = %d; want 1", idx)
	}

	if activeLyricIndex(lines, 500*time.Millisecond) != 0 {
		t.Error("expected index 0 before first boundary")
	}
	if activeLyricIndex(nil, time.Second) != -1 {
		t.Error("expected -1 for empty lines")
	}
}
