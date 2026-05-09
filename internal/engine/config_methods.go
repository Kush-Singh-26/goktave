package engine

import "github.com/Kush-Singh-26/goktave/internal/config"

func (e *DefaultEngine) GetConfig() *config.Config {
	return e.cfg
}

func (e *DefaultEngine) SaveConfig() error {
	return e.db.SaveConfig(e.cfg)
}
