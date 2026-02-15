.PHONY: help dev build clean

# 项目名称
PROJECT_NAME := nanobot

# 交叉编译的目标平台列表
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

# 编译输出目录
OUT_DIR := bin

# Go 编译参数
GO_BUILD_FLAGS := -trimpath -ldflags="-s -w"

help:
	@echo "可用的 Make 目标:"
	@echo "  make dev    - 使用 air 进行开发（热重载）"
	@echo "  make build  - 交叉编译到所有目标平台"
	@echo "  make clean  - 清理编译输出"
	@echo "  make help   - 显示此帮助信息"

dev:
	@echo "启动开发服务器（air）..."
	@command -v air >/dev/null 2>&1 || (echo "错误: air 未安装，请运行: go install github.com/cosmtrek/air@latest" && exit 1)
	air

build: clean
	@echo "开始交叉编译..."
	@mkdir -p $(OUT_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} \
		GOARCH=$${platform#*/} \
		CGO_ENABLED=0 \
		OUT_FILE=$(OUT_DIR)/$(PROJECT_NAME)-$${platform%/*}-$${platform#*/} ; \
		if [ "$${platform%/*}" = "windows" ]; then \
			OUT_FILE=$${OUT_FILE}.exe ; \
		fi ; \
		echo "编译: $${platform} -> $${OUT_FILE}" ; \
		GOOS=$${platform%/*} \
		GOARCH=$${platform#*/} \
		CGO_ENABLED=0 \
		go build $(GO_BUILD_FLAGS) -o $${OUT_FILE} . || exit 1 ; \
	done
	@echo "编译完成！输出目录: $(OUT_DIR)"

clean:
	@echo "清理编译输出..."
	@rm -rf $(OUT_DIR)
	@echo "清理完成"
