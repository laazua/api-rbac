# RBAC 权限管理系统 — 使用说明

## 目录

- [1. 快速开始](#1-快速开始)
- [2. 集成方案](#2-集成方案)
- [3. Go SDK 使用](#3-go-sdk-使用)
- [4. API 接口示例](#4-api-接口示例)
  - [4.1 健康检查](#41-健康检查)
  - [4.2 用户登录](#42-用户登录)
  - [4.3 Token 验证](#43-token-验证)
  - [4.4 权限检查](#44-权限检查)
  - [4.5 用户管理](#45-用户管理)
  - [4.6 密码修改](#46-密码修改)
  - [4.7 用户-角色绑定](#47-用户-角色绑定)
  - [4.8 角色管理](#48-角色管理)
  - [4.9 角色-权限绑定](#49-角色-权限绑定)
  - [4.10 权限管理](#410-权限管理)
- [5. 常见业务流程](#5-常见业务流程)

---

## 1. 快速开始

### 1.1 修改配置

编辑 `config/config.yaml`：

```yaml
server:
  mode: debug          # debug | release
  port: 8087

db:
  host: 127.0.0.1
  port: 3306
  user: root
  password: "your_db_password"
  dbname: api_rbac
  charset: utf8mb4

jwt:
  secret: "change-me-in-production"
  expire_hour: 24

cors:
  allow_origins:
    - "*"
```

### 1.2 创建数据库

```sql
CREATE DATABASE IF NOT EXISTS api_rbac CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 1.3 编译启动

```bash
# 编译
make build

# 首次启动 — 设置超级管理员密码
./api-rbac

# 后续启动
./api-rbac
```

> 首次启动时会交互式提示设置 admin 用户密码，数据表自动创建。

### 1.4 访问 Swagger

```
http://localhost:8087/swagger/index.html
```

---

## 2. 集成方案

### 2.1 整体架构

```
┌───────────────────────┐          ┌───────────────────────┐
│  业务系统 (任意语言)    │          │    RBAC 服务 (Go)      │
│                       │  HTTP    │                       │
│  ┌─────────────────┐  │────────→ │  POST /auth/login     │
│  │ 登录页面         │  │  JWT     │  POST /auth/verify    │
│  └─────────────────┘  │  Token   │  POST /auth/check     │
│                       │←─────────│                       │
│  ┌─────────────────┐  │          │  CRUD /users          │
│  │ 业务接口          │  │──Token──→│  CRUD /roles          │
│  │ 收到请求时调用    │  │  ←allowed│  CRUD /permissions    │
│  │ RBAC 权限检查    │  │          │                       │
│  └─────────────────┘  │          │                       │
└───────────────────────┘          └───────────────────────┘
```

### 2.2 集成流程

业务系统集成 RBAC 只需三步：

```
1. 用户登录 → 调 POST /api/v1/auth/login → 拿到 JWT Token
2. 业务接口收到请求 → 调 POST /api/v1/auth/check → 判断 allowed
3. (可选) 调 POST /api/v1/auth/verify → 校验 Token 是否过期
```

### 2.3 伪代码示例

```python
import requests

RBAC_URL = "http://rbac-service:8087/api/v1"

def handle_user_request(token, resource, action):
    """业务系统在收到请求时校验权限"""
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.post(f"{RBAC_URL}/auth/check",
        json={"resource": resource, "action": action},
        headers=headers)
    return resp.json()["data"]["allowed"]

# 用户请求删除文章
if not handle_user_request(token, "article", "delete"):
    return {"code": 403, "message": "无权限"}
# 执行业务逻辑...
```

### 2.4 Token 传递方式

所有需要认证的接口统一在 Header 中传递：

```
Authorization: Bearer <JWT_TOKEN>
```

---

## 3. Go SDK 使用

如果你的业务系统也是 Go + Gin，可以直接复用本项目的 SDK。

### 3.1 引入 Go SDK

```go
import "api-rbac/pkg/client"
```

### 3.2 SDK 调用示例

```go
package main

import (
    "fmt"
    "api-rbac/pkg/client"
)

func main() {
    c := client.NewRBACClient("http://localhost:8087/api/v1")

    // 1. 登录
    loginResp, err := c.Login("admin", "admin123")
    if err != nil {
        panic(err)
    }
    token := loginResp.Data.Token
    fmt.Printf("登录成功, Token: %s\n", token)

    // 2. 验证 Token
    verifyResp, _ := c.Verify(token)
    fmt.Printf("用户: %s (ID: %d)\n", verifyResp.Data.Username, verifyResp.Data.UserID)

    // 3. 检查权限
    checkResp, _ := c.CheckPermission(token, "user", "delete")
    fmt.Printf("权限检查结果: %v\n", checkResp.Data.Allowed)
}
```

### 3.3 SDK 中间件 (Gin)

```go
package main

import (
    "github.com/gin-gonic/gin"
    "api-rbac/pkg/client"
)

func main() {
    r := gin.Default()
    rbacClient := client.NewRBACClient("http://localhost:8087/api/v1")

    // 对 /admin/* 路径统一要求 "admin:access" 权限
    admin := r.Group("/admin")
    admin.Use(client.PermissionGuard(rbacClient, "admin", "access"))
    {
        admin.GET("/dashboard", func(c *gin.Context) { /* ... */ })
    }

    r.Run(":9090")
}
```

---

## 4. API 接口示例

> 所有示例默认服务运行在 `http://localhost:8087`，JWT Token 放在 Header 中。

### 4.1 健康检查

```bash
curl -X GET http://localhost:8087/health
```

响应:
```json
{ "status": "ok" }
```

---

### 4.2 用户登录

支持**用户名**或**邮箱**登录，统一填入 `account` 字段。

```bash
# 用户名登录
curl -X POST http://localhost:8087/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account": "admin",
    "password": "your_password"
  }'

# 邮箱登录
curl -X POST http://localhost:8087/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "account": "admin@localhost",
    "password": "your_password"
  }'
```

响应:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": 1,
    "username": "admin"
  }
}
```

---

### 4.3 Token 验证

业务系统收到用户请求后可调用此接口校验 Token 是否有效。

```bash
curl -X POST http://localhost:8087/api/v1/auth/verify \
  -H "Authorization: Bearer <TOKEN>"
```

响应:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1,
    "username": "admin"
  }
}
```

---

### 4.4 权限检查

业务系统执行敏感操作前调用此接口做鉴权。

```bash
curl -X POST http://localhost:8087/api/v1/auth/check \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "resource": "user",
    "action": "delete"
  }'
```

有权限:
```json
{ "code": 0, "message": "success", "data": { "allowed": true } }
```

无权限:
```json
{ "code": 1003, "message": "无权限", "data": { "allowed": false } }
```

---

### 4.5 用户管理

#### 创建用户

```bash
curl -X POST http://localhost:8087/api/v1/users \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "zhangsan",
    "password": "123456",
    "email": "zhangsan@example.com"
  }'
```

响应:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 2,
    "username": "zhangsan",
    "email": "zhangsan@example.com",
    "status": 1,
    "created_at": "2026-06-10T15:00:00+08:00",
    "updated_at": "2026-06-10T15:00:00+08:00"
  }
}
```

#### 获取用户列表 (分页 + 搜索)

```bash
# 基础分页
curl "http://localhost:8087/api/v1/users?page=1&page_size=10" \
  -H "Authorization: Bearer <TOKEN>"

# 关键词搜索
curl "http://localhost:8087/api/v1/users?page=1&page_size=10&keyword=admin" \
  -H "Authorization: Bearer <TOKEN>"
```

响应:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [{ "id": 1, "username": "admin", ... }],
    "total": 1,
    "page": 1,
    "page_size": 10
  }
}
```

#### 获取用户详情 (含角色和权限)

```bash
curl -X GET http://localhost:8087/api/v1/users/1 \
  -H "Authorization: Bearer <TOKEN>"
```

响应:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@localhost",
    "status": 1,
    "roles": [
      {
        "id": 1,
        "name": "超级管理员",
        "description": "内置超级管理员角色",
        "permissions": [
          { "id": 1, "name": "超级管理员权限", "resource": "*", "action": "*" }
        ]
      }
    ]
  }
}
```

#### 更新用户

```bash
curl -X PUT http://localhost:8087/api/v1/users/2 \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newemail@example.com",
    "status": 0
  }'
```

响应:
```json
{ "code": 0, "message": "success" }
```

> `status`: 1=启用, 0=禁用。禁用后该用户无法登录且 Token 即时失效。

#### 删除用户

```bash
curl -X DELETE http://localhost:8087/api/v1/users/2 \
  -H "Authorization: Bearer <TOKEN>"
```

响应:
```json
{ "code": 0, "message": "success" }
```

---

### 4.6 密码修改

```bash
curl -X PUT http://localhost:8087/api/v1/users/2/password \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "123456",
    "new_password": "new_secure_pwd"
  }'
```

响应:
```json
{ "code": 0, "message": "success" }
```

---

### 4.7 用户-角色绑定

#### 为用户分配角色 (覆盖式)

```bash
curl -X POST http://localhost:8087/api/v1/users/2/roles \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "role_ids": [1, 2]
  }'
```

响应:
```json
{ "code": 0, "message": "success" }
```

> 此操作为**覆盖式**更新，传入的 `role_ids` 将完全替代该用户当前的所有角色。

#### 移除用户的某个角色

```bash
curl -X DELETE http://localhost:8087/api/v1/users/2/roles/1 \
  -H "Authorization: Bearer <TOKEN>"
```

响应:
```json
{ "code": 0, "message": "success" }
```

---

### 4.8 角色管理

#### 创建角色

```bash
curl -X POST http://localhost:8087/api/v1/roles \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "编辑员",
    "description": "负责内容编辑的角色"
  }'
```

#### 获取角色列表

```bash
curl "http://localhost:8087/api/v1/roles?page=1&page_size=10&keyword=管理员" \
  -H "Authorization: Bearer <TOKEN>"
```

#### 获取角色详情 (含权限)

```bash
curl -X GET http://localhost:8087/api/v1/roles/1 \
  -H "Authorization: Bearer <TOKEN>"
```

#### 更新角色

```bash
curl -X PUT http://localhost:8087/api/v1/roles/2 \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高级编辑员",
    "description": "负责高级内容编辑的角色"
  }'
```

#### 删除角色

```bash
curl -X DELETE http://localhost:8087/api/v1/roles/2 \
  -H "Authorization: Bearer <TOKEN>"
```

---

### 4.9 角色-权限绑定

#### 为角色分配权限 (覆盖式)

```bash
curl -X POST http://localhost:8087/api/v1/roles/2/permissions \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "permission_ids": [1, 2, 3]
  }'
```

> 同样为**覆盖式**更新。

#### 移除角色的某个权限

```bash
curl -X DELETE http://localhost:8087/api/v1/roles/2/permissions/1 \
  -H "Authorization: Bearer <TOKEN>"
```

---

### 4.10 权限管理

#### 创建权限

权限由 `resource` (资源) 和 `action` (操作) 组成，例如"用户模块的删除权限":

```bash
curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "删除用户",
    "resource": "user",
    "action": "delete",
    "description": "允许删除其他用户账号"
  }'
```

常用权限定义示例:

| 权限 name | resource | action | 说明 |
|---|---|---|---|
| 查看用户 | user | read | 查看用户列表和详情 |
| 创建用户 | user | create | 创建新用户 |
| 更新用户 | user | update | 修改用户信息 |
| 删除用户 | user | delete | 删除用户 |
| 编辑文章 | article | edit | 编辑文章内容 |
| 发布文章 | article | publish | 发布文章 |
| 删除文章 | article | delete | 删除文章 |
| 系统管理 | system | manage | 系统级管理操作 |

> `resource="*" action="*"` 为通配符权限，匹配任何资源和操作（通常只给超级管理员）。

#### 获取权限列表

```bash
curl "http://localhost:8087/api/v1/permissions?page=1&page_size=10&keyword=删除" \
  -H "Authorization: Bearer <TOKEN>"
```

#### 获取权限详情

```bash
curl -X GET http://localhost:8087/api/v1/permissions/1 \
  -H "Authorization: Bearer <TOKEN>"
```

#### 更新权限

```bash
curl -X PUT http://localhost:8087/api/v1/permissions/2 \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "删除任意用户",
    "resource": "user",
    "action": "delete",
    "description": "允许删除任意用户账号"
  }'
```

#### 删除权限

```bash
curl -X DELETE http://localhost:8087/api/v1/permissions/2 \
  -H "Authorization: Bearer <TOKEN>"
```

---

## 5. 常见业务流程

### 5.1 新业务模块接入

假设业务系统新增了"订单管理"模块，需要接入权限控制：

**第一步：定义权限**

```bash
# 在 RBAC 系统中创建 4 个基础权限
curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"查看订单","resource":"order","action":"read","description":"查看订单列表和详情"}'

curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"创建订单","resource":"order","action":"create","description":"创建新订单"}'

curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"删除订单","resource":"order","action":"delete","description":"删除订单"}'
```

**第二步：创建角色并绑定权限**

```bash
# 创建"订单管理员"角色
curl -X POST http://localhost:8087/api/v1/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"订单管理员","description":"管理订单模块"}'

# 将 4 个订单权限分配给该角色 (假设权限 ID 为 3,4,5)
curl -X POST http://localhost:8087/api/v1/roles/2/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"permission_ids":[3,4,5]}'
```

**第三步：给用户分配角色**

```bash
curl -X POST http://localhost:8087/api/v1/users/2/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"role_ids":[2]}'
```

**第四步：业务代码中校验权限**

```python
# 订单删除接口
@app.route("/api/orders/<id>", methods=["DELETE"])
def delete_order(id):
    token = request.headers.get("Authorization", "").replace("Bearer ", "")
    # 调 RBAC 检查权限
    resp = requests.post(f"{RBAC_URL}/auth/check",
        json={"resource": "order", "action": "delete"},
        headers={"Authorization": f"Bearer {token}"})
    if not resp.json()["data"]["allowed"]:
        return {"code": 403, "message": "无权限删除订单"}, 403
    # 执行业务删除...
```

### 5.2 安全建议

- **JWT Secret**: 生产环境务必修改 `config.yaml` 中的 `jwt.secret`
- **HTTPS**: 生产环境应使用 HTTPS 或在反向代理层启用 TLS
- **CORS**: 生产环境将 `cors.allow_origins` 改为具体域名，如 `["https://myapp.com"]`
- **数据库密码**: 使用环境变量覆盖 `db.password`，避免明文写入配置文件
- **Gin Mode**: 生产环境设置 `server.mode: release` 关闭调试输出
- **Token 管理**: 前端应将 Token 存储在 httpOnly cookie 或安全存储中，避免 XSS 泄露
