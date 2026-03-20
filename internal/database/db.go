package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/models"
)

// Config 数据库配置（简化版，主要配置在 config.Config 中）
type Config struct {
	DataDir      string // 数据目录完整路径
	DBName       string // 数据库文件名
	MaxOpenConns int    // 最大打开连接数
	MaxIdleConns int    // 最大空闲连接数
}

// NewConfigFromConfig 从全局配置创建数据库配置
// 如果 cfg.Database.DataDir 为空，使用 DefaultConfig() 的固定路径 (程序目录/data)
func NewConfigFromConfig(cfg *config.Config) *Config {
	if cfg == nil || !cfg.Database.Enabled {
		return nil
	}

	// 如果 DataDir 为空，使用 DefaultConfig 的固定路径
	dataDir := cfg.Database.DataDir
	if dataDir == "" {
		return DefaultConfig()
	}

	// 如果配置中的 DataDir 是相对路径，基于 Workspace 创建完整路径
	if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(cfg.Agents.Defaults.Workspace, dataDir)
	}

	return &Config{
		DataDir:      dataDir,
		DBName:       cfg.Database.DBName,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	}
}

// DefaultConfig 返回默认配置
// 数据目录固定为当前工作目录下的 data 文件夹
func DefaultConfig() *Config {
	// 使用当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		// 如果获取失败，使用相对路径
		wd = "."
	}

	return &Config{
		DataDir:      filepath.Join(wd, "data"),
		DBName:       "nanobot.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}
}

// Client 数据库客户端
// 提供统一的数据库连接管理
type Client struct {
	db     *gorm.DB
	dbPath string
	config *Config
	mu     sync.RWMutex
}

// NewClient 创建数据库客户端
// 如果 config 为 nil，使用默认配置
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 确保数据目录存在
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 数据库文件路径
	dbPath := filepath.Join(config.DataDir, config.DBName)

	// 打开数据库连接
	// 默认显示详细日志，NANOBOT_DB_LOG=0 可关闭
	logLevel := logger.Info
	if os.Getenv("NANOBOT_DB_LOG") == "0" {
		logLevel = logger.Silent
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)

	return &Client{
		db:     db,
		dbPath: dbPath,
		config: config,
	}, nil
}

// DB 获取 GORM 数据库连接
// 注意：返回的是 *gorm.DB 的副本，每个副本有独立的状态
func (c *Client) DB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// DBPath 获取数据库文件路径
func (c *Client) DBPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dbPath
}

// InitSchema 初始化数据库表结构和索引
func (c *Client) InitSchema() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 自动迁移用户管理相关表
	if err := c.db.AutoMigrate(
		&models.User{},
		&models.Agent{},
		&models.Channel{},
		&models.Session{},
		&models.LLMProvider{},
		&models.CronJob{},
	); err != nil {
		return fmt.Errorf("创建用户管理表失败: %w", err)
	}

	// 自动迁移对话记录表
	if err := c.db.AutoMigrate(&models.ConversationRecord{}); err != nil {
		return fmt.Errorf("创建 conversation_records 表失败: %w", err)
	}


	// 自动迁移 MCP 相关表
	if err := c.db.AutoMigrate(&models.MCPServer{}, &models.AgentMCPBinding{}, &models.MCPToolModel{}, &models.MCPToolLog{}); err != nil {
		return fmt.Errorf("创建 MCP 表失败: %w", err)
	}

	// 迁移：将 agent_mcp_bindings 表的 is_enabled 列重命名为 is_active
	if err := c.migrateAgentMCPBindingColumn(); err != nil {
		return fmt.Errorf("迁移 agent_mcp_bindings 表列失败: %w", err)
	}

	// 创建索引
	indexes := []string{
		// 对话记录表索引
		"CREATE INDEX IF NOT EXISTS idx_conv_records_event_type ON conversation_records(event_type);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_session_key ON conversation_records(session_key);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_timestamp ON conversation_records(timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_trace_id ON conversation_records(trace_id);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_role ON conversation_records(role);",
		// 新增：归属信息索引（使用 Code 字段）
		"CREATE INDEX IF NOT EXISTS idx_conv_records_user_code ON conversation_records(user_code);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_agent_code ON conversation_records(agent_code);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_channel_code ON conversation_records(channel_code);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_channel_type ON conversation_records(channel_type);",
		// 复合索引：优化常见查询场景
		"CREATE INDEX IF NOT EXISTS idx_conv_records_user_code_timestamp ON conversation_records(user_code, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_agent_code_timestamp ON conversation_records(agent_code, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_channel_code_timestamp ON conversation_records(channel_code, timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_session_key_timestamp ON conversation_records(session_key, timestamp);",
		// MCP 服务器表索引
		"CREATE INDEX IF NOT EXISTS idx_mcp_servers_code ON mcp_servers(code);",
		"CREATE INDEX IF NOT EXISTS idx_mcp_servers_status ON mcp_servers(status);",
		// Agent MCP 绑定表索引
		"CREATE INDEX IF NOT EXISTS idx_agent_mcp_bindings_agent_id ON agent_mcp_bindings(agent_id);",
		"CREATE INDEX IF NOT EXISTS idx_agent_mcp_bindings_mcp_server_id ON agent_mcp_bindings(mcp_server_id);",
	}

	for _, indexSQL := range indexes {
		if err := c.db.Exec(indexSQL).Error; err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	return nil
}

// migrateAgentMCPBindingColumn 迁移 agent_mcp_bindings 表的列
// 将 is_enabled 重命名为 is_active
func (c *Client) migrateAgentMCPBindingColumn() error {
	// 使用 GORM Migrator 检查列是否存在
	migrator := c.db.Migrator()

	// 检查是否存在旧的 is_enabled 列
	hasOldColumn := migrator.HasColumn(&models.AgentMCPBinding{}, "IsEnabled")
	hasNewColumn := migrator.HasColumn(&models.AgentMCPBinding{}, "IsActive")

	if hasOldColumn && !hasNewColumn {
		// SQLite 支持 RENAME COLUMN
		if err := c.db.Exec("ALTER TABLE agent_mcp_bindings RENAME COLUMN is_enabled TO is_active").Error; err != nil {
			return fmt.Errorf("重命名列失败: %w", err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err != nil {
			return fmt.Errorf("获取数据库连接失败: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}
