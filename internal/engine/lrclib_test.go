package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/config"
)

func TestCleanArtist(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Rick Astley - Topic", "rick astley"},
		{"OfficialVEVO", ""},
		{"  The Beatles  ", "the beatles"},
	}
	for _, tt := range tests {
		if got := cleanArtist(tt.in); got != tt.want {
			t.Errorf("cleanArtist(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestCleanTitle(t *testing.T) {
	got := cleanTitle("Artist Name - Song Title (Official Video)", "Artist Name - Topic")
	if got != "song title" {
		t.Errorf("cleanTitle() = %q; want %q", got, "song title")
	}

	got = cleanTitle("Never Gonna Give You Up [Remastered HQ]", "Rick Astley")
	if got != "never gonna give you up" {
		t.Errorf("cleanTitle() = %q; want %q", got, "never gonna give you up")
	}
}

func TestLyricsFromLrcLibResponse(t *testing.T) {
	synced, err := lyricsFromLrcLibResponse(LrcLibResponse{
		SyncedLyrics: "[00:12.00]Hello",
		PlainLyrics:  "Hello plain",
	})
	if err != nil || synced != "[00:12.00]Hello" {
		t.Fatalf("expected synced lyrics, got %q err=%v", synced, err)
	}

	plain, err := lyricsFromLrcLibResponse(LrcLibResponse{PlainLyrics: "line one"})
	if err != nil || plain != "line one" {
		t.Fatalf("expected plain lyrics, got %q err=%v", plain, err)
	}

	_, err = lyricsFromLrcLibResponse(LrcLibResponse{Instrumental: true})
	if err == nil {
		t.Fatal("expected error for instrumental track")
	}

	_, err = lyricsFromLrcLibResponse(LrcLibResponse{})
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestLrcLibGetURL(t *testing.T) {
	u := lrcLibGetURL("ac/dc", "back in black", 257)
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("artist_name") != "ac/dc" {
		t.Errorf("artist_name = %q", parsed.Query().Get("artist_name"))
	}
	if parsed.Query().Get("track_name") != "back in black" {
		t.Errorf("track_name = %q", parsed.Query().Get("track_name"))
	}
	if parsed.Query().Get("duration") != "257" {
		t.Errorf("duration = %q", parsed.Query().Get("duration"))
	}

	noDur := lrcLibGetURL("artist", "title", 0)
	if strings.Contains(noDur, "duration=") {
		t.Errorf("expected no duration param, got %s", noDur)
	}
}

func TestFetchLrcLibLyrics_httptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("duration") != "" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(LrcLibResponse{
			SyncedLyrics: "[00:01.00]Test line",
		})
	}))
	defer srv.Close()

	orig := lrcLibAPIBase
	lrcLibAPIBase = srv.URL
	t.Cleanup(func() { lrcLibAPIBase = orig })

	cfg := config.Default()
	e := &DefaultEngine{cfg: cfg}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lyrics, err := e.fetchLrcLibLyrics(ctx, "Test line", "Test Artist", 200)
	if err != nil {
		t.Fatalf("fetchLrcLibLyrics: %v", err)
	}
	if !strings.Contains(lyrics, "[00:01.00]Test line") {
		t.Fatalf("unexpected lyrics: %q", lyrics)
	}
}
