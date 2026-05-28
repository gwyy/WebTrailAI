package utils

import (
	"context"
	"errors"
	"testing"
)

func TestNewDashScopeClientRejectsEmptyAPIKey(t *testing.T) {
	_, err := NewDashScopeClient(LLMOptions{
		APIKey: " ",
	})
	if !errors.Is(err, ErrLLMAPIKeyEmpty) {
		t.Fatalf("期望空 API Key 返回 ErrLLMAPIKeyEmpty，实际为 %v", err)
	}
}

func TestDashScopeClientRejectsEmptyPromptBeforeRequest(t *testing.T) {
	client, err := NewDashScopeClient(LLMOptions{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("初始化测试大模型客户端失败: %v", err)
	}

	_, err = client.Generate(context.Background(), "   ")
	if !errors.Is(err, ErrLLMPromptEmpty) {
		t.Fatalf("期望空提示词返回 ErrLLMPromptEmpty，实际为 %v", err)
	}
}
