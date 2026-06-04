package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/ctrl"
	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
	"github.com/gwyy/WebTrailAI/server/internal/service"
	scribble_manager "github.com/gwyy/WebTrailAI/server/pkg/scribble-manager"
)

type testConfig struct {
	values map[string]string
}

func (c *testConfig) GetString(key string) string {
	return c.values[key]
}

func (c *testConfig) GetInt(key string) int {
	return 0
}

func (c *testConfig) GetBool(key string) bool {
	return c.values[key] == "true"
}

func (c *testConfig) Unmarshal(target interface{}) error {
	return nil
}

func (c *testConfig) WatchConfig(onChange func(e fsnotify.Event)) {
}

type testLogger struct{}

func (l *testLogger) Debug(args ...interface{})                        {}
func (l *testLogger) Info(args ...interface{})                         {}
func (l *testLogger) Warn(args ...interface{})                         {}
func (l *testLogger) Error(args ...interface{})                        {}
func (l *testLogger) DPanic(args ...interface{})                       {}
func (l *testLogger) Panic(args ...interface{})                        {}
func (l *testLogger) Fatal(args ...interface{})                        {}
func (l *testLogger) Debugf(template string, args ...interface{})      {}
func (l *testLogger) Infof(template string, args ...interface{})       {}
func (l *testLogger) Warnf(template string, args ...interface{})       {}
func (l *testLogger) Errorf(template string, args ...interface{})      {}
func (l *testLogger) DPanicf(template string, args ...interface{})     {}
func (l *testLogger) Panicf(template string, args ...interface{})      {}
func (l *testLogger) Fatalf(template string, args ...interface{})      {}
func (l *testLogger) Debugw(msg string, keysAndValues ...interface{})  {}
func (l *testLogger) Infow(msg string, keysAndValues ...interface{})   {}
func (l *testLogger) Warnw(msg string, keysAndValues ...interface{})   {}
func (l *testLogger) Errorw(msg string, keysAndValues ...interface{})  {}
func (l *testLogger) DPanicw(msg string, keysAndValues ...interface{}) {}
func (l *testLogger) Panicw(msg string, keysAndValues ...interface{})  {}
func (l *testLogger) Fatalw(msg string, keysAndValues ...interface{})  {}
func (l *testLogger) Sync() error                                      { return nil }

type testRouterEnv struct {
	engine *gin.Engine
	srv    *service.Service
	sm     *scribble_manager.ScribbleManager
}

func newTestRouterEnv(t *testing.T) *testRouterEnv {
	t.Helper()

	cfg := &testConfig{
		values: map[string]string{
			"db.filedir": filepath.Join(t.TempDir(), "filedb"),
			"name":       "web-trail-ai-test",
			"secret":     "test-secret",
		},
	}
	log := &testLogger{}
	sm, err := scribble_manager.NewScribbleManager(cfg, log)
	if err != nil {
		t.Fatalf("初始化 scribble manager 失败: %v", err)
	}

	srv := service.NewService(cfg, log, sm)
	controller := ctrl.NewCtrl(srv, cfg, log)
	newRouter := NewRouter(controller, cfg, log)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	newRouter.Init(engine)

	return &testRouterEnv{
		engine: engine,
		srv:    srv,
		sm:     sm,
	}
}

func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()

	return newTestRouterEnv(t).engine
}

func TestRegisterLoginAndProtectedRoute(t *testing.T) {
	engine := newTestEngine(t)

	registerReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"Route_User","password":"secret123"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	engine.ServeHTTP(registerResp, registerReq)

	if registerResp.Code != http.StatusOK {
		t.Fatalf("注册接口状态码错误，期望 200，实际 %d", registerResp.Code)
	}

	registerBody := map[string]any{}
	if err := json.Unmarshal(registerResp.Body.Bytes(), &registerBody); err != nil {
		t.Fatalf("解析注册响应失败: %v", err)
	}
	if code, ok := registerBody["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("注册响应 code 不正确: %v", registerBody["code"])
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"Route_User","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	engine.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("登录接口状态码错误，期望 200，实际 %d，响应：%s", loginResp.Code, loginResp.Body.String())
	}

	loginBody := map[string]any{}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("解析登录响应失败: %v", err)
	}

	accessToken, ok := loginBody["access_token"].(string)
	if !ok || accessToken == "" {
		t.Fatalf("登录响应中缺少 access_token: %v", loginBody)
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/api/list", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+accessToken)
	protectedResp := httptest.NewRecorder()
	engine.ServeHTTP(protectedResp, protectedReq)

	if protectedResp.Code != http.StatusOK {
		t.Fatalf("受保护接口状态码错误，期望 200，实际 %d，响应：%s", protectedResp.Code, protectedResp.Body.String())
	}

	protectedBody := map[string]any{}
	if err := json.Unmarshal(protectedResp.Body.Bytes(), &protectedBody); err != nil {
		t.Fatalf("解析受保护接口响应失败: %v", err)
	}
	if code, ok := protectedBody["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("受保护接口 code 不正确: %v", protectedBody["code"])
	}

	trailReq := httptest.NewRequest(http.MethodPost, "/api/trailAdd", strings.NewReader(`{"url":"https://example.com/page","title":"路由测试页面标题内容","innerText":"这里是页面正文内容"}`))
	trailReq.Header.Set("Content-Type", "application/json")
	trailReq.Header.Set("Authorization", "Bearer "+accessToken)
	trailResp := httptest.NewRecorder()
	engine.ServeHTTP(trailResp, trailReq)

	if trailResp.Code != http.StatusOK {
		t.Fatalf("添加浏览记录接口状态码错误，期望 200，实际 %d，响应：%s", trailResp.Code, trailResp.Body.String())
	}

	trailBody := map[string]any{}
	if err := json.Unmarshal(trailResp.Body.Bytes(), &trailBody); err != nil {
		t.Fatalf("解析添加浏览记录响应失败: %v", err)
	}
	if code, ok := trailBody["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("添加浏览记录响应 code 不正确: %v", trailBody["code"])
	}
	trailData, ok := trailBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("添加浏览记录响应 data 格式不正确: %v", trailBody["data"])
	}
	if filtered, ok := trailData["filtered"].(bool); !ok || filtered {
		t.Fatalf("期望浏览记录正常写入，实际 filtered 为：%v", trailData["filtered"])
	}
	trailItem, ok := trailData["trail"].(map[string]any)
	if !ok {
		t.Fatalf("添加浏览记录响应 trail 格式不正确: %v", trailData["trail"])
	}
	if _, exists := trailItem["inner_text"]; exists {
		t.Fatalf("添加浏览记录响应不应回传 inner_text，实际为：%v", trailItem["inner_text"])
	}

	cleanReq := httptest.NewRequest(http.MethodPost, "/api/cleanTodayTrail", nil)
	cleanReq.Header.Set("Authorization", "Bearer "+accessToken)
	cleanResp := httptest.NewRecorder()
	engine.ServeHTTP(cleanResp, cleanReq)

	if cleanResp.Code != http.StatusOK {
		t.Fatalf("清空今日浏览记录接口状态码错误，期望 200，实际 %d，响应：%s", cleanResp.Code, cleanResp.Body.String())
	}

	cleanBody := map[string]any{}
	if err := json.Unmarshal(cleanResp.Body.Bytes(), &cleanBody); err != nil {
		t.Fatalf("解析清空今日浏览记录响应失败: %v", err)
	}
	if code, ok := cleanBody["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("清空今日浏览记录响应 code 不正确: %v", cleanBody["code"])
	}
}

func TestTrailAddReturnsFilteredForShortTitle(t *testing.T) {
	engine := newTestEngine(t)
	accessToken := registerAndLoginForTest(t, engine, "Filtered_User")

	trailReq := httptest.NewRequest(http.MethodPost, "/api/trailAdd", strings.NewReader(`{"url":"https://example.com/short","title":"七字标题文本呀"}`))
	trailReq.Header.Set("Content-Type", "application/json")
	trailReq.Header.Set("Authorization", "Bearer "+accessToken)
	trailResp := httptest.NewRecorder()
	engine.ServeHTTP(trailResp, trailReq)

	if trailResp.Code != http.StatusOK {
		t.Fatalf("短标题过滤接口状态码错误，期望 200，实际 %d，响应：%s", trailResp.Code, trailResp.Body.String())
	}

	trailBody := map[string]any{}
	if err := json.Unmarshal(trailResp.Body.Bytes(), &trailBody); err != nil {
		t.Fatalf("解析短标题过滤响应失败: %v", err)
	}
	if code, ok := trailBody["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("短标题过滤响应 code 不正确: %v", trailBody["code"])
	}

	trailData, ok := trailBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("短标题过滤响应 data 格式不正确: %v", trailBody["data"])
	}
	if filtered, ok := trailData["filtered"].(bool); !ok || !filtered {
		t.Fatalf("期望短标题被过滤，实际 filtered 为：%v", trailData["filtered"])
	}
	if reason, ok := trailData["reason"].(string); !ok || reason != "title_too_short" {
		t.Fatalf("期望过滤原因为 title_too_short，实际为：%v", trailData["reason"])
	}
	if total, ok := trailData["total"].(float64); !ok || int(total) != 0 {
		t.Fatalf("期望短标题过滤后总数为 0，实际为：%v", trailData["total"])
	}
}

func TestTrailAddRejectsUnauthenticatedRequest(t *testing.T) {
	engine := newTestEngine(t)

	trailReq := httptest.NewRequest(http.MethodPost, "/api/trailAdd", strings.NewReader(`{"url":"https://example.com/page","title":"示例页面"}`))
	trailReq.Header.Set("Content-Type", "application/json")
	trailResp := httptest.NewRecorder()
	engine.ServeHTTP(trailResp, trailReq)

	if trailResp.Code != http.StatusOK {
		t.Fatalf("未登录添加浏览记录接口状态码错误，期望 200，实际 %d，响应：%s", trailResp.Code, trailResp.Body.String())
	}

	trailBody := map[string]any{}
	if err := json.Unmarshal(trailResp.Body.Bytes(), &trailBody); err != nil {
		t.Fatalf("解析未登录添加浏览记录响应失败: %v", err)
	}
	if code, ok := trailBody["code"].(float64); !ok || int(code) != 403 {
		t.Fatalf("未登录添加浏览记录响应 code 不正确，期望 403，实际：%v", trailBody["code"])
	}
}

func TestSummaryListReturnsCurrentUserRecentSummaries(t *testing.T) {
	env := newTestRouterEnv(t)
	accessToken := registerAndLoginForTest(t, env.engine, "Summary_List_User")

	if _, err := env.srv.Register(context.Background(), &request.UserRegister{
		Username: "summary_list_other",
		Password: "secret123",
	}); err != nil {
		t.Fatalf("注册第二个 summary 测试用户失败: %v", err)
	}

	writeRouteSummary(t, env.sm, 1, "20260525", filedb.Summary{
		Pars:    []filedb.SummaryPart{{Index: 1, Content: "第一天分片"}},
		Summary: "第一天最终总结",
	})
	writeRouteSummary(t, env.sm, 1, "20260524", filedb.Summary{
		Pars:    []filedb.SummaryPart{{Index: 1, Content: "第二天分片"}},
		Summary: "第二天最终总结",
	})
	writeRouteSummary(t, env.sm, 2, "20260526", filedb.Summary{
		Summary: "其他用户总结",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summaryList", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp := httptest.NewRecorder()
	env.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("每日总结列表接口状态码错误，期望 200，实际 %d，响应：%s", resp.Code, resp.Body.String())
	}

	body := map[string]any{}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析每日总结列表响应失败: %v", err)
	}
	if code, ok := body["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("每日总结列表响应 code 不正确: %v", body["code"])
	}

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("每日总结列表响应 data 格式不正确: %v", body["data"])
	}
	if total, ok := data["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("期望当前用户返回 2 条每日总结，实际为：%v", data["total"])
	}

	list, ok := data["list"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("每日总结列表响应 list 格式不正确: %v", data["list"])
	}

	firstItem, ok := list[0].(map[string]any)
	if !ok {
		t.Fatalf("每日总结列表首项格式不正确: %v", list[0])
	}
	if date, ok := firstItem["date"].(string); !ok || date != "20260525" {
		t.Fatalf("期望首项日期为 20260525，实际为：%v", firstItem["date"])
	}
	if summary, ok := firstItem["summary"].(string); !ok || summary != "第一天最终总结" {
		t.Fatalf("期望首项 summary 为 第一天最终总结，实际为：%v", firstItem["summary"])
	}
}

func TestSummaryDetailReturnsRequestedDate(t *testing.T) {
	env := newTestRouterEnv(t)
	accessToken := registerAndLoginForTest(t, env.engine, "Summary_Detail_User")

	writeRouteSummary(t, env.sm, 1, "20260525", filedb.Summary{
		Pars: []filedb.SummaryPart{
			{Index: 1, Titles: []string{"第一段页面标题"}, Content: "第一段"},
			{Index: 2, Titles: []string{"第二段页面标题"}, Content: "第二段"},
		},
		Summary: "详情最终总结",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/summaryDetail?date=2026-05-25", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp := httptest.NewRecorder()
	env.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("每日总结详情接口状态码错误，期望 200，实际 %d，响应：%s", resp.Code, resp.Body.String())
	}

	body := map[string]any{}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析每日总结详情响应失败: %v", err)
	}
	if code, ok := body["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("每日总结详情响应 code 不正确: %v", body["code"])
	}

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("每日总结详情响应 data 格式不正确: %v", body["data"])
	}
	if date, ok := data["date"].(string); !ok || date != "20260525" {
		t.Fatalf("期望详情日期为 20260525，实际为：%v", data["date"])
	}
	if summary, ok := data["summary"].(string); !ok || summary != "详情最终总结" {
		t.Fatalf("期望详情 summary 为 详情最终总结，实际为：%v", data["summary"])
	}
	if partCount, ok := data["partCount"].(float64); !ok || int(partCount) != 2 {
		t.Fatalf("期望详情 partCount 为 2，实际为：%v", data["partCount"])
	}
	parts, ok := data["parts"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("每日总结详情响应 parts 格式不正确: %v", data["parts"])
	}
	firstPart, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("每日总结详情响应首个 part 格式不正确: %v", parts[0])
	}
	titles, ok := firstPart["titles"].([]any)
	if !ok || len(titles) != 1 || titles[0] != "第一段页面标题" {
		t.Fatalf("期望详情首个分片返回 titles，实际为：%v", firstPart["titles"])
	}
}

func TestSummaryInterfacesRejectUnauthenticatedRequest(t *testing.T) {
	engine := newTestEngine(t)

	listReq := httptest.NewRequest(http.MethodGet, "/api/summaryList", nil)
	listResp := httptest.NewRecorder()
	engine.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("未登录每日总结列表接口状态码错误，期望 200，实际 %d，响应：%s", listResp.Code, listResp.Body.String())
	}

	listBody := map[string]any{}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("解析未登录每日总结列表响应失败: %v", err)
	}
	if code, ok := listBody["code"].(float64); !ok || int(code) != 403 {
		t.Fatalf("未登录每日总结列表响应 code 不正确，期望 403，实际：%v", listBody["code"])
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/summaryDetail?date=20260525", nil)
	detailResp := httptest.NewRecorder()
	engine.ServeHTTP(detailResp, detailReq)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("未登录每日总结详情接口状态码错误，期望 200，实际 %d，响应：%s", detailResp.Code, detailResp.Body.String())
	}

	detailBody := map[string]any{}
	if err := json.Unmarshal(detailResp.Body.Bytes(), &detailBody); err != nil {
		t.Fatalf("解析未登录每日总结详情响应失败: %v", err)
	}
	if code, ok := detailBody["code"].(float64); !ok || int(code) != 403 {
		t.Fatalf("未登录每日总结详情响应 code 不正确，期望 403，实际：%v", detailBody["code"])
	}
}

func TestDebugSummaryRejectsWhenDisabled(t *testing.T) {
	engine := newTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/summary?action=run_yesterday_summary&token=test-debug-token", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("summary 调试接口状态码错误，期望 200，实际 %d，响应：%s", resp.Code, resp.Body.String())
	}

	body := map[string]any{}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析 summary 调试响应失败: %v", err)
	}
	if code, ok := body["code"].(float64); !ok || int(code) != 1 {
		t.Fatalf("summary 调试接口默认关闭时应返回错误 code，实际：%v", body["code"])
	}
}

func TestDebugSummaryTriggersYesterdaySummary(t *testing.T) {
	cfg := &testConfig{
		values: map[string]string{
			"db.filedir":            filepath.Join(t.TempDir(), "filedb"),
			"name":                  "web-trail-ai-test",
			"secret":                "test-secret",
			"summary.debug-enabled": "true",
			"summary.debug-token":   "test-debug-token",
		},
	}
	log := &testLogger{}
	sm, err := scribble_manager.NewScribbleManager(cfg, log)
	if err != nil {
		t.Fatalf("初始化 scribble manager 失败: %v", err)
	}

	srv := service.NewService(cfg, log, sm)
	srv.SetLLMClient(&routeFakeLLMClient{})

	if _, err = srv.Register(context.Background(), &request.UserRegister{
		Username: "Summary_User",
		Password: "secret123",
	}); err != nil {
		t.Fatalf("注册 summary 测试用户失败: %v", err)
	}

	targetDate := time.Now().AddDate(0, 0, -1)
	targetDateText := targetDate.Format("20060102")
	if err = sm.DB().Write("trails/1/"+targetDateText, "trail", []filedb.Trail{
		{
			Title:     "调试接口测试页面标题",
			URL:       "https://example.com/debug-summary",
			InnerText: "这里是用于调试接口生成浏览总结的正文内容。",
			UserID:    1,
			CreatedAt: targetDate.Unix(),
		},
	}); err != nil {
		t.Fatalf("写入 summary 测试浏览记录失败: %v", err)
	}

	controller := ctrl.NewCtrl(srv, cfg, log)
	newRouter := NewRouter(controller, cfg, log)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	newRouter.Init(engine)

	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/debug/summary?action=run_yesterday_summary&token=test-debug-token", nil).WithContext(reqCtx)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("summary 调试接口状态码错误，期望 200，实际 %d，响应：%s", resp.Code, resp.Body.String())
	}

	body := map[string]any{}
	if err = json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析 summary 调试响应失败: %v", err)
	}
	if code, ok := body["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("summary 调试接口应执行成功，实际 code：%v，响应：%v", body["code"], body)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("summary 调试接口 data 格式不正确: %v", body["data"])
	}
	if date, ok := data["date"].(string); !ok || date != targetDateText {
		t.Fatalf("期望生成昨日 %s 的 summary，实际为：%v", targetDateText, data["date"])
	}
	if successCount, ok := data["success_count"].(float64); !ok || int(successCount) != 1 {
		t.Fatalf("期望成功生成 1 个用户 summary，实际为：%v", data["success_count"])
	}
}

type routeFakeLLMClient struct{}

func (f *routeFakeLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "路由调试总结", nil
}

func registerAndLoginForTest(t *testing.T, engine *gin.Engine, username string) string {
	t.Helper()

	registerReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"`+username+`","password":"secret123"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	engine.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusOK {
		t.Fatalf("注册测试用户接口状态码错误，期望 200，实际 %d，响应：%s", registerResp.Code, registerResp.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"`+username+`","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	engine.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("登录测试用户接口状态码错误，期望 200，实际 %d，响应：%s", loginResp.Code, loginResp.Body.String())
	}

	loginBody := map[string]any{}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("解析登录测试用户响应失败: %v", err)
	}
	accessToken, ok := loginBody["access_token"].(string)
	if !ok || accessToken == "" {
		t.Fatalf("登录测试用户响应中缺少 access_token: %v", loginBody)
	}

	return accessToken
}

func writeRouteSummary(t *testing.T, sm *scribble_manager.ScribbleManager, userID int, dateText string, summary filedb.Summary) {
	t.Helper()

	if err := sm.DB().Write("summary/"+strconv.Itoa(userID)+"/"+dateText, "summary", summary); err != nil {
		t.Fatalf("写入路由测试每日总结失败: %v", err)
	}
}
