# RBAC Node.js SDK

用于从 Node.js 业务系统集成 RBAC 权限管理微服务。

## 安装

```bash
npm install rbac-client
# 或直接复制 src/index.js 到项目中
```

## 快速开始

```js
const { RBACClient, permissionGuard } = require('rbac-client');

const client = new RBACClient('http://localhost:8087/api/v1');

// 1. 登录
const result = await client.login('admin', 'password');
const token = result.token;
const refreshToken = result.refresh_token;

// 2. 验证 Token
const user = await client.verify(token);
console.log(`当前用户: ${user.username}`);

// 3. 检查权限
if (await client.checkPermission(token, 'user', 'delete')) {
  console.log('有删除用户的权限');
} else {
  console.log('无权限');
}

// 4. 批量检查
const perms = await client.batchCheck(token, [
  ['user', 'read'],
  ['user', 'delete'],
  ['role', 'create'],
]);
console.log(perms); // {"user:read": true, "user:delete": false, "role:create": true}

// 5. Token 自省（外部服务使用）
const info = await client.introspect(token, 'order', 'read');
if (info.active) {
  console.log(`Token 有效，用户: ${info.username}`);
}

// 6. 刷新 Token
const newTokens = await client.refresh(refreshToken);
token = newTokens.token;
refreshToken = newTokens.refresh_token;
```

## 在 Express 中使用中间件

```js
const express = require('express');
const { RBACClient, permissionGuard } = require('rbac-client');

const app = express();
const rbac = new RBACClient('http://rbac-service:8087/api/v1');

// 使用中间件保护路由
app.delete('/orders/:id', permissionGuard(rbac, 'order', 'delete'), (req, res) => {
  // 业务逻辑...
  res.json({ success: true });
});

app.listen(3000);
```

## API 文档

### new RBACClient(baseUrl, timeout?)

| 方法 | 说明 |
|------|------|
| `login(account, password)` | 登录，返回 Token 对 |
| `refresh(refreshToken)` | 刷新 Token |
| `verify(token)` | 验证 Token 有效性 |
| `checkPermission(token, resource, action)` | 检查单个权限 |
| `batchCheck(token, permissions)` | 批量检查权限 |
| `introspect(token, resource?, action?)` | Token 自省 |
| `getMenu(token)` | 获取用户权限菜单 |

### permissionGuard(client, resource, action)

Express/Connect 中间件，自动从 `Authorization` 头部提取 Token 并校验权限。
