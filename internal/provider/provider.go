package provider

import "errors"

var ErrRateLimited = errors.New("rate limited by yt")

type Track struct {
	VideoID 	string
	Title		string
	Artist 		string
	Album		string
	Duration 	int
}

type Provider interface {
	Search(query string) ([]Track, error)
}