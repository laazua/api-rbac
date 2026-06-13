package com.rbac.client;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

/**
 * Spring Boot 6 / Jakarta EE 权限校验拦截器。
 *
 * <p>适用于 Spring Boot 3.x+ (jakarta.servlet) 项目。</p>
 *
 * <h3>Spring Boot 集成</h3>
 * <pre>{@code
 * @Configuration
 * public class WebConfig implements WebMvcConfigurer {
 *     final RBACClient rbac = new RBACClient("http://localhost:8087/api/v1");
 *     final PermissionInterceptor perm = new PermissionInterceptor(rbac);
 *
 *     public void addInterceptors(InterceptorRegistry r) {
 *         r.addInterceptor(new HandlerInterceptor() {
 *             public boolean preHandle(HttpServletRequest req, HttpServletResponse res, Object h) {
 *                 if (h instanceof HandlerMethod hm) {
 *                     var ann = hm.getMethodAnnotation(RequirePermission.class);
 *                     if (ann != null) {
 *                         var err = perm.check(ann.resource(), ann.action(), req);
 *                         if (err != null) {
 *                             res.setStatus(err.httpStatus);
 *                             res.setContentType("application/json;charset=UTF-8");
 *                             res.getWriter().write(
 *                                 STR."{\"code\":\{err.code},\"message\":\"\{err.message}\"}");
 *                             return false;
 *                         }
 *                     }
 *                 }
 *                 return true;
 *             }
 *         });
 *     }
 * }
 * }</pre>
 */
public class PermissionInterceptor {

    private final RBACClient client;

    public PermissionInterceptor(RBACClient client) {
        this.client = client;
    }

    /**
     * @return null = 通过, non-null = 错误
     */
    public ErrorResult check(String resource, String action, HttpServletRequest request) {
        var authHeader = request.getHeader("Authorization");
        if (authHeader == null || authHeader.isEmpty()) {
            return new ErrorResult(401, 1002, "未提供认证Token");
        }
        var token = authHeader.replace("Bearer ", "").trim();

        try {
            if (!client.checkPermission(token, resource, action)) {
                return new ErrorResult(403, 1003, STR."无权限: \{resource}:\{action}");
            }
            return null;
        } catch (RBACException e) {
            var status = e.isUnauthorized() ? 401 : 502;
            return new ErrorResult(status, e.code(), STR."权限服务异常: \{e.getMessage()}");
        }
    }

    /** 支持 pattern matching for instanceof 解构 (Java 21+) */
    public record ErrorResult(int httpStatus, int code, String message) {}
}
