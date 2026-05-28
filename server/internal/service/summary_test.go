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

type fakeLLMClient struct {
	responses []string
	errAt     int
	calls     []string
}

func (f *fakeLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f.calls = append(f.calls, prompt)
	callIndex := len(f.calls)
	if f.errAt > 0 && callIndex == f.errAt {
		return "", errors.New("模拟大模型失败")
	}
	if callIndex <= len(f.responses) {
		return f.responses[callIndex-1], nil
	}
	return fmt.Sprintf("总结%d", callIndex), nil
}

func TestGenerateUserDailySummarySplitsTrailsAndWritesSummary(t *testing.T) {
	svc := newTestService(t)
	llm := &fakeLLMClient{}
	svc.SetLLMClient(llm)
	targetDate := time.Date(2026, 5, 24, 0, 0, 0, 0, time.Local)

	writeDailyTrails(t, svc, 1, targetDate, buildTestTrails(1, 250))

	result, err := svc.GenerateUserDailySummary(context.Background(), 1, targetDate)
	if err != nil {
		t.Fatalf("生成用户浏览总结失败: %v", err)
	}
	if result.Skipped {
		t.Fatalf("有浏览记录时不应跳过")
	}
	if result.TrailSize != 250 {
		t.Fatalf("期望处理 250 条记录，实际为 %d", result.TrailSize)
	}
	if result.PartSize != 3 {
		t.Fatalf("期望拆成 3 个分片，实际为 %d", result.PartSize)
	}
	if len(llm.calls) != 4 {
		t.Fatalf("期望调用大模型 4 次，实际为 %d", len(llm.calls))
	}

	summary := readStoredSummary(t, svc, 1, targetDate)
	if len(summary.Pars) != 3 {
		t.Fatalf("期望落盘 3 个分片总结，实际为 %d", len(summary.Pars))
	}
	for i, part := range summary.Pars {
		expectedIndex := i + 1
		if part.Index != expectedIndex {
			t.Fatalf("期望第 %d 个分片 index 为 %d，实际为 %d", expectedIndex, expectedIndex, part.Index)
		}
		expectedTitleCount := 100
		if i == 2 {
			expectedTitleCount = 50
		}
		if len(part.Titles) != expectedTitleCount {
			t.Fatalf("期望第 %d 个分片 titles 数量为 %d，实际为 %d", expectedIndex, expectedTitleCount, len(part.Titles))
		}
		expectedFirstTitle := fmt.Sprintf("测试页面标题%04d", i*summaryTrailBatchSize+1)
		if part.Titles[0] != expectedFirstTitle {
			t.Fatalf("期望第 %d 个分片首个 title 为 %q，实际为 %q", expectedIndex, expectedFirstTitle, part.Titles[0])
		}
	}
	if summary.Summary != "总结4" {
		t.Fatalf("期望最终总结来自第 4 次调用，实际为 %q", summary.Summary)
	}
	finalPrompt := llm.calls[len(llm.calls)-1]
	for _, expected := range []string{"浏览标题：", "测试页面标题0001", "测试页面标题0250", "分片总结：总结3"} {
		if !strings.Contains(finalPrompt, expected) {
			t.Fatalf("期望最终总结 prompt 包含 %q，实际 prompt 为：%s", expected, finalPrompt)
		}
	}
}

func TestListUserRecentSummariesReturnsLatestThirtyInDescendingOrder(t *testing.T) {
	svc := newTestService(t)

	for index := 0; index < 35; index++ {
		targetDate := time.Date(2026, 5, 1+index, 0, 0, 0, 0, time.Local)
		writeStoredSummary(t, svc, 1, targetDate, filedb.Summary{
			Pars: []filedb.SummaryPart{
				{Index: 1, Content: fmt.Sprintf("分片总结%d", index+1)},
			},
			Summary: fmt.Sprintf("用户一总结%d", index+1),
		})
	}
	writeStoredSummary(t, svc, 2, time.Date(2026, 6, 5, 0, 0, 0, 0, time.Local), filedb.Summary{
		Summary: "用户二总结",
	})

	list, err := svc.ListUserRecentSummaries(context.Background(), 1)
	if err != nil {
		t.Fatalf("读取用户最近每日总结失败: %v", err)
	}
	if len(list) != 30 {
		t.Fatalf("期望返回最近 30 条每日总结，实际为 %d", len(list))
	}
	if list[0].Date != "20260604" {
		t.Fatalf("期望第一条是最新日期 20260604，实际为 %s", list[0].Date)
	}
	if list[0].Summary != "用户一总结35" {
		t.Fatalf("期望第一条摘要为 用户一总结35，实际为 %s", list[0].Summary)
	}
	if list[0].PartCount != 1 {
		t.Fatalf("期望第一条分片数为 1，实际为 %d", list[0].PartCount)
	}
	if list[len(list)-1].Date != "20260506" {
		t.Fatalf("期望最后一条是第 30 新的日期 20260506，实际为 %s", list[len(list)-1].Date)
	}
}

func TestGetUserSummaryDetailSupportsHyphenatedDate(t *testing.T) {
	svc := newTestService(t)
	targetDate := time.Date(2026, 5, 24, 0, 0, 0, 0, time.Local)
	writeStoredSummary(t, svc, 1, targetDate, filedb.Summary{
		Pars: []filedb.SummaryPart{
			{Index: 1, Content: "第一段总结"},
			{Index: 2, Content: "第二段总结"},
		},
		Summary: "最终每日总结",
	})

	detail, err := svc.GetUserSummaryDetail(context.Background(), 1, "2026-05-24")
	if err != nil {
		t.Fatalf("按带横线日期读取每日总结详情失败: %v", err)
	}
	if detail.Date != "20260524" {
		t.Fatalf("期望详情日期为 20260524，实际为 %s", detail.Date)
	}
	if detail.Summary != "最终每日总结" {
		t.Fatalf("期望详情摘要为 最终每日总结，实际为 %s", detail.Summary)
	}
	if len(detail.Parts) != 2 {
		t.Fatalf("期望详情返回 2 个分片，实际为 %d", len(detail.Parts))
	}
}

func TestGetUserSummaryDetailReturnsNotFoundForMissingDate(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.GetUserSummaryDetail(context.Background(), 1, "20260524")
	if !errors.Is(err, ErrSummaryNotFound) {
		t.Fatalf("期望缺失日期返回 ErrSummaryNotFound，实际为 %v", err)
	}
}

func TestGenerateUserDailySummaryKeepsLatestTwoThousand(t *testing.T) {
	svc := newTestService(t)
	llm := &fakeLLMClient{}
	svc.SetLLMClient(llm)
	targetDate := time.Date(2026, 5, 24, 0, 0, 0, 0, time.Local)

	writeDailyTrails(t, svc, 1, targetDate, buildTestTrails(1, 2100))

	result, err := svc.GenerateUserDailySummary(context.Background(), 1, targetDate)
	if err != nil {
		t.Fatalf("生成用户浏览总结失败: %v", err)
	}
	if result.TrailSize != summaryMaxTrailCount {
		t.Fatalf("期望最多处理 %d 条记录，实际为 %d", summaryMaxTrailCount, result.TrailSize)
	}
	if result.PartSize != summaryMaxPartCount {
		t.Fatalf("期望最多生成 %d 个分片，实际为 %d", summaryMaxPartCount, result.PartSize)
	}
	if len(llm.calls) != summaryMaxPartCount+1 {
		t.Fatalf("期望调用大模型 %d 次，实际为 %d", summaryMaxPartCount+1, len(llm.calls))
	}
	if strings.Contains(llm.calls[0], "测试页面标题0001") {
		t.Fatalf("超过 2000 条时应只保留最新记录，首个分片不应包含最旧记录")
	}
	if !strings.Contains(llm.calls[0], "测试页面标题0101") {
		t.Fatalf("超过 2000 条时首个分片应从第 101 条开始")
	}

	summary := readStoredSummary(t, svc, 1, targetDate)
	if summary.Pars[0].Titles[0] != "测试页面标题0101" {
		t.Fatalf("超过 2000 条时首个 titles 应从第 101 条开始，实际为 %q", summary.Pars[0].Titles[0])
	}
	lastPart := summary.Pars[len(summary.Pars)-1]
	lastTitle := lastPart.Titles[len(lastPart.Titles)-1]
	if lastTitle != "测试页面标题2100" {
		t.Fatalf("超过 2000 条时最后一个 titles 应保留最新记录，实际为 %q", lastTitle)
	}
}

func TestGenerateUserDailySummarySkipsEmptyTrails(t *testing.T) {
	svc := newTestService(t)
	llm := &fakeLLMClient{}
	svc.SetLLMClient(llm)
	targetDate := time.Date(2026, 5, 24, 0, 0, 0, 0, time.Local)

	result, err := svc.GenerateUserDailySummary(context.Background(), 1, targetDate)
	if err != nil {
		t.Fatalf("无浏览记录时不应返回错误: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("无浏览记录时应跳过")
	}
	if len(llm.calls) != 0 {
		t.Fatalf("无浏览记录时不应调用大模型，实际调用 %d 次", len(llm.calls))
	}
}

func TestGenerateDailySummariesContinuesWhenOneUserFails(t *testing.T) {
	svc := newTestService(t)
	llm := &fakeLLMClient{errAt: 1}
	svc.SetLLMClient(llm)
	targetDate := time.Date(2026, 5, 24, 0, 0, 0, 0, time.Local)

	if _, err := svc.Register(context.Background(), newUserRegister("alice")); err != nil {
		t.Fatalf("注册用户 alice 失败: %v", err)
	}
	if _, err := svc.Register(context.Background(), newUserRegister("bob")); err != nil {
		t.Fatalf("注册用户 bob 失败: %v", err)
	}
	writeDailyTrails(t, svc, 1, targetDate, buildTestTrails(1, 1))
	writeDailyTrails(t, svc, 2, targetDate, buildTestTrails(2, 1))

	result, err := svc.GenerateDailySummaries(context.Background(), targetDate)
	if err != nil {
		t.Fatalf("生成全部用户浏览总结失败: %v", err)
	}
	if result.UserCount != 2 {
		t.Fatalf("期望处理 2 个用户，实际为 %d", result.UserCount)
	}
	if result.FailedCount != 1 {
		t.Fatalf("期望 1 个用户失败，实际为 %d", result.FailedCount)
	}
	if result.SuccessCount != 1 {
		t.Fatalf("期望 1 个用户成功，实际为 %d", result.SuccessCount)
	}

	summaryCount := 0
	for _, userID := range []int{1, 2} {
		if summaryExists(t, svc, userID, targetDate) {
			summaryCount++
		}
	}
	if summaryCount != 1 {
		t.Fatalf("期望只有 1 个用户成功生成浏览总结，实际为 %d", summaryCount)
	}
}

func TestGenerateDailySummariesRequiresLLMClient(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.GenerateDailySummaries(context.Background(), time.Now())
	if !errors.Is(err, ErrSummaryLLMNotConfigured) {
		t.Fatalf("期望未配置大模型客户端时返回 ErrSummaryLLMNotConfigured，实际为 %v", err)
	}
}

func buildTestTrails(userID int, count int) []filedb.Trail {
	trails := make([]filedb.Trail, count)
	for i := 0; i < count; i++ {
		index := i + 1
		trails[i] = filedb.Trail{
			Title:     fmt.Sprintf("测试页面标题%04d", index),
			URL:       fmt.Sprintf("https://example.com/%04d", index),
			InnerText: fmt.Sprintf("这是第 %d 条测试正文内容，用于验证浏览总结分片逻辑。", index),
			UserID:    userID,
			CreatedAt: time.Date(2026, 5, 24, 12, 0, 0, 0, time.Local).Unix(),
		}
	}
	return trails
}

func writeDailyTrails(t *testing.T, svc *Service, userID int, targetDate time.Time, trails []filedb.Trail) {
	t.Helper()

	if err := svc.sm.DB().Write(dailyTrailCollection(userID, targetDate), trailDailyResource, trails); err != nil {
		t.Fatalf("写入测试浏览记录失败: %v", err)
	}
}

func readStoredSummary(t *testing.T, svc *Service, userID int, targetDate time.Time) filedb.Summary {
	t.Helper()

	summary := filedb.Summary{}
	if err := svc.sm.DB().Read(summaryCollection(userID, targetDate), summaryResource, &summary); err != nil {
		t.Fatalf("读取已保存浏览总结失败: %v", err)
	}
	return summary
}

func writeStoredSummary(t *testing.T, svc *Service, userID int, targetDate time.Time, summary filedb.Summary) {
	t.Helper()

	if err := svc.sm.DB().Write(summaryCollection(userID, targetDate), summaryResource, summary); err != nil {
		t.Fatalf("写入测试每日总结失败: %v", err)
	}
}

func summaryExists(t *testing.T, svc *Service, userID int, targetDate time.Time) bool {
	t.Helper()

	summary := filedb.Summary{}
	return svc.sm.DB().Read(summaryCollection(userID, targetDate), summaryResource, &summary) == nil
}

func newUserRegister(username string) *request.UserRegister {
	return &request.UserRegister{
		Username: username,
		Password: "secret123",
	}
}
