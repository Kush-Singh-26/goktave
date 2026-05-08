package provider

import (
	"context"
	"errors"
)

var ErrRateLimited = errors.New("rate limited by yt")

type Track struct {
	VideoID 	string
	Title		string
	Artist 		string
	Album		string
	Duration 	int
	ThumbURL    string
	ThumbASCII  string
	ThumbWidth  int
	LocalPath   string
	LastPlayed  int64 // Unix timestamp
}

type Provider interface {
	Search(ctx context.Context, query string) ([]Track, error)
	GetSuggestions(ctx context.Context, input string) ([]string, error)
	GetUpNext(videoID string) ([]Track, string, error)
	GetLyrics(ctx context.Context, browseID string) (string, error)
}
