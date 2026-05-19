package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
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

func newTestService(t *testing.T) *Service {
	t.Helper()

	cfg := &testConfig{
		values: map[string]string{
			"db.filedir": filepath.Join(t.TempDir(), "filedb"),
		},
	}
	log := &testLogger{}
	sm, err := scribble_manager.NewScribbleManager(cfg, log)
	if err != nil {
		t.Fatalf("初始化 scribble manager 失败: %v", err)
	}

	return NewService(cfg, log, sm)
}

func TestRegisterSuccess(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.Register(context.Background(), &request.UserRegister{
		Username: " Alice ",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	if user.ID != 1 {
		t.Fatalf("期望用户 ID 为 1，实际为 %d", user.ID)
	}
	if user.Username != "alice" {
		t.Fatalf("期望用户名被规范化为 alice，实际为 %s", user.Username)
	}
	if user.Status != userStatusActive {
		t.Fatalf("期望用户状态为 %d，实际为 %d", userStatusActive, user.Status)
	}
	if user.Password != "secret123" {
		t.Fatalf("期望密码按明文保存，实际为 %s", user.Password)
	}

	storedUser, err := svc.loadUserByUsername("alice")
	if err != nil {
		t.Fatalf("读取已保存用户失败: %v", err)
	}
	if storedUser.Password != "secret123" {
		t.Fatalf("期望保存的密码为明文 secret123，实际为 %s", storedUser.Password)
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), &request.UserRegister{
		Username: "alice",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}

	_, err = svc.Register(context.Background(), &request.UserRegister{
		Username: " ALICE ",
		Password: "another123",
	})
	if !errors.Is(err, ErrUsernameAlreadyExists) {
		t.Fatalf("期望重复注册返回 ErrUsernameAlreadyExists，实际为 %v", err)
	}
}

func TestRegisterRejectInvalidUsername(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), &request.UserRegister{
		Username: "../alice",
		Password: "secret123",
	})
	if !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("期望非法用户名返回 ErrInvalidUsername，实际为 %v", err)
	}
}

func TestAuthenticateSuccess(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), &request.UserRegister{
		Username: "alice",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	user, err := svc.Authenticate(context.Background(), &request.UserLogin{
		Username: " ALICE ",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("登录校验失败: %v", err)
	}
	if user.Username != "alice" {
		t.Fatalf("期望登录用户为 alice，实际为 %s", user.Username)
	}
}

func TestAuthenticateWrongPassword(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), &request.UserRegister{
		Username: "alice",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	_, err = svc.Authenticate(context.Background(), &request.UserLogin{
		Username: "alice",
		Password: "wrong123",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("期望错误密码返回 ErrInvalidCredentials，实际为 %v", err)
	}
}

func TestRegisterConcurrentUsers(t *testing.T) {
	svc := newTestService(t)

	const userCount = 8
	userIDs := make(chan int, userCount)
	errCh := make(chan error, userCount)
	var wg sync.WaitGroup

	for i := 0; i < userCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			user, err := svc.Register(context.Background(), &request.UserRegister{
				Username: fmt.Sprintf("user_%d", index),
				Password: "secret123",
			})
			if err != nil {
				errCh <- err
				return
			}
			userIDs <- user.ID
		}(i)
	}

	wg.Wait()
	close(errCh)
	close(userIDs)

	for err := range errCh {
		if err != nil {
			t.Fatalf("并发注册失败: %v", err)
		}
	}

	seenIDs := make(map[int]struct{}, userCount)
	for userID := range userIDs {
		if _, exists := seenIDs[userID]; exists {
			t.Fatalf("发现重复用户 ID: %d", userID)
		}
		seenIDs[userID] = struct{}{}
	}
	if len(seenIDs) != userCount {
		t.Fatalf("期望注册 %d 个用户，实际为 %d 个", userCount, len(seenIDs))
	}
}
