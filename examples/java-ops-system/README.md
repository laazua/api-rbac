# Java 运维管理系统 — api-rbac 解耦集成完整示例

> 演示 Java Spring Boot 如何将 **api-rbac (Go)** 作为独立权限微服务集成，业务与权限**完全解耦**。

## 项目结构

```
java-ops-system/
├── pom.xml                                    # Maven (Spring Boot 3.3 + Java 17)
├── README.md
├── setup_permissions.sh                       # 一键初始化运维权限
└── src/main/
    ├── resources/application.yml              # 配置 (仅 RBAC 地址)
    └── java/com/
        ├── rbac/                              # RBAC SDK（内嵌，纯 JDK 零依赖）
        │   ├── RBACClient.java                # HTTP 客户端
        │   ├── RBACException.java             # 异常
        │   ├── JsonUtil.java                  # JSON 解析
        │   ├── PermissionUtil.java            # 权限辅助 (前端/后端通用)
        │   └── RequirePermission.java         # @注解
        └── ops/
            ├── Application.java               # Spring Boot 入口
            ├── config/
            │   ├── AppConfig.java             # RBAC 客户端 Bean 配置
            │   └── WebConfig.java             # 拦截器: 自动校验 @RequirePermission
            └── controller/
                ├── AuthController.java        # 登录/刷新/获取权限
                ├── ServerController.java      # 服务器管理
                ├── DeploymentController.java  # 发布管理
                └── AlertController.java       # 告警管理
```

## 核心设计

```
Controller 业务代码         →   拦截器 (WebConfig)    →   api-rbac (Go :8087)
─────────────────────          ──────────────────        ─────────────────────
                                                                              
@RequirePermission(           1. 提取 Authorization 头部                      
  resource="server",          2. 获取 JWT Token                               
  action="restart"            3. POST /auth/check ──────→ 查询权限 → true/false
)                             4. true → 放行, 执行业务                         
public Map restart() {        5. false → 返回 403                             
    // 纯业务逻辑                                                              
}                                                                            
```

**关键点**:
- 业务 Controller 中**没有任何权限判断代码**，只有 `@RequirePermission` 注解
- 所有鉴权逻辑集中在 `WebConfig.PermissionCheckInterceptor` 一个地方
- 权限数据、用户密码全部在 api-rbac 侧，业务系统不存储

## 运行步骤

### 前提

- Java 17+
- Maven 3.6+
- api-rbac 服务已启动在 `localhost:8087`

### 1. 初始化权限

```bash
cd examples/java-ops-system
chmod +x setup_permissions.sh
./setup_permissions.sh
```

### 2. 启动

```bash
mvn spring-boot:run
# 启动在 http://localhost:8080
```

### 3. 测试

```bash
# 登录
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account":"admin","password":"your_password"}'

# 提取 token
TOKEN="<token>"

# 查看服务器列表 (server:read)
curl http://localhost:8080/api/servers -H "Authorization: Bearer $TOKEN"

# 重启服务器 (server:restart)
curl -X POST http://localhost:8080/api/servers/1/restart -H "Authorization: Bearer $TOKEN"

# 无权限操作 → 403
curl -X POST http://localhost:8080/api/servers/1/stop -H "Authorization: Bearer $TOKEN"
# {"code":403,"message":"无权限: server:stop"}
```

## 代码导读

### RBAC 客户端 Bean 配置 (AppConfig.java)

```java
@Configuration
public class AppConfig {
    @Value("${ops.rbac.url}")              // ← application.yml 中配置
    private String rbacUrl;

    @Bean
    public RBACClient rbacClient() {
        return new RBACClient(rbacUrl);    // ← 全局单例
    }
}
```

### 权限拦截器 — 整个系统只有一个鉴权点 (WebConfig.java)

```java
// 拦截所有 Controller 请求, 自动校验 @RequirePermission 注解

if (!(handler instanceof HandlerMethod hm)) return true;   // 非 Controller → 放行

var ann = hm.getMethodAnnotation(RequirePermission.class); // 查找注解
if (ann == null) return true;                              // 无注解 → 放行

String token = request.getHeader("Authorization")          // 提取 Token
    .replace("Bearer ", "");

if (!client.checkPermission(token, ann.resource(), ann.action())) {
    sendError(response, 403, 1003, "无权限");              // 鉴权失败 → 403
    return false;
}
return true;                                               // 鉴权通过 → 执行业务
```

### 业务 Controller — 零权限代码 (ServerController.java)

```java
@RestController
@RequestMapping("/api/servers")
public class ServerController {

    @GetMapping
    @RequirePermission(resource = "server", action = "read")    // ← 仅此一行权限代码
    public Map<String, Object> list() {
        return Map.of("code", 0, "data", db.values());          // ← 纯业务逻辑
    }

    @PostMapping("/{id}/restart")
    @RequirePermission(resource = "server", action = "restart")  // ← 仅此一行
    public Map<String, Object> restart(@PathVariable long id) {
        var s = db.get(id);
        db.put(id, new Server(s.id, s.name, s.ip, "running"));
        return Map.of("code", 0, "message", "服务器 " + s.name + " 重启成功");
    }
}
```

### 前端权限判断 (PermissionUtil.java)

```java
// 获取用户全部权限
Map<String, List<String>> perms = rbacClient.getMenu(token);
// → {"server": ["read","restart","stop"], "deployment": ["read","execute"], ...}

// 菜单显隐
if (PermissionUtil.hasAnyPerm(perms, "server")) {
    // 渲染 "服务器管理" 菜单
}

// 按钮显隐
if (PermissionUtil.hasPerm(perms, "server", "restart")) {
    // 渲染 "重启" 按钮
}
```

## 与 Python 版对比

| | Python Flask 示例 | **Java Spring Boot 示例** |
|---|---|---|
| 鉴权方式 | `@require_permission` 装饰器 | `@RequirePermission` 注解 + 拦截器 |
| RBAC 客户端 | `rbac_client.py` | `RBACClient.java` (内嵌) |
| 前端 | static/index.html | 可复用 Python 版的 index.html |
| 权限判断 | `hasPerm()` / `hasAnyPerm()` | `PermissionUtil.hasPerm()` / `hasAnyPerm()` |

## 新增业务模块

假设新增 "日志管理":

**1. 在 api-rbac 创建权限**:
```bash
curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"查看日志","resource":"log","action":"read"}'
```

**2. 创建 Controller**:
```java
@RestController
@RequestMapping("/api/logs")
public class LogController {

    @GetMapping
    @RequirePermission(resource = "log", action = "read")   // ← 仅此一行
    public Map<String, Object> list() {
        return Map.of("code", 0, "data", Files.list(Path.of("/var/log")));
    }
}
```

完成。不需要修改任何权限配置、用户表、拦截器。
