package com.rbac.client;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

/**
 * Spring Boot / Spring MVC 权限校验拦截器（Java 8+ javax.servlet）。
 *
 * <p>配合 {@link RequirePermission} 注解使用。</p>
 *
 * <h3>Spring Boot 集成示例</h3>
 * <pre>{@code
 * @Configuration
 * public class WebConfig implements WebMvcConfigurer {
 *     private final RBACClient rbacClient = new RBACClient("http://localhost:8087/api/v1");
 *     private final PermissionInterceptor permInterceptor = new PermissionInterceptor(rbacClient);
 *
 *     @Override
 *     public void addInterceptors(InterceptorRegistry registry) {
 *         registry.addInterceptor(new HandlerInterceptor() {
 *             public boolean preHandle(HttpServletRequest req, HttpServletResponse res, Object handler) {
 *                 if (handler instanceof HandlerMethod) {
 *                     RequirePermission ann = ((HandlerMethod) handler).getMethodAnnotation(RequirePermission.class);
 *                     if (ann != null) {
 *                         ErrorResult err = permInterceptor.check(ann.resource(), ann.action(), req);
 *                         if (err != null) {
 *                             res.setStatus(err.httpStatus);
 *                             res.setContentType("application/json;charset=UTF-8");
 *                             res.getWriter().write(String.format("{\"code\":%d,\"message\":\"%s\"}", err.code, err.message));
 *                             return false;
 *                         }
 *                     }
 *                 }
 *                 return true;
 *             }
 *         });
 *     }
 * }
 *
 * // Controller 中使用
 * @RestController
 * public class ServerController {
 *     @GetMapping("/api/servers")
 *     @RequirePermission(resource = "server", action = "read")
 *     public List<Server> listServers() { ... }
 * }
 * }</pre>
 */
public class PermissionInterceptor {

    private final RBACClient client;

    public PermissionInterceptor(RBACClient client) {
        this.client = client;
    }

    /**
     * 执行权限检查。
     *
     * @return null 表示通过，非 null 表示错误
     */
    public ErrorResult check(String resource, String action, HttpServletRequest request) {
        String authHeader = request.getHeader("Authorization");
        if (authHeader == null || authHeader.isEmpty()) {
            return new ErrorResult(HttpServletResponse.SC_UNAUTHORIZED, 1002, "未提供认证Token");
        }

        String token = authHeader.replace("Bearer ", "").trim();

        try {
            if (!client.checkPermission(token, resource, action)) {
                return new ErrorResult(HttpServletResponse.SC_FORBIDDEN, 1003,
                        "无权限: " + resource + ":" + action);
            }
            return null;
        } catch (RBACException e) {
            if (e.isUnauthorized()) {
                return new ErrorResult(HttpServletResponse.SC_UNAUTHORIZED, e.getCode(), e.getMessage());
            }
            return new ErrorResult(HttpServletResponse.SC_BAD_GATEWAY, 1005,
                    "权限服务异常: " + e.getMessage());
        }
    }

    public static class ErrorResult {
        public final int httpStatus;
        public final int code;
        public final String message;

        public ErrorResult(int httpStatus, int code, String message) {
            this.httpStatus = httpStatus;
            this.code = code;
            this.message = message;
        }
    }
}
