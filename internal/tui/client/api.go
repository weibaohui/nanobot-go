package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/weibaohui/nanobot-go/internal/models"
)

// Client TUI API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建 API 客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetBaseURL 设置基础 URL
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// doJSON 执行请求并解析 JSON 响应
func (c *Client) doJSON(method, path string, body, result interface{}) error {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck() error {
	resp, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}
	return nil
}

// ========== User API ==========

// ListUsers 获取用户列表
func (c *Client) ListUsers() ([]models.User, error) {
	var resp struct {
		Data []models.User `json:"data"`
	}
	err := c.doJSON("GET", "/api/v1/users", nil, &resp)
	return resp.Data, err
}

// GetUser 获取用户详情
func (c *Client) GetUser(id uint) (*models.User, error) {
	var user models.User
	err := c.doJSON("GET", fmt.Sprintf("/api/v1/users/%d", id), nil, &user)
	return &user, err
}

// CreateUser 创建用户
func (c *Client) CreateUser(req map[string]interface{}) (*models.User, error) {
	var user models.User
	err := c.doJSON("POST", "/api/v1/users", req, &user)
	return &user, err
}

// UpdateUser 更新用户
func (c *Client) UpdateUser(id uint, req map[string]interface{}) (*models.User, error) {
	var user models.User
	err := c.doJSON("PUT", fmt.Sprintf("/api/v1/users/%d", id), req, &user)
	return &user, err
}

// DeleteUser 删除用户
func (c *Client) DeleteUser(id uint) error {
	return c.doJSON("DELETE", fmt.Sprintf("/api/v1/users/%d", id), nil, nil)
}

// ========== Agent API ==========

// ListAgents 获取 Agent 列表
func (c *Client) ListAgents(userID uint) ([]models.Agent, error) {
	var resp struct {
		Data []models.Agent `json:"data"`
	}
	path := fmt.Sprintf("/api/v1/agents?user_id=%d", userID)
	err := c.doJSON("GET", path, nil, &resp)
	return resp.Data, err
}

// GetAgent 获取 Agent 详情
func (c *Client) GetAgent(id uint) (*models.Agent, error) {
	var agent models.Agent
	err := c.doJSON("GET", fmt.Sprintf("/api/v1/agents/%d", id), nil, &agent)
	return &agent, err
}

// CreateAgent 创建 Agent
func (c *Client) CreateUserAgent(userID uint, req map[string]interface{}) (*models.Agent, error) {
	req["user_id"] = userID
	var agent models.Agent
	err := c.doJSON("POST", "/api/v1/agents", req, &agent)
	return &agent, err
}

// UpdateAgent 更新 Agent
func (c *Client) UpdateAgent(id uint, req map[string]interface{}) (*models.Agent, error) {
	var agent models.Agent
	err := c.doJSON("PUT", fmt.Sprintf("/api/v1/agents/%d", id), req, &agent)
	return &agent, err
}

// DeleteAgent 删除 Agent
func (c *Client) DeleteAgent(id uint) error {
	return c.doJSON("DELETE", fmt.Sprintf("/api/v1/agents/%d", id), nil, nil)
}

// GetAgentConfig 获取 Agent 配置
func (c *Client) GetAgentConfig(id uint) (map[string]interface{}, error) {
	var config map[string]interface{}
	err := c.doJSON("GET", fmt.Sprintf("/api/v1/agents/%d/config", id), nil, &config)
	return config, err
}

// UpdateAgentConfig 更新 Agent 配置
func (c *Client) UpdateAgentConfig(id uint, config map[string]interface{}) error {
	return c.doJSON("PUT", fmt.Sprintf("/api/v1/agents/%d/config", id), config, nil)
}

// ========== Channel API ==========

// ListChannels 获取 Channel 列表
func (c *Client) ListChannels(userID uint) ([]models.Channel, error) {
	var resp struct {
		Data []models.Channel `json:"data"`
	}
	path := fmt.Sprintf("/api/v1/channels?user_id=%d", userID)
	err := c.doJSON("GET", path, nil, &resp)
	return resp.Data, err
}

// GetChannel 获取 Channel 详情
func (c *Client) GetChannel(id uint) (*models.Channel, error) {
	var channel models.Channel
	err := c.doJSON("GET", fmt.Sprintf("/api/v1/channels/%d", id), nil, &channel)
	return &channel, err
}

// CreateChannel 创建 Channel
func (c *Client) CreateChannel(req map[string]interface{}) (*models.Channel, error) {
	var channel models.Channel
	err := c.doJSON("POST", "/api/v1/channels", req, &channel)
	return &channel, err
}

// UpdateChannel 更新 Channel
func (c *Client) UpdateChannel(id uint, req map[string]interface{}) (*models.Channel, error) {
	var channel models.Channel
	err := c.doJSON("PUT", fmt.Sprintf("/api/v1/channels/%d", id), req, &channel)
	return &channel, err
}

// DeleteChannel 删除 Channel
func (c *Client) DeleteChannel(id uint) error {
	return c.doJSON("DELETE", fmt.Sprintf("/api/v1/channels/%d", id), nil, nil)
}

// BindAgent 绑定 Agent
func (c *Client) BindAgent(channelID, agentID uint) error {
	req := map[string]interface{}{"agent_id": agentID}
	return c.doJSON("POST", fmt.Sprintf("/api/v1/channels/%d/bind", channelID), req, nil)
}

// UnbindAgent 解绑 Agent
func (c *Client) UnbindAgent(channelID uint) error {
	return c.doJSON("POST", fmt.Sprintf("/api/v1/channels/%d/unbind", channelID), nil, nil)
}

// ========== Session API ==========

// ListSessions 获取 Session 列表
func (c *Client) ListSessions(userID uint) ([]models.Session, error) {
	var resp struct {
		Data []models.Session `json:"data"`
	}
	path := fmt.Sprintf("/api/v1/sessions?user_id=%d", userID)
	err := c.doJSON("GET", path, nil, &resp)
	return resp.Data, err
}

// GetSession 获取 Session 详情
func (c *Client) GetSession(key string) (*models.Session, error) {
	var session models.Session
	err := c.doJSON("GET", fmt.Sprintf("/api/v1/sessions/%s", key), nil, &session)
	return &session, err
}

// DeleteSession 删除 Session
func (c *Client) DeleteSession(key string) error {
	return c.doJSON("DELETE", fmt.Sprintf("/api/v1/sessions/%s", key), nil, nil)
}

// TouchSession 更新 Session 活跃时间
func (c *Client) TouchSession(key string) error {
	return c.doJSON("POST", fmt.Sprintf("/api/v1/sessions/%s/touch", key), nil, nil)
}
