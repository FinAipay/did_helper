.PHONY: help build run test clean install

.DEFAULT_GOAL := help

help:
	@echo "FinAI DID Helper :"
	@echo ""
	@echo "  make build      - build"
	@echo "  make run        - run with args: make run ARGS=\"your arguments here\""
	@echo "  make test       - run test"
	@echo "  make clean      - clean"
	@echo "  make install    - install"
	@echo "  make help       - show help message"
	@echo ""

# build
build:
	@echo "building..."
	@go build -o did_helper .
	@echo "✓ build completed!"

# run
run:
	@./did_helper $(ARGS)

# run test
test:
	@echo "run test..."
	@go test -v ./...

clean:
	@echo "清理编译文件..."
	@rm -f did_helper
	@go clean

# install
install:
	@echo "安装到GOPATH..."
	@go install

# 查看版本信息
version:
	@./did_helper --version

# Quick command test
quick-test: build
	@echo ""
	@echo "=== test generate ethereum wallet==="
	@./did_helper wallet generate --type ethereum --password "test123456"
	@echo ""
	@echo "=== list wallets ==="
	@./did_helper wallet list
