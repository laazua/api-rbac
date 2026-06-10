.PHONY: build run test clean fmt vet

# 构建
build:
	go build -o api-rbac ./cmd/server

# 运行
run:
	go run ./cmd/server

# 测试
test:
	go test ./...

# 代码格式化
fmt:
	go fmt ./...

# 静态检查
vet:
	go vet ./...

# 清理
clean:
	rm -f api-rbac

# 安装依赖
deps:
	go mod tidy
