package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	defaultDashScopeBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	defaultDashScopeModel   = "qwen-plus"
	defaultLLMTimeout       = 60 * time.Second
)

var (
	ErrLLMAPIKeyEmpty = errors.New("大模型 API Key 不能为空")
	ErrLLMPromptEmpty = errors.New("大模型提示词不能为空")
	ErrLLMNoContent   = errors.New("大模型返回内容为空")
)

type LLMOptions struct {
	APIKey    string
	BaseURL   string
	Model     string
	Timeout   time.Duration
	MaxTokens int64
}

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type DashScopeClient struct {
	client    openai.Client
	model     string
	timeout   time.Duration
	maxTokens int64
}

func NewDashScopeClient(opts LLMOptions) (*DashScopeClient, error) {
	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" {
		return nil, ErrLLMAPIKeyEmpty
	}

	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		baseURL = defaultDashScopeBaseURL
	}

	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = defaultDashScopeModel
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultLLMTimeout
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
		option.WithRequestTimeout(timeout),
	)

	return &DashScopeClient{
		client:    client,
		model:     model,
		timeout:   timeout,
		maxTokens: opts.MaxTokens,
	}, nil
}

func (c *DashScopeClient) Generate(ctx context.Context, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", ErrLLMPromptEmpty
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model:       openai.ChatModel(c.model),
		Temperature: openai.Float(0.2),
	}
	if c.maxTokens > 0 {
		params.MaxTokens = openai.Int(c.maxTokens)
	}

	chatCompletion, err := c.client.Chat.Completions.New(reqCtx, params)
	if err != nil {
		return "", fmt.Errorf("调用大模型失败: %w", err)
	}
	if len(chatCompletion.Choices) == 0 {
		return "", ErrLLMNoContent
	}

	content := strings.TrimSpace(chatCompletion.Choices[0].Message.Content)
	if content == "" {
		return "", ErrLLMNoContent
	}

	return content, nil
}
