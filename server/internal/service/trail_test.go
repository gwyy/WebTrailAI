package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	result, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:       " https://example.com/path?from=test ",
		Title:     " 示例页面标题内容 ",
		InnerText: "第一段正文\n第二段正文",
	})
	if err != nil {
		t.Fatalf("添加浏览记录失败: %v", err)
	}
	if result.Filtered {
		t.Fatalf("期望浏览记录正常写入，实际被过滤，原因：%s", result.Reason)
	}
	if result.Total != 1 {
		t.Fatalf("期望总数为 1，实际为 %d", result.Total)
	}
	if result.Trail.Title != "示例页面标题内容" {
		t.Fatalf("期望标题被去除空白，实际为 %q", result.Trail.Title)
	}
	if result.Trail.URL != "https://example.com/path?from=test" {
		t.Fatalf("期望 URL 被规范化，实际为 %q", result.Trail.URL)
	}
	if result.Trail.InnerText != "第一段正文\n第二段正文" {
		t.Fatalf("期望正文原样保存，实际为 %q", result.Trail.InnerText)
	}
	if result.Trail.UserID != 1 {
		t.Fatalf("期望用户 ID 为 1，实际为 %d", result.Trail.UserID)
	}
	if result.Trail.CreatedAt != fixedNow.Unix() {
		t.Fatalf("期望创建时间为 %d，实际为 %d", fixedNow.Unix(), result.Trail.CreatedAt)
	}

	storedTrails := readStoredTrails(t, svc, 1, fixedNow)
	if len(storedTrails) != 1 {
		t.Fatalf("期望文件中有 1 条记录，实际为 %d", len(storedTrails))
	}
	if storedTrails[0].URL != result.Trail.URL {
		t.Fatalf("期望文件记录 URL 为 %q，实际为 %q", result.Trail.URL, storedTrails[0].URL)
	}
	if storedTrails[0].InnerText != result.Trail.InnerText {
		t.Fatalf("期望文件记录正文为 %q，实际为 %q", result.Trail.InnerText, storedTrails[0].InnerText)
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
			Title:     fmt.Sprintf("历史浏览记录标题%d", index),
			URL:       fmt.Sprintf("https://example.com/%d", index),
			UserID:    1,
			CreatedAt: fixedNow.Unix(),
		}
	}

	collection := dailyTrailCollection(1, fixedNow)
	if err := svc.sm.DB().Write(collection, trailDailyResource, existingTrails); err != nil {
		t.Fatalf("写入测试浏览记录失败: %v", err)
	}

	result, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/new",
		Title: "最新浏览记录标题",
	})
	if err != nil {
		t.Fatalf("添加第 2001 条浏览记录失败: %v", err)
	}
	if result.Filtered {
		t.Fatalf("期望第 2001 条浏览记录正常写入，实际被过滤，原因：%s", result.Reason)
	}
	if result.Total != maxDailyTrailCount {
		t.Fatalf("期望总数仍为 %d，实际为 %d", maxDailyTrailCount, result.Total)
	}

	storedTrails := readStoredTrails(t, svc, 1, fixedNow)
	if len(storedTrails) != maxDailyTrailCount {
		t.Fatalf("期望文件中保留 %d 条记录，实际为 %d", maxDailyTrailCount, len(storedTrails))
	}
	if storedTrails[0].Title != "历史浏览记录标题2" {
		t.Fatalf("期望最早的第1条被删除，当前首条为 %q", storedTrails[0].Title)
	}
	lastTrail := storedTrails[len(storedTrails)-1]
	if lastTrail.Title != "最新浏览记录标题" {
		t.Fatalf("期望最后一条为新记录，实际为 %q", lastTrail.Title)
	}
}

func TestAddTrailRejectInvalidInput(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "ftp://example.com/file",
		Title: "非法协议标题内容",
	})
	if !errors.Is(err, ErrInvalidTrailURL) {
		t.Fatalf("期望非法 URL 返回 ErrInvalidTrailURL，实际为 %v", err)
	}

	_, err = svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com",
		Title: strings.Repeat("长", maxTrailTitleRunes+1),
	})
	if !errors.Is(err, ErrInvalidTrailTitle) {
		t.Fatalf("期望超长标题返回 ErrInvalidTrailTitle，实际为 %v", err)
	}

	_, err = svc.AddTrail(context.Background(), 0, &request.TrailAdd{
		URL:   "https://example.com",
		Title: "示例页面标题内容",
	})
	if !errors.Is(err, ErrInvalidTrailUserID) {
		t.Fatalf("期望非法用户 ID 返回 ErrInvalidTrailUserID，实际为 %v", err)
	}
}

func TestAddTrailTruncatesInnerTextToFiveThousandRunes(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	rawInnerText := strings.Repeat("正文", 3000)
	expectedInnerText := string([]rune(rawInnerText)[:maxTrailInnerTextRunes])

	result, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:       "https://example.com/long-inner-text",
		Title:     "正文截断测试标题",
		InnerText: rawInnerText,
	})
	if err != nil {
		t.Fatalf("添加超长正文浏览记录失败: %v", err)
	}
	if result.Filtered {
		t.Fatalf("超长正文不应触发过滤，实际原因：%s", result.Reason)
	}
	if result.Trail.InnerText != expectedInnerText {
		t.Fatalf("期望正文被截断为前 5000 个字符，实际长度为 %d", len([]rune(result.Trail.InnerText)))
	}
	if len([]rune(result.Trail.InnerText)) != maxTrailInnerTextRunes {
		t.Fatalf("期望正文长度为 %d，实际为 %d", maxTrailInnerTextRunes, len([]rune(result.Trail.InnerText)))
	}

	storedTrails := readStoredTrails(t, svc, 1, fixedNow)
	if storedTrails[0].InnerText != expectedInnerText {
		t.Fatalf("期望落盘正文为截断后的内容，实际长度为 %d", len([]rune(storedTrails[0].InnerText)))
	}
}

func TestAddTrailFiltersShortTitle(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	result, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/short",
		Title: "七字标题文本呀",
	})
	if err != nil {
		t.Fatalf("短标题过滤不应返回错误，实际为：%v", err)
	}
	if !result.Filtered {
		t.Fatalf("期望短标题被过滤")
	}
	if result.Reason != TrailFilterReasonTitleTooShort {
		t.Fatalf("期望过滤原因为 %s，实际为 %s", TrailFilterReasonTitleTooShort, result.Reason)
	}
	if result.Total != 0 {
		t.Fatalf("期望过滤后总数为 0，实际为 %d", result.Total)
	}
	if result.Trail != nil {
		t.Fatalf("期望过滤后不返回浏览记录，实际为 %+v", result.Trail)
	}

	trails := make([]filedb.Trail, 0)
	collection := dailyTrailCollection(1, fixedNow)
	if err = svc.sm.DB().Read(collection, trailDailyResource, &trails); err == nil {
		t.Fatalf("短标题过滤后不应创建今日浏览记录文件")
	}
}

func TestAddTrailAllowsEightRuneTitle(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	result, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/eight",
		Title: "正好八个字标题啊",
	})
	if err != nil {
		t.Fatalf("添加 8 字标题失败: %v", err)
	}
	if result.Filtered {
		t.Fatalf("8 字标题不应被过滤，实际原因：%s", result.Reason)
	}
	if result.Total != 1 {
		t.Fatalf("期望总数为 1，实际为 %d", result.Total)
	}
}

func TestAddTrailFiltersDuplicateTitleInSameDay(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	firstResult, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/first",
		Title: "重复页面标题内容",
	})
	if err != nil {
		t.Fatalf("添加首条浏览记录失败: %v", err)
	}
	if firstResult.Filtered {
		t.Fatalf("首条浏览记录不应被过滤")
	}

	secondResult, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/second",
		Title: " 重复页面标题内容 ",
	})
	if err != nil {
		t.Fatalf("重复标题过滤不应返回错误，实际为：%v", err)
	}
	if !secondResult.Filtered {
		t.Fatalf("期望同日重复标题被过滤")
	}
	if secondResult.Reason != TrailFilterReasonDuplicateTitle {
		t.Fatalf("期望过滤原因为 %s，实际为 %s", TrailFilterReasonDuplicateTitle, secondResult.Reason)
	}
	if secondResult.Total != 1 {
		t.Fatalf("期望重复过滤后总数仍为 1，实际为 %d", secondResult.Total)
	}

	storedTrails := readStoredTrails(t, svc, 1, fixedNow)
	if len(storedTrails) != 1 {
		t.Fatalf("期望文件中仍只有 1 条记录，实际为 %d", len(storedTrails))
	}
	if storedTrails[0].URL != "https://example.com/first" {
		t.Fatalf("期望保留首条浏览记录，实际 URL 为 %q", storedTrails[0].URL)
	}
}

func TestAddTrailAllowsSameTitleForDifferentDayOrUser(t *testing.T) {
	svc := newTestService(t)
	firstDay := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	secondDay := time.Date(2026, 5, 23, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return firstDay
	}

	if _, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/user-one-first-day",
		Title: "相同标题不同范围",
	}); err != nil {
		t.Fatalf("添加用户 1 第一天浏览记录失败: %v", err)
	}

	svc.nowFunc = func() time.Time {
		return secondDay
	}
	result, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
		URL:   "https://example.com/user-one-second-day",
		Title: "相同标题不同范围",
	})
	if err != nil {
		t.Fatalf("添加用户 1 第二天浏览记录失败: %v", err)
	}
	if result.Filtered {
		t.Fatalf("跨日期相同标题不应被过滤，实际原因：%s", result.Reason)
	}

	svc.nowFunc = func() time.Time {
		return firstDay
	}
	result, err = svc.AddTrail(context.Background(), 2, &request.TrailAdd{
		URL:   "https://example.com/user-two-first-day",
		Title: "相同标题不同范围",
	})
	if err != nil {
		t.Fatalf("添加用户 2 第一天浏览记录失败: %v", err)
	}
	if result.Filtered {
		t.Fatalf("跨用户相同标题不应被过滤，实际原因：%s", result.Reason)
	}
}

func TestCleanTodayTrail(t *testing.T) {
	svc := newTestService(t)
	fixedNow := time.Date(2026, 5, 22, 10, 30, 0, 0, time.Local)
	svc.nowFunc = func() time.Time {
		return fixedNow
	}

	for i := 0; i < 2; i++ {
		_, err := svc.AddTrail(context.Background(), 1, &request.TrailAdd{
			URL:   fmt.Sprintf("https://example.com/%d", i),
			Title: fmt.Sprintf("清空测试记录标题%d", i),
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
