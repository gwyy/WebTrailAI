package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
)

const summaryListDefaultLimit = 30

var (
	ErrInvalidSummaryDate = errors.New("日期格式无效，应为 YYYYMMDD 或 YYYY-MM-DD")
	ErrSummaryNotFound    = errors.New("未找到对应日期的每日总结")
)

type SummaryListItem struct {
	Date      string
	Summary   string
	PartCount int
}

type SummaryDetailResult struct {
	Date    string
	Summary string
	Parts   []filedb.SummaryPart
}

// ListUserRecentSummaries 读取当前用户最近的每日总结，默认最多返回 30 条，按日期倒序输出。
func (s *Service) ListUserRecentSummaries(ctx context.Context, userID int) ([]SummaryListItem, error) {
	return s.listUserSummaries(ctx, userID, summaryListDefaultLimit)
}

// GetUserSummaryDetail 按日期读取当前用户的每日总结详情，支持 YYYYMMDD 和 YYYY-MM-DD 两种日期格式。
func (s *Service) GetUserSummaryDetail(ctx context.Context, userID int, dateText string) (*SummaryDetailResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrInvalidTrailUserID
	}

	targetDate, canonicalDate, err := parseSummaryDate(dateText)
	if err != nil {
		return nil, err
	}

	summary := filedb.Summary{}
	if err = s.sm.DB().Read(summaryCollection(userID, targetDate), summaryResource, &summary); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSummaryNotFound
		}
		return nil, fmt.Errorf("读取每日总结详情失败: %w", err)
	}

	parts := make([]filedb.SummaryPart, len(summary.Pars))
	for i, part := range summary.Pars {
		titles := make([]string, 0, len(part.Titles))
		for _, title := range part.Titles {
			title = strings.TrimSpace(title)
			if title == "" {
				continue
			}
			titles = append(titles, title)
		}
		parts[i] = filedb.SummaryPart{
			Index:   part.Index,
			Titles:  titles,
			Content: part.Content,
		}
	}
	return &SummaryDetailResult{
		Date:    canonicalDate,
		Summary: strings.TrimSpace(summary.Summary),
		Parts:   parts,
	}, nil
}

func (s *Service) listUserSummaries(ctx context.Context, userID int, limit int) ([]SummaryListItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrInvalidTrailUserID
	}
	if limit <= 0 {
		limit = summaryListDefaultLimit
	}

	entries, err := os.ReadDir(summaryUserDir(s.cfg.GetString("db.filedir"), userID))
	if err != nil {
		if os.IsNotExist(err) {
			return []SummaryListItem{}, nil
		}
		return nil, fmt.Errorf("读取每日总结目录失败: %w", err)
	}

	dates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, _, err = parseSummaryDate(entry.Name()); err != nil {
			continue
		}
		dates = append(dates, entry.Name())
	}

	sort.Slice(dates, func(i int, j int) bool {
		return dates[i] > dates[j]
	})
	if len(dates) > limit {
		dates = dates[:limit]
	}

	list := make([]SummaryListItem, 0, len(dates))
	for _, dateText := range dates {
		targetDate, _, err := parseSummaryDate(dateText)
		if err != nil {
			continue
		}

		summary := filedb.Summary{}
		if err = s.sm.DB().Read(summaryCollection(userID, targetDate), summaryResource, &summary); err != nil {
			s.log.Warnf("读取每日总结列表项失败: user_id=%d date=%s err=%v", userID, dateText, err)
			continue
		}

		list = append(list, SummaryListItem{
			Date:      dateText,
			Summary:   strings.TrimSpace(summary.Summary),
			PartCount: len(summary.Pars),
		})
	}

	return list, nil
}

func parseSummaryDate(dateText string) (time.Time, string, error) {
	normalizedDate := strings.TrimSpace(dateText)
	if normalizedDate == "" {
		return time.Time{}, "", ErrInvalidSummaryDate
	}

	layout := "20060102"
	if len(normalizedDate) == len("2006-01-02") && strings.Count(normalizedDate, "-") == 2 {
		layout = "2006-01-02"
	}

	targetDate, err := time.ParseInLocation(layout, normalizedDate, time.Local)
	if err != nil {
		return time.Time{}, "", ErrInvalidSummaryDate
	}

	return targetDate, targetDate.Format("20060102"), nil
}

func summaryUserDir(dbDir string, userID int) string {
	return filepath.Join(dbDir, summaryRootCollection, strconv.Itoa(userID))
}
