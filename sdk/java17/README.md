# RBAC Java SDK (Java 17+)

Java 17+ 优化版 SDK，利用现代 Java 特性：

| 特性 | Java 8 版 | **Java 17+ 版** |
|------|-----------|-----------------|
| HTTP 客户端 | `HttpURLConnection` | **`java.net.http.HttpClient`** (native async) |
| 结果类型 | POJO (getter/setter) | **`record`** (不可变, 自动 equals/hashCode) |
| Servlet API | `javax.servlet` | **`jakarta.servlet`** (Jakarta EE 9+) |
| 字符串模板 | `String.format()` | **`STR."\{}"`** (Java 21+) |
| Switch | `if/else` | **Switch Expressions** |
| 虚拟线程 | — | **Virtual Threads** 支持 (Java 21+) |
| 模块系统 | — | **JPMS** `module-info.java` |
| 零依赖 | ✅ | ✅ |

## 版本选择

| 你的项目环境 | 推荐 SDK |
|-------------|---------|
| Java 8 ~ 16, Spring Boot 2.x, javax | [`sdk/java`](../java/) |
| **Java 17+, Spring Boot 3.x, Jakarta EE** | **`sdk/java17/`** |
| Java 21 LTS, 虚拟线程 | **`sdk/java17/`** |

## 快速开始

```java
import com.rbac.client.RBACClient;

var client = new RBACClient("http://localhost:8087/api/v1");

// 1. 登录 — 返回 record
var login = client.login("admin", "password");
// login.token(), login.refreshToken(), login.expiresIn(), login.userId(), login.username()

// 2. 检查权限
if (client.checkPermission(login.token(), "server", "restart")) {
    System.out.println("有权限重启服务器");
}

// 3. 批量检查
var results = client.batchCheck(login.token(), List.of(
    new CheckItem("user", "read"),
    new CheckItem("user", "delete")
));

// 4. Token 自省
var info = client.introspect(login.token(), "order", "read");
// info.active(), info.userId(), info.username()

// 5. 获取全部权限 + 前端判断
var perms = client.getMenu(login.token());
if (PermissionUtil.hasPerm(perms, "server", "restart")) { ... }
if (PermissionUtil.hasAnyPerm(perms, "server")) { ... }
```

## record 类型的优势

```java
// Java 8 版 — POJO
LoginResult login = client.login("admin", "123456");
String token = login.getToken();   // getter

// Java 17+ 版 — record
LoginResult login = client.login("admin", "123456");
String token = login.token();      // 直接访问

// record 自动提供:
// - 全参数构造器 (编译时检查)
// - equals / hashCode / toString
// - 不可变 (线程安全)
// - 解构模式匹配 (Java 21+):
if (login instanceof LoginResult(var t, var rt, var exp, var uid, var un)) {
    System.out.println(STR."用户 \{un} 登录成功, Token \{t}");
}
```

## Spring Boot 3.x 集成

```java
@RestController
@RequestMapping("/api")
public class ServerController {

    // 使用 record 作为响应体
    record ServerInfo(long id, String name, String ip, String status) {}

    @GetMapping("/servers")
    @RequirePermission(resource = "server", action = "read")
    public List<ServerInfo> listServers() {
        return serverService.findAll().stream()
            .map(s -> new ServerInfo(s.id(), s.name(), s.ip(), s.status()))
            .toList();
    }

    @PostMapping("/servers/{id}/restart")
    @RequirePermission(resource = "server", action = "restart")
    public void restart(@PathVariable long id) {
        serverService.restart(id);
    }
}
```

## 虚拟线程 (Java 21+)

```java
// 对于高并发场景，可以使用虚拟线程发送 HTTP 请求
var client = new RBACClient("http://localhost:8087/api/v1");

try (var executor = Executors.newVirtualThreadPerTaskExecutor()) {
    var futures = permissions.stream()
        .map(p -> executor.submit(() -> client.checkPermission(token, p.resource(), p.action())))
        .toList();

    for (var f : futures) {
        System.out.println("Result: " + f.get());
    }
}
```

## API 对照

| RBACClient 方法 | API 端点 | 返回类型 |
|---|---|---|
| `login(account, password)` | POST /auth/login | `LoginResult` (record) |
| `refresh(refreshToken)` | POST /auth/refresh | `RefreshResult` (record) |
| `verify(token)` | POST /auth/verify | `VerifyResult` (record) |
| `checkPermission(token, res, act)` | POST /auth/check | `boolean` |
| `batchCheck(token, items)` | POST /auth/batch-check | `Map<String,Boolean>` |
| `introspect(token, res, act)` | POST /auth/introspect | `IntrospectResult` (record) |
| `getMenu(token)` | GET /auth/menu | `Map<String,List<String>>` |

## 迁移指南: Java 8 → 17+

```java
// Java 8                             // Java 17+
RBACClient client = new RBACClient();  var client = new RBACClient();

LoginResult r = client.login(a,p);     var r = client.login(a, p);
String t = r.getToken();               String t = r.token();

Map<String,Boolean> m =                var m = client.batchCheck(
  client.batchCheck(t, items);             t, items);

// 注解                              // 注解 (相同)
@RequirePermission(                   @RequirePermission(
  resource="server",                    resource="server",
  action="restart"                      action="restart"
)                                     )

// javax.servlet                      // jakarta.servlet
import javax.servlet...               import jakarta.servlet...
```
