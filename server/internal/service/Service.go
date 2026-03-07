package service

import (
	"github.com/gwyy/WebTrailAI/server/internal/config"
	"github.com/gwyy/WebTrailAI/server/internal/logger"
)

type Service struct {
	cfg config.Config
	log logger.Logger
}

func NewService(cfg config.Config, log logger.Logger) *Service {
	return &Service{
		cfg: cfg,
		log: log,
	}
}
