package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
)

const (
	trailRootCollection = "trails"
	trailDailyResource  = "trail"
	maxDailyTrailCount  = 2000
	maxTrailTitleRunes  = 512
	maxTrailURLBytes    = 8192
)

var (
	ErrInvalidTrailUserID = errors.New("用户身份无效")
	ErrInvalidTrailTitle  = errors.New("标题不能为空且长度不能超过512个字符")
	ErrInvalidTrailURL    = errors.New("URL必须是有效的 http 或 https 地址，且长度不能超过8192字节")
)

// AddTrail 将用户的一条浏览记录追加到当天 trail 文件中，并保证当天最多保留最新 2000 条。
func (s *Service) AddTrail(ctx context.Context, userID int, req *request.TrailAdd) (*filedb.Trail, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if userID <= 0 {
		return nil, 0, ErrInvalidTrailUserID
	}
	if req == nil {
		return nil, 0, errors.New("浏览记录参数不能为空")
	}

	now := s.nowFunc()
	trail, err := buildTrail(userID, req, now)
	if err != nil {
		return nil, 0, err
	}

	s.trailMu.Lock()
	defer s.trailMu.Unlock()

	collection := dailyTrailCollection(userID, now)
	trails, err := s.readDailyTrails(collection)
	if err != nil {
		return nil, 0, err
	}

	trails = append(trails, *trail)
	if len(trails) > maxDailyTrailCount {
		trails = trails[len(trails)-maxDailyTrailCount:]
	}

	if err = s.sm.DB().Write(collection, trailDailyResource, trails); err != nil {
		return nil, 0, fmt.Errorf("保存浏览记录失败: %w", err)
	}

	return trail, len(trails), nil
}

// CleanTodayTrail 清空当前登录用户当天的浏览记录；即使当天尚无文件，也会写入一个空数组保持日文件结构稳定。
func (s *Service) CleanTodayTrail(ctx context.Context, userID int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if userID <= 0 {
		return 0, ErrInvalidTrailUserID
	}

	s.trailMu.Lock()
	defer s.trailMu.Unlock()

	now := s.nowFunc()
	collection := dailyTrailCollection(userID, now)
	trails, err := s.readDailyTrails(collection)
	if err != nil {
		return 0, err
	}

	if err = s.sm.DB().Write(collection, trailDailyResource, []filedb.Trail{}); err != nil {
		return 0, fmt.Errorf("清空今日浏览记录失败: %w", err)
	}

	return len(trails), nil
}

func buildTrail(userID int, req *request.TrailAdd, now time.Time) (*filedb.Trail, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" || len([]rune(title)) > maxTrailTitleRunes {
		return nil, ErrInvalidTrailTitle
	}

	normalizedURL, err := normalizeTrailURL(req.URL)
	if err != nil {
		return nil, err
	}

	return &filedb.Trail{
		Title:     title,
		URL:       normalizedURL,
		UserID:    userID,
		CreatedAt: now.Unix(),
	}, nil
}

func normalizeTrailURL(rawURL string) (string, error) {
	normalizedURL := strings.TrimSpace(rawURL)
	if normalizedURL == "" || len([]byte(normalizedURL)) > maxTrailURLBytes {
		return "", ErrInvalidTrailURL
	}

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil || parsedURL.Host == "" {
		return "", ErrInvalidTrailURL
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", ErrInvalidTrailURL
	}

	return parsedURL.String(), nil
}

func (s *Service) readDailyTrails(collection string) ([]filedb.Trail, error) {
	trails := make([]filedb.Trail, 0)
	if err := s.sm.DB().Read(collection, trailDailyResource, &trails); err != nil {
		if os.IsNotExist(err) {
			return trails, nil
		}
		return nil, fmt.Errorf("读取今日浏览记录失败: %w", err)
	}

	return trails, nil
}

func dailyTrailCollection(userID int, now time.Time) string {
	return trailRootCollection + "/" + strconv.Itoa(userID) + "/" + now.Format("20060102")
}
