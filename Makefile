.PHONY: help install build dev test clean

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
VERSION ?= v1.0.0
BINARY_NAME = mk

help:
	@echo "mk - Go API 脚手架工具"
	@echo ""
	@echo "可用命令:"
	@echo "  make install     - 安装依赖"
	@echo "  make build       - 构建二进制文件"
	@echo "  make dev         - 开发模式运行"
	@echo "  make test        - 运行测试"
	@echo "  make clean       - 清理构建文件"

install:
	@echo "安装依赖..."
	go mod download
	go mod tidy

build:
	@echo "构建 $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) ./

dev: install
	@echo "以开发模式运行..."
	go run ./main.go

test:
	@echo "运行测试..."
	go test ./... -v

clean:
	@echo "清理构建文件..."
	rm -rf bin/
	go clean
