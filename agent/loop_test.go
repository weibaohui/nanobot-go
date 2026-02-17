package agent

import (
	"testing"

	"github.com/weibaohui/nanobot-go/bus"
	"github.com/weibaohui/nanobot-go/config"
	"github.com/weibaohui/nanobot-go/session"
	"go.uber.org/zap"
)

// TestNewLoop 测试创建代理循环
func TestNewLoop(t *testing.T) {
	t.Run("配置为空返回nil", func(t *testing.T) {
		loop := NewLoop(nil)
		if loop != nil {
			t.Error("NewLoop(nil) 应该返回 nil")
		}
	})

	t.Run("基本配置", func(t *testing.T) {
		cfg := &config.Config{}
		logger := zap.NewNop()
		messageBus := bus.NewMessageBus(logger)
		sessionMgr := session.NewManager(cfg, "/tmp")

		loop := NewLoop(&LoopConfig{
			Config:         cfg,
			MessageBus:     messageBus,
			Workspace:      "/tmp/workspace",
			MaxIterations:  10,
			ExecTimeout:    30,
			SessionManager: sessionMgr,
			Logger:         logger,
		})

		if loop == nil {
			t.Fatal("NewLoop() 返回 nil")
		}

		if loop.workspace != "/tmp/workspace" {
			t.Errorf("loop.workspace = %q, 期望 /tmp/workspace", loop.workspace)
		}

		if loop.maxIterations != 10 {
			t.Errorf("loop.maxIterations = %d, 期望 10", loop.maxIterations)
		}

		if loop.execTimeout != 30 {
			t.Errorf("loop.execTimeout = %d, 期望 30", loop.execTimeout)
		}

		if loop.bus == nil {
			t.Error("loop.bus 不应该为 nil")
		}

		if loop.tools == nil {
			t.Error("loop.tools 不应该为 nil")
		}

		if loop.interruptManager == nil {
			t.Error("loop.interruptManager 不应该为 nil")
		}
	})

	t.Run("无logger使用默认", func(t *testing.T) {
		cfg := &config.Config{}
		logger := zap.NewNop()
		messageBus := bus.NewMessageBus(logger)

		loop := NewLoop(&LoopConfig{
			Config:     cfg,
			MessageBus: messageBus,
			Workspace:  "/tmp/workspace",
		})

		if loop == nil {
			t.Fatal("NewLoop() 返回 nil")
		}

		if loop.logger == nil {
			t.Error("loop.logger 应该使用默认的 nop logger")
		}
	})
}

// TestLoop_Stop 测试停止代理循环
func TestLoop_Stop(t *testing.T) {
	cfg := &config.Config{}
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)

	loop := NewLoop(&LoopConfig{
		Config:     cfg,
		MessageBus: messageBus,
		Workspace:  "/tmp/workspace",
		Logger:     logger,
	})

	if loop == nil {
		t.Fatal("NewLoop() 返回 nil")
	}

	if loop.running {
		t.Error("新创建的 loop 应该 running = false")
	}

	loop.Stop()

	if loop.running {
		t.Error("Stop() 后 running 应该为 false")
	}
}

// TestLoop_GetMasterAgent 测试获取 Master Agent
func TestLoop_GetMasterAgent(t *testing.T) {
	cfg := &config.Config{}
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)

	loop := NewLoop(&LoopConfig{
		Config:     cfg,
		MessageBus: messageBus,
		Workspace:  "/tmp/workspace",
		Logger:     logger,
	})

	if loop == nil {
		t.Fatal("NewLoop() 返回 nil")
	}

	agent := loop.GetMasterAgent()
	_ = agent
}

// TestLoop_GetSupervisor 测试获取 Supervisor Agent
func TestLoop_GetSupervisor(t *testing.T) {
	cfg := &config.Config{}
	logger := zap.NewNop()
	messageBus := bus.NewMessageBus(logger)

	loop := NewLoop(&LoopConfig{
		Config:     cfg,
		MessageBus: messageBus,
		Workspace:  "/tmp/workspace",
		Logger:     logger,
	})

	if loop == nil {
		t.Fatal("NewLoop() 返回 nil")
	}

	supervisor := loop.GetSupervisor()
	_ = supervisor
}

// TestLoopConfig 测试 LoopConfig 结构
func TestLoopConfig(t *testing.T) {
	logger := zap.NewNop()
	cfg := &LoopConfig{
		Config:              &config.Config{},
		MessageBus:          bus.NewMessageBus(logger),
		Workspace:           "/test/workspace",
		MaxIterations:       20,
		ExecTimeout:         60,
		RestrictToWorkspace: true,
		Logger:              logger,
	}

	if cfg.Workspace != "/test/workspace" {
		t.Errorf("LoopConfig.Workspace = %q, 期望 /test/workspace", cfg.Workspace)
	}

	if cfg.MaxIterations != 20 {
		t.Errorf("LoopConfig.MaxIterations = %d, 期望 20", cfg.MaxIterations)
	}

	if !cfg.RestrictToWorkspace {
		t.Error("LoopConfig.RestrictToWorkspace 应该为 true")
	}
}
