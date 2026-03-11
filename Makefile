.PHONY: help build clean dev dev-backend dev-web test migrate

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  make build      - 构建后端和前端"
	@echo "  make clean      - 清理构建产物"
	@echo "  make dev        - 同时启动后端和前端开发服务器"
	@echo "  make dev-backend - 仅启动后端开发服务器"
	@echo "  make dev-web    - 仅启动前端开发服务器"
	@echo "  make test       - 运行测试"
	@echo "  make migrate    - 运行配置迁移工具"
	@echo "  make fmt        - 格式化代码"
	@echo "  make lint       - 运行代码检查"

# 构建
build:
	@echo "构建后端..."
	go build -o bin/nanobot ./cmd/nanobot
	@echo "构建前端..."
	cd web && npm run build

# 清理
clean:
	rm -rf bin/
	cd web && rm -rf dist/

# 开发模式 - 同时启动后端和前端
dev:
	@echo "========================================="
	@echo "  启动 Nanobot 开发环境"
	@echo "========================================="
	@echo "  后端 API: http://localhost:8081"
	@echo "  前端界面: http://localhost:5173"
	@echo "  按 Ctrl+C 停止所有服务"
	@echo "========================================="
	@(trap 'kill 0' INT; \
		go run ./cmd/nanobot gateway --api --api-port=8081 2>&1 | sed 's/^/[后端] /' & \
		cd web && npm run dev 2>&1 | sed 's/^/[前端] /' & \
		wait)

# 启动后端开发服务器
dev-backend:
	go run ./cmd/nanobot gateway --api --api-port=8081

# 启动前端开发服务器
dev-web:
	cd web && npm run dev

# 运行测试
test:
	go test ./...
	cd web && npm test

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
	cd web && npm run lint
