.PHONY: all build install uninstall clean help

# 检测操作系统
ifeq ($(OS),Windows_NT)
    DETECTED_OS := windows
    BINARY := db-mcp.exe
else
    DETECTED_OS := linux
    BINARY := db-mcp
endif

# 默认目标
all: build

# 编译当前平台
build:
	@echo "Building db-mcp for $(DETECTED_OS)..."
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/main.go
	@echo "[OK] Built: $(BINARY)"

# 安装到系统 (GOPATH)
install: build
	@if [ -z "$(GOPATH)" ]; then \
		echo "Error: GOPATH is not set"; \
		exit 1; \
	fi
	@echo "Installing to $(GOPATH)/bin..."
	@mkdir -p "$(GOPATH)/bin"
	@cp $(BINARY) "$(GOPATH)/bin/$(BINARY)"
	@echo "[OK] Installed to $(GOPATH)/bin/$(BINARY)"

# 卸载
uninstall:
	@if [ -z "$(GOPATH)" ]; then \
		echo "Error: GOPATH is not set"; \
		exit 1; \
	fi
	@echo "Uninstalling..."
	@rm -f "$(GOPATH)/bin/$(BINARY)"
	@echo "[OK] Uninstalled from $(GOPATH)/bin/$(BINARY)"

# 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY)
	@echo "[OK] Cleaned"

# 显示帮助
help:
	@echo "Database MCP Server - Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build     - Build for current platform (default)"
	@echo "  install  - Build and install to GOPATH/bin"
	@echo "  uninstall- Remove from GOPATH/bin"
	@echo "  clean    - Remove build artifacts"
	@echo "  help     - Show this help"
	@echo ""
	@echo "Platform: $(DETECTED_OS)"
	@echo ""
	@echo "Examples:"
	@echo "  make          # build"
	@echo "  make install  # install to GOPATH/bin"
	@echo "  make clean    # remove build artifacts"
