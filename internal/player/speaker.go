package player

import (
	"fmt"

	"github.com/Kush-Singh-26/goktave/internal/config"
	oto "github.com/ebitengine/oto/v3"
)

type Speaker struct {
	ctx *oto.Context
}

func NewSpeaker(cfg *config.Config) (*Speaker, error) {
	otoCtx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   cfg.SampleRate,
		ChannelCount: cfg.ChannelCount,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   cfg.BufferSize,
	})
	if err != nil {
		return nil, fmt.Errorf("oto init: %w", err)
	}
	<-ready
	return &Speaker{ctx: otoCtx}, nil
}
