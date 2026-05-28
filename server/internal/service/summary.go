package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
)

const (
	summaryRootCollection      = "summary"
	summaryResource            = "summary"
	summaryTrailBatchSize      = 100
	summaryMaxPartCount        = 20
	summaryMaxTrailCount       = summaryTrailBatchSize * summaryMaxPartCount
	summaryPromptMaxTextRunes  = 300
	summaryPromptMaxTitleRunes = 120
)

var (
	ErrSummaryLLMNotConfigured = errors.New("大模型客户端未配置")
)

type SummaryRunResult struct {
	Date         string
	UserCount    int
	SuccessCount int
	SkippedCount int
	FailedCount  int
}

type SummaryUserResult struct {
	UserID    int
	Date      string
	TrailSize int
	PartSize  int
	Skipped   bool
}

// GenerateYesterdaySummaries 生成前一天全部账号的浏览总结，供凌晨定时任务调用。
func (s *Service) GenerateYesterdaySummaries(ctx context.Context) (*SummaryRunResult, error) {
	targetDate := s.nowFunc().AddDate(0, 0, -1)
	return s.GenerateDailySummaries(ctx, targetDate)
}

// RunYesterdaySummaryOnce 串行执行一次昨日浏览总结生成，供定时任务和调试接口共用。
func (s *Service) RunYesterdaySummaryOnce(ctx context.Context) (*SummaryRunResult, error) {
	s.summaryRunMu.Lock()
	defer s.summaryRunMu.Unlock()

	return s.GenerateYesterdaySummaries(ctx)
}

// GenerateDailySummaries 按指定日期读取全部用户的 trails，并逐个生成 summary。
func (s *Service) GenerateDailySummaries(ctx context.Context, targetDate time.Time) (*SummaryRunResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.llm == nil {
		return nil, ErrSummaryLLMNotConfigured
	}

	users, err := s.listActiveUsers()
	if err != nil {
		return nil, err
	}

	result := &SummaryRunResult{
		Date:      targetDate.Format("20060102"),
		UserCount: len(users),
	}
	for _, user := range users {
		if err = ctx.Err(); err != nil {
			return result, err
		}

		userResult, userErr := s.GenerateUserDailySummary(ctx, user.ID, targetDate)
		if userErr != nil {
			result.FailedCount++
			s.log.Errorf("生成用户浏览总结失败: user_id=%d date=%s err=%v", user.ID, result.Date, userErr)
			continue
		}
		if userResult.Skipped {
			result.SkippedCount++
			continue
		}
		result.SuccessCount++
	}

	return result, nil
}

// GenerateUserDailySummary 负责单个用户单日的分片总结和最终总结落盘。
func (s *Service) GenerateUserDailySummary(ctx context.Context, userID int, targetDate time.Time) (*SummaryUserResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrInvalidTrailUserID
	}
	if s.llm == nil {
		return nil, ErrSummaryLLMNotConfigured
	}

	date := targetDate.Format("20060102")
	trails, err := s.readDailyTrails(dailyTrailCollection(userID, targetDate))
	if err != nil {
		return nil, err
	}
	if len(trails) == 0 {
		return &SummaryUserResult{
			UserID:  userID,
			Date:    date,
			Skipped: true,
		}, nil
	}
	if len(trails) > summaryMaxTrailCount {
		trails = trails[len(trails)-summaryMaxTrailCount:]
	}

	parts := make([]filedb.SummaryPart, 0, (len(trails)+summaryTrailBatchSize-1)/summaryTrailBatchSize)
	for index, start := 1, 0; start < len(trails); index, start = index+1, start+summaryTrailBatchSize {
		end := start + summaryTrailBatchSize
		if end > len(trails) {
			end = len(trails)
		}

		content, err := s.llm.Generate(ctx, buildSummaryPartPrompt(userID, date, index, trails[start:end]))
		if err != nil {
			return nil, fmt.Errorf("生成第 %d 个分片总结失败: %w", index, err)
		}
		parts = append(parts, filedb.SummaryPart{
			Index:   index,
			Titles:  collectSummaryPartTitles(trails[start:end]),
			Content: content,
		})
	}

	finalSummary, err := s.llm.Generate(ctx, buildFinalSummaryPrompt(userID, date, parts))
	if err != nil {
		return nil, fmt.Errorf("生成最终总结失败: %w", err)
	}

	summary := &filedb.Summary{
		Pars:    parts,
		Summary: finalSummary,
	}
	if err = s.sm.DB().Write(summaryCollection(userID, targetDate), summaryResource, summary); err != nil {
		return nil, fmt.Errorf("保存浏览总结失败: %w", err)
	}

	return &SummaryUserResult{
		UserID:    userID,
		Date:      date,
		TrailSize: len(trails),
		PartSize:  len(parts),
	}, nil
}

func (s *Service) listActiveUsers() ([]filedb.User, error) {
	records, err := s.sm.DB().ReadAll(userCollection)
	if err != nil {
		if os.IsNotExist(err) {
			return []filedb.User{}, nil
		}
		return nil, fmt.Errorf("读取用户列表失败: %w", err)
	}

	users := make([]filedb.User, 0, len(records))
	for _, record := range records {
		user := filedb.User{}
		if err = json.Unmarshal(record, &user); err != nil {
			return nil, fmt.Errorf("解析用户数据失败: %w", err)
		}
		if user.ID > 0 && user.Status == userStatusActive {
			users = append(users, user)
		}
	}

	return users, nil
}

func buildSummaryPartPrompt(userID int, date string, index int, trails []filedb.Trail) string {
	var builder strings.Builder
	builder.WriteString("你是个人浏览记录分析助手。请总结用户前一天浏览的网页分片，输出中文。\n")
	builder.WriteString("要求：\n")
	builder.WriteString("1. 提炼主要关注主题、关键信息和可能的行动建议。\n")
	builder.WriteString("2. 不要逐条复述，不要编造浏览记录之外的信息。\n")
	builder.WriteString("3. 内容控制在 800 字以内，允许使用 HTML 的 <br> 换行。\n\n")
	builder.WriteString(fmt.Sprintf("用户ID：%d\n日期：%s\n分片序号：%d\n记录数：%d\n\n", userID, date, index, len(trails)))
	builder.WriteString("浏览记录：\n")

	for i, trail := range trails {
		builder.WriteString(fmt.Sprintf("%d. 标题：%s\n", i+1, limitRunes(trail.Title, summaryPromptMaxTitleRunes)))
		builder.WriteString(fmt.Sprintf("   URL：%s\n", trail.URL))
		if strings.TrimSpace(trail.InnerText) != "" {
			builder.WriteString(fmt.Sprintf("   正文摘录：%s\n", normalizePromptText(trail.InnerText, summaryPromptMaxTextRunes)))
		}
	}

	return builder.String()
}

func buildFinalSummaryPrompt(userID int, date string, parts []filedb.SummaryPart) string {
	var builder strings.Builder
	builder.WriteString("你是我聪明、敏锐的私人助理。这是我过去24小时的浏览器记录的汇总。请用**详细、毒舌、直击要害**的风格给我汇报。\n")
	builder.WriteString("要求：\n")
	builder.WriteString("请不要写成那种死板的日报！请用**微信聊天**的口吻，直接和我说人话。\n")
	builder.WriteString("不需要“你好”、“总结如下”这种废话开头，也不要用 # 大标题。用 Emoji 做段落区分即可。\n")
	builder.WriteString("请帮我复盘一下昨天过得怎么样，重点包含这几个维度：\n💻 ")
	builder.WriteString("**昨天搞了什么** \n")
	builder.WriteString("不要罗列网页，告诉我到底解决了什么问题？（比如：别说“看了n8n文档”，要说“终于把n8n那个循环报错的问题搞定了”）。如果看起来是在反复折腾一个技术点，总结一下进度。\n")
	builder.WriteString("**摸鱼和充电**\n")
	builder.WriteString("我看视频或者刷社区的时候，主要在关注哪类内容？（比如：“下午看了好几个黑神话的攻略，看来是手痒了” 或者 “一直在看 NAS 的测评视频”）。\n")
	builder.WriteString("**点评**: 一句话犀利吐槽我的口味。\n🧠")
	builder.WriteString("**脑子里在想啥 (深度洞察)** \n")
	builder.WriteString("这是最重要的！透过浏览记录，分析我当下的**心智焦点**是什么？\n")
	builder.WriteString("（例如：如果我既看了 Docker 文档又搜了服务器价格，说明“你正在筹备一个新的自托管服务，而且已经在做成本预算了”）。\n✅ ")
	//builder.WriteString("**接下来得干啥**\n ")
	//builder.WriteString("根据我看了一半的教程、或者还在调研阶段的商品，直接给我列 3 个具体的下一步行动建议。\n ")
	builder.WriteString("分片总结：\n")

	for _, part := range parts {
		builder.WriteString(fmt.Sprintf("%d. 浏览标题：\n", part.Index))
		for _, title := range part.Titles {
			builder.WriteString(fmt.Sprintf("   - %s\n", limitRunes(title, summaryPromptMaxTitleRunes)))
		}
		builder.WriteString(fmt.Sprintf("   分片总结：%s\n", strings.TrimSpace(part.Content)))
	}

	return builder.String()
}

func collectSummaryPartTitles(trails []filedb.Trail) []string {
	titles := make([]string, 0, len(trails))
	for _, trail := range trails {
		title := strings.TrimSpace(trail.Title)
		if title == "" {
			continue
		}
		titles = append(titles, title)
	}
	return titles
}

func normalizePromptText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.Join(strings.Fields(text), " ")
	return limitRunes(text, maxRunes)
}

func limitRunes(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func summaryCollection(userID int, targetDate time.Time) string {
	return summaryRootCollection + "/" + strconv.Itoa(userID) + "/" + targetDate.Format("20060102")
}
