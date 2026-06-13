package com.ops.config;

import com.rbac.RBACClient;
import com.rbac.RequirePermission;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.servlet.HandlerInterceptor;
import org.springframework.web.servlet.config.annotation.InterceptorRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

import java.io.IOException;

/**
 * Spring MVC 配置 — 注册 @RequirePermission 注解拦截器。
 *
 * <p>原理: 每个请求到达 Controller 前, 拦截器检查方法上的 @RequirePermission 注解,
 * 提取 Authorization 头部中的 JWT Token, 调用 api-rbac 校验权限。</p>
 */
@Configuration
public class WebConfig implements WebMvcConfigurer {

    private final RBACClient rbacClient;

    public WebConfig(RBACClient rbacClient) {
        this.rbacClient = rbacClient;
    }

    @Override
    public void addInterceptors(InterceptorRegistry registry) {
        registry.addInterceptor(new PermissionCheckInterceptor(rbacClient))
                .order(1);
    }

    /**
     * 权限校验拦截器 — 整个系统只有这一个地方做权限校验。
     * 业务 Controller 只需加 @RequirePermission 注解即可。
     */
    static class PermissionCheckInterceptor implements HandlerInterceptor {

        private final RBACClient client;

        PermissionCheckInterceptor(RBACClient client) {
            this.client = client;
        }

        @Override
        public boolean preHandle(HttpServletRequest request, HttpServletResponse response,
                                 Object handler) throws IOException {
            // 只处理 Controller 方法
            if (!(handler instanceof HandlerMethod hm)) return true;

            // 查找 @RequirePermission 注解
            var ann = hm.getMethodAnnotation(RequirePermission.class);
            if (ann == null) return true; // 无注解 → 放行

            // 提取 Token
            String authHeader = request.getHeader("Authorization");
            if (authHeader == null || authHeader.isEmpty()) {
                sendError(response, 401, 1002, "未提供认证Token");
                return false;
            }

            String token = authHeader.replace("Bearer ", "").trim();

            // 调用 api-rbac 校验
            try {
                if (!client.checkPermission(token, ann.resource(), ann.action())) {
                    sendError(response, 403, 1003,
                            "无权限: " + ann.resource() + ":" + ann.action());
                    return false;
                }
            } catch (Exception e) {
                sendError(response, 502, 1005, "权限服务异常: " + e.getMessage());
                return false;
            }

            return true; // 校验通过
        }

        private void sendError(HttpServletResponse resp, int httpStatus,
                               int code, String message) throws IOException {
            resp.setStatus(httpStatus);
            resp.setContentType("application/json;charset=UTF-8");
            resp.getWriter().write(
                    "{\"code\":" + code + ",\"message\":\"" + message + "\"}");
        }
    }
}
