package service

import (
	"sync"
	"time"

	"github.com/gwyy/WebTrailAI/server/internal/config"
	"github.com/gwyy/WebTrailAI/server/internal/logger"
	scribble_manager "github.com/gwyy/WebTrailAI/server/pkg/scribble-manager"
)

type Service struct {
	cfg     config.Config
	log     logger.Logger
	sm      *scribble_manager.ScribbleManager
	userMu  sync.Mutex
	trailMu sync.Mutex
	nowFunc func() time.Time
}

func NewService(cfg config.Config, log logger.Logger, sm *scribble_manager.ScribbleManager) *Service {
	return &Service{
		cfg:     cfg,
		log:     log,
		sm:      sm,
		nowFunc: time.Now,
	}
}
