package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"im-system/internal/model"
	"im-system/internal/repository"
	"im-system/pkg/jwt"
	"im-system/pkg/password"
)

type UserService struct {
	repo       *repository.UserRepository
	jwtService *jwt.JWTService
}

func NewUserService(repo *repository.UserRepository, jwtService *jwt.JWTService) *UserService {
	return &UserService{repo: repo, jwtService: jwtService}
}

// Register 注册
func (s *UserService) Register(username, email, plainPassword string) (*model.User, string, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" || plainPassword == "" {
		return nil, "", errors.New("username and password are required")
	}
	// 密码哈希
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, "", err
	}
	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Status:       "offline",
		LastSeen:     time.Now(),
	}
	if err := s.repo.Create(user); err != nil {
		return nil, "", err
	}
	// 默认签发 token
	token, err := s.jwtService.GenerateToken(
		// 使用用户ID作为 subject
		// 转成字符串
		fmt.Sprintf("%d", user.ID),
		map[string]interface{}{"username": user.Username},
	)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// Login 登录
func (s *UserService) Login(identifier, plainPassword string) (*model.User, string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || plainPassword == "" {
		return nil, "", errors.New("identifier and password are required")
	}
	u, err := s.repo.GetByUsernameOrEmail(identifier)
	if err != nil {
		return nil, "", err
	}
	if !password.Verify(plainPassword, u.PasswordHash) {
		return nil, "", errors.New("invalid credentials")
	}
	token, err := s.jwtService.GenerateToken(
		fmt.Sprintf("%d", u.ID),
		map[string]interface{}{"username": u.Username},
	)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}
