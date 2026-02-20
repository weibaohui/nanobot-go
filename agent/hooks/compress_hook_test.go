package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

func TestCompressHook_Name(t *testing.T) {
	cfg := &config.Config{
		Compress: config.CompressConfig{
			Enabled:    true,
			MinMessages: 20,
			MinTokens:  50000,
			MaxHistory: 5,
		},
	}
	hook := NewCompressHook(cfg, zap.NewNop(), nil, nil)

	if hook.Name() != "compress" {
		t.Errorf("Expected name 'compress', got '%s'", hook.Name())
	}
}

func TestCompressHook_ShouldCompress(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		session      *session.Session
		should       bool
	}{
		{
			name: "should compress when thresholds met",
			cfg: &config.Config{
				Compress: config.CompressConfig{
					MinMessages: 20,
					MinTokens:  50000,
				},
			},
			session: &session.Session{
				Messages: make([]session.Message, 20),
				TokenUsage: session.TokenUsage{
					TotalTokens: 50000,
				},
			},
			should: true,
		},
		{
			name: "should not compress when message count below threshold",
			cfg: &config.Config{
				Compress: config.CompressConfig{
					MinMessages: 20,
					MinTokens:  50000,
				},
			},
			session: &session.Session{
				Messages: make([]session.Message, 10),
				TokenUsage: session.TokenUsage{
					TotalTokens: 60000,
				},
			},
			should: false,
		},
		{
			name: "should not compress when token count below threshold",
			cfg: &config.Config{
				Compress: config.CompressConfig{
					MinMessages: 20,
					MinTokens:  50000,
				},
			},
			session: &session.Session{
				Messages: make([]session.Message, 25),
				TokenUsage: session.TokenUsage{
					TotalTokens: 30000,
				},
			},
			should: false,
		},
		{
			name: "should not compress when both below threshold",
			cfg: &config.Config{
				Compress: config.CompressConfig{
					MinMessages: 20,
					MinTokens:  50000,
				},
			},
			session: &session.Session{
				Messages: make([]session.Message, 10),
				TokenUsage: session.TokenUsage{
					TotalTokens: 30000,
				},
			},
			should: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewCompressHook(tt.cfg, zap.NewNop(), nil, nil)
			result := hook.shouldCompress(tt.session)
			if result != tt.should {
				t.Errorf("Expected %v, got %v", tt.should, result)
			}
		})
	}
}

func TestCompressHook_ParseExtractionResult(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		longTerm  string
		shortTerm string
		hasError  bool
	}{
		{
			name: "valid JSON",
			content: `{"long_term": "User likes Python","short_term": "Working on web project"}`,
			longTerm:  "User likes Python",
			shortTerm: "Working on web project",
			hasError:  false,
		},
		{
			name: "JSON with extra text",
			content: `Here is the result: {"long_term": "User prefers dark mode","short_term": "Debugging issue"}. End.`,
			longTerm:  "User prefers dark mode",
			shortTerm: "Debugging issue",
			hasError:  false,
		},
		{
			name:      "no JSON",
			content:   `No JSON here`,
			longTerm:  "",
			shortTerm: "",
			hasError:  true,
		},
		{
			name:      "incomplete JSON",
			content:   `{"long_term": "test"`,
			longTerm:  "",
			shortTerm: "",
			hasError:  true,
		},
		{
			name: "empty values",
			content: `{"long_term":"","short_term":""}`,
			longTerm:  "",
			shortTerm: "",
			hasError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewCompressHook(&config.Config{}, zap.NewNop(), nil, nil)
			longTerm, shortTerm, err := hook.parseExtractionResult(tt.content)

			if tt.hasError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if longTerm != tt.longTerm {
				t.Errorf("Expected longTerm '%s', got '%s'", tt.longTerm, longTerm)
			}
			if shortTerm != tt.shortTerm {
				t.Errorf("Expected shortTerm '%s', got '%s'", tt.shortTerm, shortTerm)
			}
		})
	}
}

func TestCompressHook_CleanupSession(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		initialCount  int
		maxHistory    int
		expectedCount int
	}{
		{
			name: "reduce to maxHistory",
			cfg: &config.Config{
				Compress: config.CompressConfig{
					MaxHistory: 5,
				},
			},
			initialCount:  20,
			maxHistory:    5,
			expectedCount: 5,
		},
		{
			name: "keep all if below maxHistory",
			cfg: &config.Config{
				Compress: config.CompressConfig{
					MaxHistory: 10,
				},
			},
			initialCount:  5,
			maxHistory:    10,
			expectedCount: 5,
		},
		{
			name: "use default maxHistory",
			cfg: &config.Config{
				Compress: config.CompressConfig{
					MaxHistory: 0,
				},
			},
			initialCount:  20,
			maxHistory:    5, // default
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewCompressHook(tt.cfg, zap.NewNop(), nil, nil)

			sess := &session.Session{
				Messages: make([]session.Message, tt.initialCount),
				TokenUsage: session.TokenUsage{
					TotalTokens: 100000,
				},
			}

			for i := range sess.Messages {
				sess.Messages[i] = session.Message{
					Role:       "user",
					Content:    "test message",
					Timestamp:  time.Now(),
					TokenUsage: nil,
				}
			}

			hook.cleanupSession(sess)

			if len(sess.Messages) != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, len(sess.Messages))
			}

			if sess.TokenUsage.TotalTokens != 0 {
				t.Errorf("Expected token usage to be reset, got %d", sess.TokenUsage.TotalTokens)
			}
		})
	}
}

func TestCompressHook_BuildDialogueSummary(t *testing.T) {
	cfg := &config.Config{
		Compress: config.CompressConfig{
			MaxHistory: 5,
		},
	}
	hook := NewCompressHook(cfg, zap.NewNop(), nil, nil)

	sess := &session.Session{
		Messages: []session.Message{
			{
				Role:       "user",
				Content:    "Hello",
				Timestamp:  time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC),
				TokenUsage: nil,
			},
			{
				Role:       "assistant",
				Content:    "Hi there!",
				Timestamp:  time.Date(2026, 2, 21, 10, 0, 1, 0, time.UTC),
				TokenUsage: nil,
			},
		},
	}

	summary := hook.buildDialogueSummary(sess)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	if !contains(summary, "对话历史") {
		t.Error("Summary should contain '对话历史'")
	}
	if !contains(summary, "user") {
		t.Error("Summary should contain user role")
	}
	if !contains(summary, "Hello") {
		t.Error("Summary should contain message content")
	}
}

func TestCompressHook_AfterMessageProcess(t *testing.T) {
	cfg := &config.Config{
		Compress: config.CompressConfig{
			Enabled:    false, // Disabled
			MinMessages: 20,
			MinTokens:  50000,
			MaxHistory: 5,
		},
	}
	hook := NewCompressHook(cfg, zap.NewNop(), nil, nil)

	// Below thresholds - should not compress
	sess := &session.Session{
		Messages: make([]session.Message, 10),
		TokenUsage: session.TokenUsage{
			TotalTokens: 30000,
		},
	}

	ctx := context.Background()
	err := hook.AfterMessageProcess(ctx, nil, sess, "test response")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
