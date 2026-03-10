package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/repository"
)

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	SessionKey string
	ExternalID string
	AgentID    *uint
	ChannelID  uint
	Metadata   map[string]interface{}
}

// SessionService 会话服务接口
type SessionService interface {
	// CRUD
	CreateSession(userID uint, req CreateSessionRequest) (*models.Session, error)
	GetSession(id uint) (*models.Session, error)
	GetSessionByKey(key string) (*models.Session, error)
	GetChannelSessions(channelID uint) ([]models.Session, error)
	GetUserSessions(userID uint) ([]models.Session, error)
	UpdateSession(session *models.Session) error
	DeleteSession(sessionKey string) error
	DeleteChannelSessions(channelID uint) error

	// 活跃管理
	TouchSession(sessionKey string) error
	GetLastActive(sessionKey string) (*time.Time, error)

	// 元数据管理
	GetSessionMetadata(sessionKey string) (map[string]interface{}, error)
	UpdateSessionMetadata(sessionKey string, metadata map[string]interface{}) error
}

// sessionService 会话服务实现
type sessionService struct {
	sessionRepo repository.SessionRepository
}

// NewSessionService 创建会话服务
func NewSessionService(sessionRepo repository.SessionRepository) SessionService {
	return &sessionService{
		sessionRepo: sessionRepo,
	}
}

// CreateSession 创建会话
func (s *sessionService) CreateSession(userID uint, req CreateSessionRequest) (*models.Session, error) {
	// 序列化元数据
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("序列化元数据失败: %w", err)
	}

	now := time.Now()
	session := &models.Session{
		UserID:       userID,
		AgentID:      req.AgentID,
		ChannelID:    req.ChannelID,
		SessionKey:   req.SessionKey,
		ExternalID:   req.ExternalID,
		LastActiveAt: &now,
		Metadata:     string(metadataJSON),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession 获取会话
func (s *sessionService) GetSession(id uint) (*models.Session, error) {
	return s.sessionRepo.GetByID(id)
}

// GetSessionByKey 根据 SessionKey 获取会话
func (s *sessionService) GetSessionByKey(key string) (*models.Session, error) {
	return s.sessionRepo.GetBySessionKey(key)
}

// GetChannelSessions 获取 Channel 的所有会话
func (s *sessionService) GetChannelSessions(channelID uint) ([]models.Session, error) {
	return s.sessionRepo.GetByChannelID(channelID)
}

// GetUserSessions 获取用户的所有会话
func (s *sessionService) GetUserSessions(userID uint) ([]models.Session, error) {
	return s.sessionRepo.GetActiveByUserID(userID)
}

// UpdateSession 更新会话
func (s *sessionService) UpdateSession(session *models.Session) error {
	return s.sessionRepo.Update(session)
}

// DeleteSession 删除会话
func (s *sessionService) DeleteSession(sessionKey string) error {
	return s.sessionRepo.Delete(sessionKey)
}

// DeleteChannelSessions 删除 Channel 的所有会话
func (s *sessionService) DeleteChannelSessions(channelID uint) error {
	return s.sessionRepo.DeleteByChannelID(channelID)
}

// TouchSession 更新会话最后活跃时间
func (s *sessionService) TouchSession(sessionKey string) error {
	return s.sessionRepo.UpdateLastActive(sessionKey)
}

// GetLastActive 获取会话最后活跃时间
func (s *sessionService) GetLastActive(sessionKey string) (*time.Time, error) {
	session, err := s.sessionRepo.GetBySessionKey(sessionKey)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	return session.LastActiveAt, nil
}

// GetSessionMetadata 获取会话元数据
func (s *sessionService) GetSessionMetadata(sessionKey string) (map[string]interface{}, error) {
	session, err := s.sessionRepo.GetBySessionKey(sessionKey)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	if session.Metadata == "" || session.Metadata == "null" {
		return map[string]interface{}{}, nil
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(session.Metadata), &metadata); err != nil {
		return nil, fmt.Errorf("解析元数据失败: %w", err)
	}

	return metadata, nil
}

// UpdateSessionMetadata 更新会话元数据
func (s *sessionService) UpdateSessionMetadata(sessionKey string, metadata map[string]interface{}) error {
	session, err := s.sessionRepo.GetBySessionKey(sessionKey)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	session.Metadata = string(metadataJSON)
	return s.sessionRepo.Update(session)
}
