package com.rbac.client;

import javax.servlet.*;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Servlet Filter — 通用权限校验过滤器（Java 8+ javax.servlet）。
 *
 * <p>根据 URL 路径匹配所需的权限，自动从 Authorization 头部提取 Token 并校验。</p>
 *
 * <h3>web.xml 配置示例</h3>
 * <pre>{@code
 * <filter>
 *     <filter-name>permissionFilter</filter-name>
 *     <filter-class>com.rbac.client.PermissionFilter</filter-class>
 *     <init-param>
 *         <param-name>rbacUrl</param-name>
 *         <param-value>http://localhost:8087/api/v1</param-value>
 *     </init-param>
 *     <init-param>
 *         <param-name>rules</param-name>
 *         <param-value>
 *             /api/servers/* = server:read
 *             /api/servers/restart = server:restart
 *             /api/deployments/execute = deployment:execute
 *         </param-value>
 *     </init-param>
 * </filter>
 * }</pre>
 */
public class PermissionFilter implements Filter {

    private RBACClient client;
    private Map<String, String> pathRules;

    @Override
    public void init(FilterConfig config) throws ServletException {
        String rbacUrl = config.getInitParameter("rbacUrl");
        if (rbacUrl == null || rbacUrl.isEmpty()) {
            throw new ServletException("PermissionFilter 需要 rbacUrl 初始化参数");
        }
        this.client = new RBACClient(rbacUrl);

        String rulesStr = config.getInitParameter("rules");
        if (rulesStr != null && !rulesStr.isEmpty()) {
            this.pathRules = new LinkedHashMap<String, String>();
            for (String line : rulesStr.split("\n")) {
                line = line.trim();
                if (line.isEmpty()) continue;
                String[] parts = line.split("=", 2);
                if (parts.length == 2) {
                    pathRules.put(parts[0].trim(), parts[1].trim());
                }
            }
        }
    }

    @Override
    public void doFilter(ServletRequest req, ServletResponse resp, FilterChain chain)
            throws IOException, ServletException {
        HttpServletRequest request = (HttpServletRequest) req;
        HttpServletResponse response = (HttpServletResponse) resp;

        String path = request.getRequestURI();
        String perm = matchPath(path);

        // 无匹配规则 → 放行
        if (perm == null) {
            chain.doFilter(req, resp);
            return;
        }

        String[] parts = perm.split(":", 2);
        String resource = parts[0];
        String action = parts.length > 1 ? parts[1] : "*";

        String authHeader = request.getHeader("Authorization");
        if (authHeader == null || authHeader.isEmpty()) {
            sendError(response, HttpServletResponse.SC_UNAUTHORIZED, 1002, "未提供认证Token");
            return;
        }

        String token = authHeader.replace("Bearer ", "").trim();

        try {
            if (!client.checkPermission(token, resource, action)) {
                sendError(response, HttpServletResponse.SC_FORBIDDEN, 1003, "无权限: " + perm);
                return;
            }
            chain.doFilter(req, resp);
        } catch (RBACException e) {
            sendError(response, HttpServletResponse.SC_BAD_GATEWAY, 1005,
                    "权限服务异常: " + e.getMessage());
        }
    }

    @Override
    public void destroy() {}

    private String matchPath(String path) {
        if (pathRules == null) return null;
        for (Map.Entry<String, String> entry : pathRules.entrySet()) {
            String pattern = entry.getKey();
            if (pattern.endsWith("*") && path.startsWith(pattern.substring(0, pattern.length() - 1))) {
                return entry.getValue();
            }
            if (pattern.equals(path)) {
                return entry.getValue();
            }
        }
        return null;
    }

    private void sendError(HttpServletResponse resp, int httpStatus, int code, String message) throws IOException {
        resp.setStatus(httpStatus);
        resp.setContentType("application/json;charset=UTF-8");
        resp.getWriter().write(String.format("{\"code\":%d,\"message\":\"%s\"}", code, message));
    }
}
