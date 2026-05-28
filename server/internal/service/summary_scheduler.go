package service

import (
	"context"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const defaultSummaryCronSpec = "0 2 * * *"

type SummaryScheduler struct {
	cron   *cron.Cron
	srv    *Service
	log    cron.Logger
	cancel context.CancelFunc
}

func NewSummaryScheduler(srv *Service) *SummaryScheduler {
	cronLogger := &summaryCronLogger{srv: srv}
	c := cron.New(
		cron.WithLocation(time.Local),
		cron.WithLogger(cronLogger),
		cron.WithChain(
			cron.Recover(cronLogger),
			cron.SkipIfStillRunning(cronLogger),
		),
	)

	return &SummaryScheduler{
		cron: c,
		srv:  srv,
		log:  cronLogger,
	}
}

func (s *SummaryScheduler) Start(ctx context.Context, spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		spec = defaultSummaryCronSpec
	}

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	if _, err := s.cron.AddFunc(spec, func() {
		result, err := s.srv.RunYesterdaySummaryOnce(runCtx)
		if err != nil {
			s.srv.log.Errorf("每日浏览总结任务失败: err=%v", err)
			return
		}
		s.srv.log.Infof("每日浏览总结任务完成: date=%s users=%d success=%d skipped=%d failed=%d",
			result.Date, result.UserCount, result.SuccessCount, result.SkippedCount, result.FailedCount)
	}); err != nil {
		cancel()
		return err
	}

	s.cron.Start()
	s.srv.log.Infof("每日浏览总结任务已启动: cron=%s", spec)
	return nil
}

func (s *SummaryScheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(5 * time.Second):
		s.srv.log.Warn("等待每日浏览总结任务停止超时")
	}
}

type summaryCronLogger struct {
	srv *Service
}

func (l *summaryCronLogger) Info(msg string, keysAndValues ...interface{}) {
	l.srv.log.Infow("summary cron: "+msg, keysAndValues...)
}

func (l *summaryCronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	args := append([]interface{}{"error", err}, keysAndValues...)
	l.srv.log.Errorw("summary cron: "+msg, args...)
}
