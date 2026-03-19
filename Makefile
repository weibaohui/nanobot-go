.PHONY: help build clean dev dev-backend dev-web stop test migrate

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  make build      - 构建后端和前端"
	@echo "  make clean      - 清理构建产物"
	@echo "  make dev        - 同时启动后端和前端开发服务器"
	@echo "  make dev-backend - 仅启动后端开发服务器"
	@echo "  make dev-web    - 仅启动前端开发服务器"
	@echo "  make stop       - 停止所有 nanobot 进程"
	@echo "  make test       - 运行测试"
	@echo "  make setup      - 安装依赖"
	@echo "  make migrate    - 运行配置迁移工具"
	@echo "  make fmt        - 格式化代码"
	@echo "  make lint       - 运行代码检查"

# Setup - install dependencies
setup:
	go mod tidy
	cd web && pnpm install
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/air-verse/air@latest; }


# 构建
build:
	@echo "构建后端..."
	go build -o bin/nanobot ./cmd/nanobot
	@echo "构建前端..."
	cd web && pnpm run build

# 清理
clean:
	rm -rf bin/
	cd web && rm -rf dist/

# 开发模式 - 同时启动后端和前端
dev:
	@echo "========================================="
	@echo "  启动 Nanobot 开发环境"
	@echo "========================================="
	@echo "  后端 API: http://localhost:8080"
	@echo "  前端界面: http://localhost:5173"
	@echo "  按 Ctrl+C 停止所有服务"
	@echo "========================================="
	@(trap 'kill 0' INT; \
		air 2>&1 & \
		cd web && pnpm run dev 2>&1 & \
		wait)

# 启动后端开发服务器
dev-backend:
	go run ./cmd/nanobot gateway --api --api-port=8080

# 启动前端开发服务器
dev-web:
	cd web && pnpm run dev

# 停止所有 nanobot 进程
stop:
	@echo "正在停止所有 nanobot 相关进程..."
	@echo "  - 停止 go run nanobot..."
	@ps -ef | grep "go run ./cmd/nanobot" | grep -v grep | awk '{print $$2}' | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 nanobot gateway 二进制..."
	@ps -ef | grep "nanobot gateway" | grep -v grep | awk '{print $$2}' | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 make dev 相关 shell..."
	@ps -ef | grep -E "make dev|trap.*kill 0" | grep -v grep | awk '{print $$2}' | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 node/vite..."
	@pgrep -f "vite" | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 esbuild..."
	@pgrep -f "esbuild.*nanobot" | xargs -I {} kill -9 {} 2>/dev/null || true
	@sleep 1
	@remaining=$$(ps -ef | grep -E "(nanobot|go run.*nanobot)" | grep -v grep | wc -l); \
	if [ $$remaining -eq 0 ]; then \
		echo "已停止所有 nanobot 进程"; \
	else \
		echo "警告: 仍有 $$remaining 个进程在运行"; \
		ps -ef | grep -E "(nanobot|go run.*nanobot)" | grep -v grep; \
	fi

# 运行测试
test:
	go test ./...
	cd web && pnpm test

# 配置迁移
migrate:
	go run ./cmd/migrate/main.go ~/.nanobot/config.json

# 格式化代码
fmt:
	go fmt ./...
	gofmt -w .

# 代码检查
lint:
	golangci-lint run
	cd web && pnpm run lint
