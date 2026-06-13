# RBAC Java SDK

用于从 Java 业务系统集成 RBAC 权限管理微服务。兼容 Java 11+，**纯 JDK 实现，零外部依赖**。

## 安装

### Maven

```xml
<dependency>
    <groupId>com.rbac</groupId>
    <artifactId>rbac-client</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'com.rbac:rbac-client:1.0.0'
```

### 直接复制

只需将 `src/main/java/com/rbac/client/` 下的 6 个 `.java` 文件复制到项目中即可，无需任何外部依赖。

## 快速开始

```java
import com.rbac.client.RBACClient;
import com.rbac.client.RBACException;

public class Main {
    public static void main(String[] args) throws RBACException {
        RBACClient client = new RBACClient("http://localhost:8087/api/v1");

        // 1. 登录
        var loginResult = client.login("admin", "password");
        String token = loginResult.getToken();
        String refreshToken = loginResult.getRefreshToken();
        System.out.println("登录用户: " + loginResult.getUsername());

        // 2. 检查权限
        boolean canDelete = client.checkPermission(token, "user", "delete");
        System.out.println("能否删除用户: " + canDelete);

        // 3. 批量检查
        var perms = client.batchCheck(token, List.of(
            new CheckItem("user", "read"),
            new CheckItem("user", "delete"),
            new CheckItem("role", "create")
        ));
        System.out.println("批量结果: " + perms); // {user:read=true, user:delete=false, role:create=true}

        // 4. Token 自省
        var info = client.introspect(token, "order", "read");
        if (info.isActive()) {
            System.out.println("Token 有效, 用户: " + info.getUsername());
        }

        // 5. 刷新 Token
        var refreshResult = client.refresh(refreshToken);
        token = refreshResult.getToken();
    }
}
```

## Spring Boot 集成 (注解+拦截器)

### 1. 注册拦截器

```java
import com.rbac.client.RBACClient;
import com.rbac.client.PermissionInterceptor;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.servlet.config.annotation.InterceptorRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

@Configuration
public class WebConfig implements WebMvcConfigurer {

    private final RBACClient rbacClient = new RBACClient("http://localhost:8087/api/v1");
    private final PermissionInterceptor permInterceptor = new PermissionInterceptor(rbacClient);

    @Override
    public void addInterceptors(InterceptorRegistry registry) {
        registry.addInterceptor(new HandlerInterceptor() {
            @Override
            public boolean preHandle(HttpServletRequest request,
                                     HttpServletResponse response,
                                     Object handler) {
                // 查找方法上的 @RequirePermission 注解
                if (handler instanceof HandlerMethod hm) {
                    RequirePermission ann = hm.getMethodAnnotation(RequirePermission.class);
                    if (ann != null) {
                        PermissionInterceptor.ErrorResult err =
                            permInterceptor.check(ann.resource(), ann.action(), request);
                        if (err != null) {
                            response.setStatus(err.httpStatus);
                            response.setContentType("application/json;charset=UTF-8");
                            response.getWriter().write(
                                String.format("{\"code\":%d,\"message\":\"%s\"}", err.code, err.message));
                            return false;
                        }
                    }
                }
                return true;
            }
        });
    }
}
```

### 2. 在 Controller 中使用

```java
import com.rbac.client.RequirePermission;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api")
public class ServerController {

    @GetMapping("/servers")
    @RequirePermission(resource = "server", action = "read")
    public List<Server> listServers() {
        // 有 server:read 权限才会执行到这里
        return serverService.findAll();
    }

    @PostMapping("/servers/{id}/restart")
    @RequirePermission(resource = "server", action = "restart")
    public void restartServer(@PathVariable int id) {
        serverService.restart(id);
    }

    @DeleteMapping("/servers/{id}")
    @RequirePermission(resource = "server", action = "delete")
    public void deleteServer(@PathVariable int id) {
        serverService.delete(id);
    }
}
```

## Servlet Filter 集成 (非 Spring)

```java
// web.xml 配置
// <filter>
//     <filter-name>permissionFilter</filter-name>
//     <filter-class>com.rbac.client.PermissionFilter</filter-class>
//     <init-param>
//         <param-name>rbacUrl</param-name>
//         <param-value>http://localhost:8087/api/v1</param-value>
//     </init-param>
//     <init-param>
//         <param-name>rules</param-name>
//         <param-value>
//             /api/servers/* = server:read
//             POST /api/servers/restart = server:restart
//             POST /api/deployments = deployment:execute
//         </param-value>
//     </init-param>
// </filter>
```

## 前端权限判断

```java
import com.rbac.client.PermissionUtil;
import java.util.Map;

// 从 RBAC 加载当前用户权限
Map<String, List<String>> perms = client.getMenu(token);

// 菜单显隐
if (PermissionUtil.hasAnyPerm(perms, "server")) {
    // 渲染"服务器管理"菜单项
}

// 按钮显隐
if (PermissionUtil.hasPerm(perms, "server", "restart")) {
    // 渲染"重启服务器"按钮
}
```

## API 对照

| RBACClient 方法 | HTTP 端点 | 说明 |
|---|---|---|
| `login(account, password)` | POST /auth/login | 登录，返回 Token 对 |
| `refresh(refreshToken)` | POST /auth/refresh | 刷新 Token |
| `verify(token)` | POST /auth/verify | 验证 Token |
| `checkPermission(token, res, act)` | POST /auth/check | 检查单个权限 |
| `batchCheck(token, items)` | POST /auth/batch-check | 批量检查 |
| `introspect(token, res, act)` | POST /auth/introspect | Token 自省 |
| `getMenu(token)` | GET /auth/menu | 获取全部权限 |

## 异常处理

所有 RBAC 错误通过 `RBACException` 抛出：

```java
try {
    client.login("admin", "wrong_password");
} catch (RBACException e) {
    if (e.isUnauthorized()) {
        // Token 无效或过期
    } else if (e.isForbidden()) {
        // 权限不足
    } else {
        System.out.println("错误码: " + e.getCode() + ", 消息: " + e.getMessage());
    }
}
```
