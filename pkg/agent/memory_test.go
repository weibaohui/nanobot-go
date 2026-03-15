package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewMemoryStore 测试创建内存存储
func TestNewMemoryStore(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewMemoryStore(tmpDir)
	if store == nil {
		t.Fatal("NewMemoryStore 返回 nil")
	}

	expectedMemoryDir := filepath.Join(tmpDir, "memory")
	if store.memoryDir != expectedMemoryDir {
		t.Errorf("memoryDir = %q, 期望 %q", store.memoryDir, expectedMemoryDir)
	}

	if _, err := os.Stat(expectedMemoryDir); os.IsNotExist(err) {
		t.Errorf("内存目录 %q 应该被创建", expectedMemoryDir)
	}
}

// TestMemoryStore_GetTodayFile 测试获取今日内存文件路径
func TestMemoryStore_GetTodayFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewMemoryStore(tmpDir)

	path := store.GetTodayFile()
	today := time.Now().Format("2006-01-02")
	expected := filepath.Join(store.memoryDir, today+".md")

	if path != expected {
		t.Errorf("GetTodayFile() = %q, 期望 %q", path, expected)
	}
}

// TestMemoryStore_ReadToday 测试读取今日内存笔记
func TestMemoryStore_ReadToday(t *testing.T) {
	t.Run("读取存在的文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		todayFile := store.GetTodayFile()
		content := "# 今日笔记\n\n测试内容"
		os.WriteFile(todayFile, []byte(content), 0644)

		result := store.ReadToday()
		if result != content {
			t.Errorf("ReadToday() = %q, 期望 %q", result, content)
		}
	})

	t.Run("读取不存在的文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		result := store.ReadToday()
		if result != "" {
			t.Errorf("ReadToday() = %q, 期望空字符串", result)
		}
	})
}

// TestMemoryStore_AppendToday 测试追加内容到今日内存笔记
func TestMemoryStore_AppendToday(t *testing.T) {
	t.Run("追加到新文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		err := store.AppendToday("新笔记内容")
		if err != nil {
			t.Fatalf("AppendToday 返回错误: %v", err)
		}

		content := store.ReadToday()
		today := time.Now().Format("2006-01-02")
		if content == "" {
			t.Fatal("文件内容为空")
		}

		if content[:len("# "+today)] != "# "+today {
			t.Errorf("文件应该以日期标题开头")
		}
	})

	t.Run("追加到已存在的文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		store.AppendToday("第一条内容")
		store.AppendToday("第二条内容")

		content := store.ReadToday()
		if content == "" {
			t.Fatal("文件内容为空")
		}

		if len(content) < 20 {
			t.Errorf("内容太短，可能追加失败")
		}
	})
}

// TestMemoryStore_ReadLongTerm 测试读取长期内存
func TestMemoryStore_ReadLongTerm(t *testing.T) {
	t.Run("读取存在的文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		content := "# 长期内存\n\n重要信息"
		os.WriteFile(store.memoryFile, []byte(content), 0644)

		result := store.ReadLongTerm()
		if result != content {
			t.Errorf("ReadLongTerm() = %q, 期望 %q", result, content)
		}
	})

	t.Run("读取不存在的文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		result := store.ReadLongTerm()
		if result != "" {
			t.Errorf("ReadLongTerm() = %q, 期望空字符串", result)
		}
	})
}

// TestMemoryStore_WriteLongTerm 测试写入长期内存
func TestMemoryStore_WriteLongTerm(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewMemoryStore(tmpDir)

	content := "# 长期内存\n\n重要信息"
	err := store.WriteLongTerm(content)
	if err != nil {
		t.Fatalf("WriteLongTerm 返回错误: %v", err)
	}

	result := store.ReadLongTerm()
	if result != content {
		t.Errorf("ReadLongTerm() = %q, 期望 %q", result, content)
	}
}

// TestMemoryStore_GetRecentMemories 测试获取最近 N 天的内存
func TestMemoryStore_GetRecentMemories(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewMemoryStore(tmpDir)

	// 创建今天和昨天的内存文件
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	todayFile := filepath.Join(store.memoryDir, today.Format("2006-01-02")+".md")
	yesterdayFile := filepath.Join(store.memoryDir, yesterday.Format("2006-01-02")+".md")

	os.WriteFile(todayFile, []byte("今日内容"), 0644)
	os.WriteFile(yesterdayFile, []byte("昨日内容"), 0644)

	memories := store.GetRecentMemories(2)

	if memories == "" {
		t.Fatal("GetRecentMemories 返回空字符串")
	}

	if len(memories) < 10 {
		t.Errorf("内存内容太短: %q", memories)
	}
}

// TestMemoryStore_ListMemoryFiles 测试列出所有内存文件
func TestMemoryStore_ListMemoryFiles(t *testing.T) {
	t.Run("列出多个文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		// 创建多个内存文件
		dates := []string{"2024-01-15", "2024-01-16", "2024-01-17"}
		for _, date := range dates {
			file := filepath.Join(store.memoryDir, date+".md")
			os.WriteFile(file, []byte("内容"), 0644)
		}

		files := store.ListMemoryFiles()
		if len(files) != 3 {
			t.Errorf("ListMemoryFiles 返回 %d 个文件, 期望 3", len(files))
		}

		// 验证排序（最新的在前）
		if len(files) >= 2 {
			if files[0] < files[1] {
				t.Error("文件应该按日期降序排列")
			}
		}
	})

	t.Run("空目录", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		files := store.ListMemoryFiles()
		if len(files) != 0 {
			t.Errorf("ListMemoryFiles 返回 %d 个文件, 期望 0", len(files))
		}
	})

	t.Run("忽略非日期格式文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		// 创建正确格式的文件
		os.WriteFile(filepath.Join(store.memoryDir, "2024-01-15.md"), []byte("内容"), 0644)
		// 创建错误格式的文件
		os.WriteFile(filepath.Join(store.memoryDir, "MEMORY.md"), []byte("内容"), 0644)
		os.WriteFile(filepath.Join(store.memoryDir, "notes.md"), []byte("内容"), 0644)

		files := store.ListMemoryFiles()
		if len(files) != 1 {
			t.Errorf("ListMemoryFiles 返回 %d 个文件, 期望 1 (忽略非日期格式文件)", len(files))
		}
	})
}

// TestMemoryStore_GetMemoryContext 测试获取内存上下文
func TestMemoryStore_GetMemoryContext(t *testing.T) {
	t.Run("有长期内存和今日笔记", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		store.WriteLongTerm("长期内存内容")
		store.AppendToday("今日笔记内容")

		context := store.GetMemoryContext()
		if context == "" {
			t.Fatal("GetMemoryContext 返回空字符串")
		}

		if len(context) < 20 {
			t.Errorf("上下文内容太短: %q", context)
		}
	})

	t.Run("只有长期内存", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		store.WriteLongTerm("长期内存内容")

		context := store.GetMemoryContext()
		if context == "" {
			t.Fatal("GetMemoryContext 返回空字符串")
		}
	})

	t.Run("只有今日笔记", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		store.AppendToday("今日笔记内容")

		context := store.GetMemoryContext()
		if context == "" {
			t.Fatal("GetMemoryContext 返回空字符串")
		}
	})

	t.Run("无任何内存", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := NewMemoryStore(tmpDir)

		context := store.GetMemoryContext()
		if context != "" {
			t.Errorf("GetMemoryContext = %q, 期望空字符串", context)
		}
	})
}
