package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
)

func TestAddTrailCreatesDailyFile(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	trail, total, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   " https://example.com/path?from=test ",
		Title: " 示例页面 ",
	})
	if err != nil {
		t.Fatalf("添加浏览记录失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("期望总数为 1，实际为 %d", total)
	}
	if trail.Title != "示例页面" {
		t.Fatalf("期望标题被去除空白，实际为 %q", trail.Title)
	}
	if trail.URL != "https://example.com/path?from=test" {
		t.Fatalf("期望 URL 被规范化，实际为 %q", trail.URL)
	}
	if trail.UserID != 1 {
		t.Fatalf("期望用户 ID 为 1，实际为 %d", trail.UserID)
	}
	if trail.CreatedAt != fixedNow.Unix() {
		t.Fatalf("期望创建时间为 %d，实际为 %d", fixedNow.Unix(), trail.CreatedAt)
	}

	storedTrails := readStoredTrails(t, svc, 1, fixedNow)
	if len(storedTrails) != 1 {
		t.Fatalf("期望文件中有 1 条记录，实际为 %d", len(storedTrails))
	}
	if storedTrails[0].URL != trail.URL {
		t.Fatalf("期望文件记录 URL 为 %q，实际为 %q", trail.URL, storedTrails[0].URL)
	}
}

func TestAddTrailKeepsLatestTwoThousand(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	existingTrails := make([]filedb.Trail, maxDailyTrailCount)
	for i := 0; i < maxDailyTrailCount; i++ {
		index := i + 1
		existingTrails[i] = filedb.Trail{
			Title:     fmt.Sprintf("第%d条", index),
			URL:       fmt.Sprintf("https://example.com/%d", index),
			UserID:    1,
			CreatedAt: fixedNow.Unix(),
		}
	}

	collection := dailyTrailCollection(1, fixedNow)
	if err := svc.sm.DB().Write(collection, trailDailyResource, existingTrails); err != nil {
		t.Fatalf("写入测试浏览记录失败: %v", err)
	}

	_, total, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/new",
		Title: "新记录",
	})
	if err != nil {
		t.Fatalf("添加第 2001 条浏览记录失败: %v", err)
	}
	if total != maxDailyTrailCount {
		t.Fatalf("期望总数仍为 %d，实际为 %d", maxDailyTrailCount, total)
	}

	storedTrails := readStoredTrails(t, svc, 1, fixedNow)
	if len(storedTrails) != maxDailyTrailCount {
		t.Fatalf("期望文件中保留 %d 条记录，实际为 %d", maxDailyTrailCount, len(storedTrails))
	}
	if storedTrails[0].Title != "第2条" {
		t.Fatalf("期望最早的第1条被删除，当前首条为 %q", storedTrails[0].Title)
	}
	lastTrail := storedTrails[len(storedTrails)-1]
	if lastTrail.Title != "新记录" {
		t.Fatalf("期望最后一条为新记录，实际为 %q", lastTrail.Title)
	}
}

func TestAddTrailRejectInvalidInput(t *testing.T) {
	svc := newTestService(t)

	_, _, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "ftp://example.com/file",
		Title: "非法协议",
	})
	if !errors.Is(err, ErrInvalidTrailURL) {
		t.Fatalf("期望非法 URL 返回 ErrInvalidTrailURL，实际为 %v", err)
	}

	_, _, err = svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com",
		Title: "   ",
	})
	if !errors.Is(err, ErrInvalidTrailTitle) {
		t.Fatalf("期望空标题返回 ErrInvalidTrailTitle，实际为 %v", err)
	}

	_, _, err = svc.AddTrail(context.Background(), 0, &request.TrailAdd{
		URL:   "https://example.com",
		Title: "示例页面",
	})
	if !errors.Is(err, ErrInvalidTrailUserID) {
		t.Fatalf("期望非法用户 ID 返回 ErrInvalidTrailUserID，实际为 %v", err)
	}
}

func TestCleanTodayTrail(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	for i := 0; i < 2; i++ {
		_, _, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
			URL:   fmt.Sprintf("https://example.com/%d", i),
			Title: fmt.Sprintf("记录%d", i),
		})
		if err != nil {
			t.Fatalf("添加测试浏览记录失败: %v", err)
		}
	}

	deleted, err := svc.CleanTodayTrail(context.Background(), 1)
	if err != nil {
		t.Fatalf("清空今日浏览记录失败: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("期望删除 2 条记录，实际为 %d", deleted)
	}

	storedTrails := readStoredTrails(t, svc, 1, fixedNow)
	if len(storedTrails) != 0 {
		t.Fatalf("期望清空后文件中无记录，实际为 %d", len(storedTrails))
	}

	deleted, err = svc.CleanTodayTrail(context.Background(), 2)
	if err != nil {
		t.Fatalf("清空不存在的今日浏览记录失败: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("期望不存在文件时删除 0 条，实际为 %d", deleted)
	}
}

func readStoredTrails(t *testing.T, svc *Service, userID int, now time.Time) []filedb.Trail {
	t.Helper()

	trails := make([]filedb.Trail, 0)
	collection := dailyTrailCollection(userID, now)
	if err := svc.sm.DB().Read(collection, trailDailyResource, &trails); err != nil {
		t.Fatalf("读取已保存浏览记录失败: %v", err)
	}
	return trails
}
