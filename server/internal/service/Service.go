package service

import "github.com/gwyy/WebTrailAI/server/config"

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		cfg: cfg,
	}
}
