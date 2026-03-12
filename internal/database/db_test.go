package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/internal/models"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	// 新行为：使用程序所在目录下的 data 文件夹
	assert.Contains(t, config.DataDir, "data")
	assert.Equal(t, "nanobot.db", config.DBName)
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

	dbPath := filepath.Join(tmpDir, "test.db")
	_, err = os.Stat(dbPath)
	require.NoError(t, err, "数据库文件应该被创建")

	assert.Equal(t, dbPath, client.DBPath())
	assert.NotNil(t, client.DB())
}

func TestNewClientWithNilConfig(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	assert.Contains(t, client.DBPath(), "nanobot.db")
}

func TestNewClientWithInvalidDataDir(t *testing.T) {
	config := &Config{
		DataDir: "\x00/invalid",
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewConfigFromConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
			},
		},
		Database: config.DatabaseConfig{
			Enabled:      true,
			DataDir:      ".data",
			DBName:       "events.db",
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
	}

	dbConfig := NewConfigFromConfig(cfg)
	require.NotNil(t, dbConfig)

	expectedPath := filepath.Join(tmpDir, ".data")
	assert.Equal(t, expectedPath, dbConfig.DataDir)
	assert.Equal(t, "events.db", dbConfig.DBName)
}

func TestNewConfigFromConfig_Disabled(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Enabled: false,
		},
	}

	dbConfig := NewConfigFromConfig(cfg)
	assert.Nil(t, dbConfig)
}

func TestNewConfigFromConfig_NilConfig(t *testing.T) {
	dbConfig := NewConfigFromConfig(nil)
	assert.Nil(t, dbConfig)
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

	err = client.InitSchema()
	require.NoError(t, err)

	db := client.DB()

	var tableName string
	err = db.Raw(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='conversation_records'",
	).Scan(&tableName).Error
	require.NoError(t, err)
	assert.Equal(t, "conversation_records", tableName)

	record := models.ConversationRecord{
		TraceID:    "test-trace",
		EventType:  "test",
		Timestamp:  db.NowFunc(),
		SessionKey: "test-session",
		Role:       "user",
		Content:    "test content",
	}
	err = db.Create(&record).Error
	require.NoError(t, err)
	assert.NotZero(t, record.ID)

	var indexName string
	err = db.Raw(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_conv_records_event_type'",
	).Scan(&indexName).Error
	require.NoError(t, err)
	assert.Equal(t, "idx_conv_records_event_type", indexName)
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

	err = client.InitSchema()
	require.NoError(t, err)

	db1 := client.DB()
	db2 := client.DB()

	assert.Same(t, db1, db2)

	var count1, count2 int64
	err = db1.Model(&models.ConversationRecord{}).Count(&count1).Error
	require.NoError(t, err)
	err = db2.Model(&models.ConversationRecord{}).Count(&count2).Error
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

	err = client.InitSchema()
	require.NoError(t, err)

	err = client.Close()
	require.NoError(t, err)

	db := client.DB()
	assert.NotNil(t, db)

	var count int64
	err = db.Model(&models.ConversationRecord{}).Count(&count).Error
	assert.Error(t, err)

	_ = client.Close()
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

	err = client.InitSchema()
	require.NoError(t, err)

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			db := client.DB()
			dbPath := client.DBPath()
			assert.NotNil(t, db)
			assert.NotEmpty(t, dbPath)
			done <- true
		}(i)
	}

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

	err = client.InitSchema()
	require.NoError(t, err)

	db := client.DB()
	record := models.ConversationRecord{
		TraceID:    "test-trace",
		EventType:  "test",
		Timestamp:  db.NowFunc(),
		SessionKey: "test-session",
		Role:       "user",
		Content:    "test content",
	}
	err = db.Create(&record).Error
	require.NoError(t, err)

	err = client.InitSchema()
	require.NoError(t, err)

	var count int64
	err = db.Model(&models.ConversationRecord{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestClientIntegration(t *testing.T) {
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

	err = client.InitSchema()
	require.NoError(t, err)

	db := client.DB()

	record := models.ConversationRecord{
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
	err = db.Create(&record).Error
	require.NoError(t, err)
	assert.NotZero(t, record.ID)

	var found models.ConversationRecord
	err = db.First(&found, record.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "test-trace-001", found.TraceID)
	assert.Equal(t, "test content", found.Content)
	assert.Equal(t, 30, found.TotalTokens)

	err = db.Model(&found).Update("content", "updated content").Error
	require.NoError(t, err)

	var updated models.ConversationRecord
	err = db.First(&updated, record.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "updated content", updated.Content)

	err = db.Delete(&updated).Error
	require.NoError(t, err)

	var deleted models.ConversationRecord
	err = db.First(&deleted, record.ID).Error
	assert.Error(t, err)
}

func TestClientWithExistingDBFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&models.ConversationRecord{})
	require.NoError(t, err)

	record := models.ConversationRecord{
		TraceID:    "existing-trace",
		EventType:  "test",
		Timestamp:  db.NowFunc(),
		SessionKey: "test-session",
		Role:       "user",
		Content:    "existing content",
	}
	err = db.Create(&record).Error
	require.NoError(t, err)

	sqlDB, _ := db.DB()
	sqlDB.Close()

	config := &Config{
		DataDir: tmpDir,
		DBName:  "test.db",
	}

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	var found models.ConversationRecord
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

	sqlDB, err := client.DB().DB()
	require.NoError(t, err)
	assert.NotNil(t, sqlDB)
}
