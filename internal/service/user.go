package service

import (
	"fmt"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// UserService 用户服务接口
type UserService interface {
	// CRUD
	CreateUser(req CreateUserRequest) (*models.User, error)
	GetUser(id uint) (*models.User, error)
	GetUserByCode(code string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	UpdateUser(id uint, req UpdateUserRequest) (*models.User, error)
	DeleteUser(id uint) error
	ListUsers(offset, limit int) ([]models.User, int64, error)

	// 密码管理
	ChangePassword(userID uint, req ChangePasswordRequest) error
	VerifyPassword(username, password string) (*models.User, error)

	// 初始化
	InitDefaultUser() (*models.User, error)
}

// userService 用户服务实现
type userService struct {
	userRepo    repository.UserRepository
	agentRepo   repository.AgentRepository
	codeService CodeService
}

// NewUserService 创建用户服务
func NewUserService(userRepo repository.UserRepository, agentRepo repository.AgentRepository, codeService CodeService) UserService {
	return &userService{
		userRepo:    userRepo,
		agentRepo:   agentRepo,
		codeService: codeService,
	}
}

// CreateUser 创建用户
func (s *userService) CreateUser(req CreateUserRequest) (*models.User, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, fmt.Errorf("username already exists")
	}

	// 检查邮箱是否已存在（如果提供了）
	if req.Email != "" {
		existingUser, err = s.userRepo.GetByEmail(req.Email)
		if err != nil {
			return nil, err
		}
		if existingUser != nil {
			return nil, fmt.Errorf("email already exists")
		}
	}

	// 哈希密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 生成唯一 UserCode
	userCode, err := GenerateUniqueCodeWithRetry(
		s.codeService.GenerateUserCode,
		s.userRepo.CheckUserCodeExists,
		3,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate user code: %w", err)
	}

	user := &models.User{
		UserCode:     userCode,
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		DisplayName:  req.DisplayName,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser 获取用户
func (s *userService) GetUser(id uint) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

// GetUserByCode 根据 Code 获取用户
func (s *userService) GetUserByCode(code string) (*models.User, error) {
	return s.userRepo.GetByUserCode(code)
}

// GetUserByUsername 根据用户名获取用户
func (s *userService) GetUserByUsername(username string) (*models.User, error) {
	return s.userRepo.GetByUsername(username)
}

// GetUserByEmail 根据邮箱获取用户
func (s *userService) GetUserByEmail(email string) (*models.User, error) {
	return s.userRepo.GetByEmail(email)
}

// UpdateUser 更新用户
func (s *userService) UpdateUser(id uint, req UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 更新字段
	if req.Email != "" {
		// 检查邮箱是否已被其他用户使用
		existingUser, err := s.userRepo.GetByEmail(req.Email)
		if err != nil {
			return nil, err
		}
		if existingUser != nil && existingUser.ID != id {
			return nil, fmt.Errorf("email already in use")
		}
		user.Email = req.Email
	}
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(id uint) error {
	return s.userRepo.Delete(id)
}

// ListUsers 获取用户列表
func (s *userService) ListUsers(offset, limit int) ([]models.User, int64, error) {
	return s.userRepo.List(offset, limit)
}

// ChangePassword 修改密码
func (s *userService) ChangePassword(userID uint, req ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return fmt.Errorf("invalid old password")
	}

	// 哈希新密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(passwordHash)
	return s.userRepo.Update(user)
}

// VerifyPassword 验证密码
func (s *userService) VerifyPassword(username, password string) (*models.User, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// InitDefaultUser 初始化默认用户
// 如果系统中没有用户，创建一个默认用户
func (s *userService) InitDefaultUser() (*models.User, error) {
	// 检查是否已有用户
	users, total, err := s.userRepo.List(0, 1)
	if err != nil {
		return nil, err
	}
	if total > 0 {
		return &users[0], nil // 返回第一个用户
	}

	// 创建默认用户
	return s.CreateUser(CreateUserRequest{
		Username:    "admin",
		Email:       "admin@nanobot.local",
		Password:    "admin123", // 生产环境应该要求用户修改
		DisplayName: "Administrator",
	})
}
