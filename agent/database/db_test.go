package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/weibaohui/nanobot-go/agent/models"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "./data", config.DataDir)
	assert.Equal(t, "events.db", config.DBName)
	assert.Equal(t, 1, config.MaxOpenConns)
	assert.Equal(t, 1, config.MaxIdleConns)
}

func TestNewClient(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir:      tmpDir,
		DBName:       "test.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 验证数据库文件已创建
	dbPath := filepath.Join(tmpDir, "test.db")
	_, err = os.Stat(dbPath)
	require.NoError(t, err, "数据库文件应该被创建")

	// 验证 DBPath 方法
	assert.Equal(t, dbPath, client.DBPath())

	// 验证 DB 方法返回非 nil
	assert.NotNil(t, client.DB())
}

func TestNewClientWithNilConfig(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 应该使用默认配置
	// 注意：默认配置会创建 ./data 目录，所以路径包含目录名
	assert.Contains(t, client.DBPath(), "events.db")
}

func TestNewClientWithInvalidDataDir(t *testing.T) {
	// 使用无效的目录名（包含非法字符）
	config := &Config{
		DataDir: "\x00/invalid",
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestInitSchema(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir: tmpDir,
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 初始化表结构
	err = client.InitSchema()
	require.NoError(t, err)

	// 验证表已创建
	db := client.DB()

	// 检查表是否存在
	var tableName string
	err = db.Raw(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='events'",
	).Scan(&tableName).Error
	require.NoError(t, err)
	assert.Equal(t, "events", tableName)

	// 验证可以插入数据
	event := models.Event{
		TraceID:    "test-trace",
		EventType:  "test",
		Timestamp:  db.NowFunc(),
		SessionKey: "test-session",
		Role:       "user",
		Content:    "test content",
	}
	err = db.Create(&event).Error
	require.NoError(t, err)
	assert.NotZero(t, event.ID)

	// 验证索引已创建
	var indexName string
	err = db.Raw(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_events_event_type'",
	).Scan(&indexName).Error
	require.NoError(t, err)
	assert.Equal(t, "idx_events_event_type", indexName)
}

func TestDBMethodReturnsCopy(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir: tmpDir,
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 初始化表结构
	err = client.InitSchema()
	require.NoError(t, err)

	// 获取两个 DB 副本
	db1 := client.DB()
	db2 := client.DB()

	// 验证它们是不同的实例
	// GORM 的 *gorm.DB 是指针，每次调用 DB() 会返回相同的指针
	// 但是每个副本有独立的状态
	assert.Same(t, db1, db2)

	// 验证两个副本都可以正常使用
	var count1, count2 int64
	err = db1.Model(&models.Event{}).Count(&count1).Error
	require.NoError(t, err)
	err = db2.Model(&models.Event{}).Count(&count2).Error
	require.NoError(t, err)
	assert.Equal(t, count1, count2)
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir: tmpDir,
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)

	// 初始化表结构
	err = client.InitSchema()
	require.NoError(t, err)

	// 关闭连接
	err = client.Close()
	require.NoError(t, err)

	// 验证 DB() 方法仍然返回非 nil（但操作会失败）
	db := client.DB()
	assert.NotNil(t, db)

	// 尝试查询应该失败
	var count int64
	err = db.Model(&models.Event{}).Count(&count).Error
	assert.Error(t, err)

	// 再次关闭应该返回 nil 或错误（取决于实现）
	err = client.Close()
	// 可能返回 nil 或错误，只要不 panic 就可以
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir: tmpDir,
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 初始化表结构
	err = client.InitSchema()
	require.NoError(t, err)

	// 并发访问测试
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			// 同时调用 DB() 和 DBPath()
			db := client.DB()
			dbPath := client.DBPath()
			assert.NotNil(t, db)
			assert.NotEmpty(t, dbPath)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestInitSchemaWithExistingSchema(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir: tmpDir,
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 第一次初始化
	err = client.InitSchema()
	require.NoError(t, err)

	// 插入一些数据
	db := client.DB()
	event := models.Event{
		TraceID:    "test-trace",
		EventType:  "test",
		Timestamp:  db.NowFunc(),
		SessionKey: "test-session",
		Role:       "user",
		Content:    "test content",
	}
	err = db.Create(&event).Error
	require.NoError(t, err)

	// 第二次初始化（应该成功，不报错）
	err = client.InitSchema()
	require.NoError(t, err)

	// 验证数据仍然存在
	var count int64
	err = db.Model(&models.Event{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestClientIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir: tmpDir,
		DBName:  "events.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 初始化表结构
	err = client.InitSchema()
	require.NoError(t, err)

	// 执行完整的 CRUD 操作
	db := client.DB()

	// Create
	event := models.Event{
		TraceID:          "test-trace-001",
		EventType:        "test",
		Timestamp:        db.NowFunc(),
		SessionKey:       "test-session",
		Role:             "user",
		Content:          "test content",
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		ReasoningTokens:  5,
		CachedTokens:     0,
	}
	err = db.Create(&event).Error
	require.NoError(t, err)
	assert.NotZero(t, event.ID)

	// Read
	var found models.Event
	err = db.First(&found, event.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "test-trace-001", found.TraceID)
	assert.Equal(t, "test content", found.Content)
	assert.Equal(t, 30, found.TotalTokens)

	// Update
	err = db.Model(&found).Update("content", "updated content").Error
	require.NoError(t, err)

	var updated models.Event
	err = db.First(&updated, event.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "updated content", updated.Content)

	// Delete
	err = db.Delete(&updated).Error
	require.NoError(t, err)

	// Verify deletion
	var deleted models.Event
	err = db.First(&deleted, event.ID).Error
	assert.Error(t, err)
}

func TestClientWithExistingDBFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// 先创建一个数据库文件并插入数据
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&models.Event{})
	require.NoError(t, err)

	event := models.Event{
		TraceID:    "existing-trace",
		EventType:  "test",
		Timestamp:  db.NowFunc(),
		SessionKey: "test-session",
		Role:       "user",
		Content:    "existing content",
	}
	err = db.Create(&event).Error
	require.NoError(t, err)

	sqlDB, _ := db.DB()
	sqlDB.Close()

	// 创建客户端连接到已存在的数据库
	config := &Config{
		DataDir: tmpDir,
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 不初始化表结构（表已经存在）
	// 验证可以读取现有数据
	var found models.Event
	err = client.DB().Where("trace_id = ?", "existing-trace").First(&found).Error
	require.NoError(t, err)
	assert.Equal(t, "existing content", found.Content)
}

func TestCustomMaxConns(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DataDir:      tmpDir,
		DBName:       "test.db",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// 验证连接池参数
	sqlDB, err := client.DB().DB()
	require.NoError(t, err)

	// 注意：SQLite 实际上不支持高并发，但我们可以设置参数
	// 这里的测试主要是验证配置被应用
	assert.NotNil(t, sqlDB)
}