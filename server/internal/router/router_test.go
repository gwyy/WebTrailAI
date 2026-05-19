package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/ctrl"
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
	return false
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

func newTestEngine(t *testing.T) *gin.Engine {
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
	return engine
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
}
