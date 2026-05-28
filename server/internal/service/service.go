package service

import (
	"sync"
	"time"

	"github.com/gwyy/WebTrailAI/server/internal/config"
	"github.com/gwyy/WebTrailAI/server/internal/logger"
	scribble_manager "github.com/gwyy/WebTrailAI/server/pkg/scribble-manager"
	"github.com/gwyy/WebTrailAI/server/pkg/utils"
)

type Service struct {
	cfg     config.Config
	log     logger.Logger
	sm      *scribble_manager.ScribbleManager
	llm     utils.LLMClient
	userMu  sync.Mutex
	trailMu sync.Mutex
	// summaryRunMu 避免调试接口和定时任务同时生成日报，重复消耗大模型调用。
	summaryRunMu sync.Mutex
	nowFunc      func() time.Time
}

func NewService(cfg config.Config, log logger.Logger, sm *scribble_manager.ScribbleManager) *Service {
	return &Service{
		cfg:     cfg,
		log:     log,
		sm:      sm,
		nowFunc: time.Now,
	}
}

func (s *Service) SetLLMClient(llm utils.LLMClient) {
	s.llm = llm
}
