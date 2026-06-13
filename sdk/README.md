# RBAC SDK 多语言支持

本目录包含 RBAC 权限管理系统的多语言 SDK，供不同语言的业务系统集成使用。

## 可用 SDK

| 语言 | 目录 | 说明 |
|------|------|------|
| **Go** | `pkg/client/` (项目根目录) | 原生 Go SDK，包含 Gin 中间件 |
| **Python** | [`sdk/python/`](python/) | Python 3.8+ SDK |
| **Node.js** | [`sdk/nodejs/`](nodejs/) | Node.js 14+ SDK，包含 Express 中间件 |
| **Java** | [`sdk/java/`](java/) | Java 8+ SDK，javax.servlet + Spring Boot 2.x |
| **Java 17+** | [`sdk/java17/`](java17/) | Java 17+ 优化版，HttpClient + Record + Jakarta EE |

## SDK 功能矩阵

所有 SDK 提供一致的功能接口：

| 功能 | Go | Python | Node.js | Java |
|------|:--:|:------:|:-------:|:----:|
| 登录 (`login`) | ✅ | ✅ | ✅ | ✅ |
| 刷新 Token (`refresh`) | ✅ | ✅ | ✅ | ✅ |
| 验证 Token (`verify`) | ✅ | ✅ | ✅ | ✅ |
| 检查权限 (`checkPermission`) | ✅ | ✅ | ✅ | ✅ |
| 批量检查 (`batchCheck`) | ✅ | ✅ | ✅ | ✅ |
| Token 自省 (`introspect`) | ✅ | ✅ | ✅ | ✅ |
| 获取菜单 (`getMenu`) | ✅ | ✅ | ✅ | ✅ |
| Web 框架中间件 | Gin | Flask 示例 | Express | Spring Boot + Servlet |

## 集成步骤

1. **获取 SDK**：选择对应语言的 SDK 文件
2. **初始化客户端**：传入 RBAC 服务地址
3. **用户登录**：调用 `login` 获取 Token
4. **权限校验**：在业务接口中调用 `checkPermission` 或批量检查，或使用中间件自动校验
5. **Token 管理**：使用 `refresh` 刷新过期 Token
