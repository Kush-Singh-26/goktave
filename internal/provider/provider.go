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
}

type Provider interface {
	Search(ctx context.Context, query string) ([]Track, error)
}
