package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gwyy/WebTrailAI/server/internal/model/filedb"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
)

const (
	userCollection   = "users"
	userStatusActive = 1
	minPasswordBytes = 6
)

var (
	usernamePattern          = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)
	ErrUsernameAlreadyExists = errors.New("用户名已存在")
	ErrInvalidUsername       = errors.New("用户名只能包含小写字母、数字和下划线，长度为3到32位")
	ErrInvalidPassword       = errors.New("密码长度不能少于6位")
	ErrInvalidCredentials    = errors.New("用户名或密码错误")
)

// Register 负责注册用户：规范化用户名、校验密码、检查重名、生成递增 ID，并将密码原样写入本地 scribble 用户库。
func (s *Service) Register(ctx context.Context, req *request.UserRegister) (*filedb.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New("注册参数不能为空")
	}

	username, err := normalizeUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if err = validatePassword(req.Password); err != nil {
		return nil, err
	}

	s.userMu.Lock()
	defer s.userMu.Unlock()

	existingUser, err := s.loadUserByUsername(username)
	if err == nil && existingUser != nil {
		return nil, ErrUsernameAlreadyExists
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取用户信息失败: %w", err)
	}

	nextID, err := s.nextUserID()
	if err != nil {
		return nil, err
	}

	user := &filedb.User{
		ID:       nextID,
		Username: username,
		Password: req.Password,
		Status:   userStatusActive,
	}
	if err = s.sm.DB().Write(userCollection, username, user); err != nil {
		return nil, fmt.Errorf("保存用户信息失败: %w", err)
	}

	return user, nil
}

// Authenticate 负责登录校验：按用户名读取本地用户文件，校验状态和明文密码，供 JWT 登录流程复用。
func (s *Service) Authenticate(ctx context.Context, req *request.UserLogin) (*filedb.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, ErrInvalidCredentials
	}

	username, err := normalizeUsername(req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if err = validatePassword(req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.loadUserByUsername(username)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("读取用户信息失败: %w", err)
	}
	if user.Status != userStatusActive {
		return nil, ErrInvalidCredentials
	}
	if user.Password != req.Password {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// normalizeUsername 会去掉首尾空白并统一转为小写，同时限制字符集，避免非法账号名和文件路径风险。
func normalizeUsername(username string) (string, error) {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	if !usernamePattern.MatchString(normalizedUsername) {
		return "", ErrInvalidUsername
	}
	return normalizedUsername, nil
}

// validatePassword 校验密码长度是否合法，当前仅限制最小长度，避免空密码和过短密码。
func validatePassword(password string) error {
	passwordLen := len([]byte(password))
	if passwordLen < minPasswordBytes {
		return ErrInvalidPassword
	}
	return nil
}

// loadUserByUsername 按用户名读取对应的用户 JSON 文件，文件名与规范化后的用户名一一对应。
func (s *Service) loadUserByUsername(username string) (*filedb.User, error) {
	user := &filedb.User{}
	if err := s.sm.DB().Read(userCollection, username, user); err != nil {
		return nil, err
	}
	return user, nil
}

// nextUserID 扫描 users 集合中的所有用户文件，取当前最大 ID 后加一，保证新注册用户拿到递增 ID。
func (s *Service) nextUserID() (int, error) {
	records, err := s.sm.DB().ReadAll(userCollection)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("读取用户列表失败: %w", err)
	}

	maxUserID := 0
	for _, record := range records {
		user := filedb.User{}
		if err = json.Unmarshal(record, &user); err != nil {
			return 0, fmt.Errorf("解析用户数据失败: %w", err)
		}
		if user.ID > maxUserID {
			maxUserID = user.ID
		}
	}

	return maxUserID + 1, nil
}
