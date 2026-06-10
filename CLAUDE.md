# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 注意事项
- 每次回复用户时都要称呼 [主人]
- 所有用户可见的信息都用中文

## 项目概述

通用的权限管理系统 (RBAC)，以微服务形式独立运行，Go + Gin 技术栈。与业务代码解耦，通过 HTTP API 对外提供权限管理能力。

## 常用命令

```bash
# 安装依赖
go mod tidy

# 构建
go build -o api-rbac ./cmd/server

# 运行服务
go run ./cmd/server

# 运行全部测试
go test ./...

# 运行单个测试
go test -v -run TestXxx ./pkg/xxx/

# 代码格式化
go fmt ./...

# 代码静态检查
go vet ./...
```

## 架构设计

### 分层架构

```
cmd/server/          # 入口，启动 HTTP 服务
internal/
  handler/           # HTTP 处理器，参数绑定与响应
  service/           # 业务逻辑层
  repository/        # 数据访问层 (DB 操作)
  model/             # 数据模型定义
  middleware/        # 中间件 (认证、鉴权等)
pkg/
  errcode/           # 统一错误码
  response/          # 统一响应格式
config/              # 配置读取
migrations/          # 数据库迁移脚本
```

### 核心接口

1. **用户认证**: 登录/登出/Token 刷新
2. **用户管理**: 用户的增删改查
3. **角色管理**: 角色的增删改查
4. **权限管理**: 权限的增删改查
5. **用户-角色绑定**: 为用户分配/移除角色

### 设计原则

- 各层级职责清晰，handler 只做参数校验和响应，service 处理业务逻辑，repository 处理数据持久化
- 统一错误码 + 统一 JSON 响应格式，方便其他语言业务系统集成
- 运行模块与权限校验模块解耦
- JWT 或 Token 方式认证，无状态设计以便水平扩展
