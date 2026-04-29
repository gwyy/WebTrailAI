package scribble_manager

import (
	"fmt"

	"github.com/gwyy/WebTrailAI/server/internal/config"
	"github.com/gwyy/WebTrailAI/server/internal/logger"
	"github.com/sdomino/scribble"
)

type ScribbleManager struct {
	cfg config.Config
	db  *scribble.Driver
	log logger.Logger
}

func NewScribbleManager(cfg config.Config, log logger.Logger) (*ScribbleManager, error) {
	scribbleManager := &ScribbleManager{
		cfg: cfg,
		log: log,
	}
	dbDir := cfg.GetString("db.filedir")
	if dbDir == "" {
		return nil, fmt.Errorf("scribble db dir is empty")
	}
	db, err := scribble.New(dbDir, nil)
	if err != nil {
		return nil, fmt.Errorf("init scribble db failed: %w", err)
	}
	scribbleManager.db = db
	scribbleManager.log.Infof("scribble db started, dir=%s", dbDir)
	return scribbleManager, nil
}
func (m *ScribbleManager) DB() *scribble.Driver {
	return m.db
}
