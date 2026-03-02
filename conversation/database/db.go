package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/config"
)

// Config 数据库配置（简化版，主要配置在 config.Config 中）
type Config struct {
	DataDir      string // 数据目录完整路径（如果为空，从 config.Config 获取）
	DBName       string // 数据库文件名
	MaxOpenConns int    // 最大打开连接数
	MaxIdleConns int    // 最大空闲连接数
}

// NewConfigFromConfig 从全局配置创建数据库配置
func NewConfigFromConfig(cfg *config.Config) *Config {
	if cfg == nil || !cfg.Database.Enabled {
		return nil
	}

	// 如果 DataDir 为空，使用 workspace 下的数据库目录
	dataDir := cfg.Database.DataDir
	if dataDir == "" {
		dataDir = ".data"
	}

	return &Config{
		DataDir:      filepath.Join(cfg.GetWorkspacePath(), dataDir),
		DBName:       cfg.Database.DBName,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	}
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DataDir:      "./data",
		DBName:       "events.db",
		MaxOpenConns: 1, // SQLite 建议单连接
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
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 默认静默日志
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

	// 自动迁移表结构
	if err := c.db.AutoMigrate(&models.ConversationRecord{}); err != nil {
		return fmt.Errorf("创建 conversation_records 表失败: %w", err)
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_conv_records_event_type ON conversation_records(event_type);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_session_key ON conversation_records(session_key);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_timestamp ON conversation_records(timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_trace_id ON conversation_records(trace_id);",
		"CREATE INDEX IF NOT EXISTS idx_conv_records_role ON conversation_records(role);",
	}

	for _, indexSQL := range indexes {
		if err := c.db.Exec(indexSQL).Error; err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
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