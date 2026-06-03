package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigReadsDeploymentEnvAliases(t *testing.T) {
	t.Setenv("WEBTRAIL_SECRET", "env-secret")
	t.Setenv("WEBTRAIL_AI_DASHSCOPE_API_KEY", "env-dashscope-key")

	cfg := NewConfig(writeTestConfig(t))

	if got := cfg.GetString("secret"); got != "env-secret" {
		t.Fatalf("期望从 WEBTRAIL_SECRET 读取 secret，实际为 %q", got)
	}
	if got := cfg.GetString("ai.dashscope_api_key"); got != "env-dashscope-key" {
		t.Fatalf("期望从 WEBTRAIL_AI_DASHSCOPE_API_KEY 读取大模型 key，实际为 %q", got)
	}
}

func TestNewConfigRejectsEmptyProductionSecret(t *testing.T) {
	path := writeTestConfigWithContent(t, `
name: web-trail-ai
mode: "prod"
version: "0.0.1"
secret: ""
`)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("期望生产配置缺少 secret 时触发 panic")
		}
	}()

	NewConfig(path)
}

func writeTestConfig(t *testing.T) string {
	t.Helper()

	return writeTestConfigWithContent(t, `
name: web-trail-ai
mode: "test"
version: "0.0.1"
secret: ""

ai:
  dashscope_api_key: ""
`)
}

func writeTestConfigWithContent(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	return path
}
