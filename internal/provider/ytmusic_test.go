package provider

import "testing"

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"3:45", 225},
		{"1:05:20", 3920},
		{"0:45", 45},
		{"10", 10},
		{"", 0},
		{"abc", 0},
	}

	for _, tt := range tests {
		got := parseDuration(tt.input)
		if got != tt.expected {
			t.Errorf("parseDuration(%q) = %d; want %d", tt.input, got, tt.expected)
		}
	}
}

func TestDig(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": "c",
		},
	}

	if digStr(data, "a", "b") != "c" {
		t.Errorf("expected c, got %v", digStr(data, "a", "b"))
	}

	if dig(data, "a", "x") != nil {
		t.Errorf("expected nil, got %v", dig(data, "a", "x"))
	}
}
