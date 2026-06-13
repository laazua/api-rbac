package com.rbac;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.*;
import java.util.stream.Collectors;

/**
 * RBAC 权限管理系统 Java SDK — Java 17+ 优化版。
 *
 * <p>利用现代 Java 特性:</p>
 * <ul>
 *   <li>{@link java.net.http.HttpClient} 原生异步 HTTP (Java 11+)</li>
 *   <li>{@code record} 类替代 POJO (Java 14+)</li>
 *   <li>Switch Expressions (Java 14+)</li>
 * </ul>
 *
 * <h3>快速开始</h3>
 * <pre>{@code
 * var client = new RBACClient("http://localhost:8087/api/v1");
 * var result = client.login("admin", "password");
 * if (client.checkPermission(result.token(), "user", "delete")) { ... }
 * }</pre>
 */
public class RBACClient {

    private final String baseUrl;
    private final HttpClient httpClient;

    public RBACClient(String baseUrl) {
        this(baseUrl, Duration.ofSeconds(10));
    }

    public RBACClient(String baseUrl, Duration timeout) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(timeout)
                .build();
    }

    // ================================================================
    // 公共 API
    // ================================================================

    /** 用户登录 */
    public LoginResult login(String account, String password) throws RBACException {
        var body = mapOf("account", account, "password", password);
        var resp = post("/auth/login", null, body);
        ensureCode(resp, 0);
        var d = resp.data();
        return new LoginResult(
                str(d, "token"),
                str(d, "refresh_token"),
                num(d, "expires_in"),
                num(d, "user_id"),
                str(d, "username"));
    }

    /** 刷新 Token */
    public RefreshResult refresh(String refreshToken) throws RBACException {
        var body = mapOf("refresh_token", refreshToken);
        var resp = post("/auth/refresh", null, body);
        ensureCode(resp, 0);
        var d = resp.data();
        return new RefreshResult(
                str(d, "token"),
                str(d, "refresh_token"),
                num(d, "expires_in"));
    }

    /** 验证 Token */
    public VerifyResult verify(String token) throws RBACException {
        var resp = postWithToken(token, "/auth/verify", null);
        ensureCode(resp, 0);
        var d = resp.data();
        return new VerifyResult(num(d, "user_id"), str(d, "username"));
    }

    /** 检查单个权限, 返回 true=有权限 false=无权限 */
    public boolean checkPermission(String token, String resource, String action) throws RBACException {
        var body = mapOf("resource", resource, "action", action);
        var resp = postWithToken(token, "/auth/check", body);
        if (resp.code() == 1003) return false;
        ensureCode(resp, 0);
        return "true".equals(resp.data().get("allowed"));
    }

    /** 批量检查权限 */
    public Map<String, Boolean> batchCheck(String token, List<CheckItem> permissions) throws RBACException {
        var items = permissions.stream()
                .map(p -> (Object) Map.of("resource", p.resource, "action", p.action))
                .collect(Collectors.toList());
        var body = Map.of("permissions", (Object) items);
        var resp = postWithToken(token, "/auth/batch-check", body);

        ensureCode(resp, 0);
        @SuppressWarnings("unchecked")
        var results = (Map<String, Object>) resp.data().get("results");
        var map = new LinkedHashMap<String, Boolean>();
        if (results != null) {
            results.forEach((k, v) -> map.put(k, "true".equals(String.valueOf(v))));
        }
        return map;
    }

    /** Token 自省 — 验证 Token + 可选权限检查 */
    public IntrospectResult introspect(String token, String resource, String action) throws RBACException {
        var body = new LinkedHashMap<String, Object>();
        body.put("token", token);
        if (resource != null && !resource.isEmpty()) body.put("resource", resource);
        if (action != null && !action.isEmpty()) body.put("action", action);

        var resp = post("/auth/introspect", null, body);
        var d = resp.data();
        return new IntrospectResult(
                "true".equals(d.get("active")),
                optNum(d, "user_id", 0L),
                optStr(d, "username", ""));
    }

    /** 获取用户全部权限 map[resource → [action...]] */
    @SuppressWarnings("unchecked")
    public Map<String, List<String>> getMenu(String token) throws RBACException {
        var request = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + "/auth/menu"))
                .header("Authorization", "Bearer " + token)
                .timeout(Duration.ofSeconds(10))
                .GET().build();

        var resp = send(request);
        ensureCode(resp, 0);

        var perms = (Map<String, Object>) resp.data().get("permissions");
        var map = new LinkedHashMap<String, List<String>>();
        if (perms != null) {
            perms.forEach((k, v) -> map.put(k, (List<String>) v));
        }
        return map;
    }

    // ================================================================
    // HTTP 底层
    // ================================================================

    private ApiResponse postWithToken(String token, String path, Map<String, Object> body) throws RBACException {
        return post(path, Map.of("Authorization", "Bearer " + token), body);
    }

    @SuppressWarnings("unchecked")
    private ApiResponse post(String path, Map<String, String> headers,
                              Map<String, Object> body) throws RBACException {
        var jsonBody = JsonUtil.toJson(body == null ? Map.of() : body);

        var builder = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .header("Content-Type", "application/json")
                .timeout(Duration.ofSeconds(10));

        if (headers != null) headers.forEach(builder::header);

        var request = builder.POST(HttpRequest.BodyPublishers.ofString(jsonBody)).build();
        return send(request);
    }

    @SuppressWarnings("unchecked")
    private ApiResponse send(HttpRequest request) throws RBACException {
        try {
            var response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            var body = response.body();
            if (body == null || body.isEmpty()) {
                throw new RBACException(-1, "RBAC 服务返回空响应 (HTTP " + response.statusCode() + ")");
            }
            var map = (Map<String, Object>) JsonUtil.parse(body);
            int code = ((Number) map.getOrDefault("code", -1)).intValue();
            String msg = (String) map.getOrDefault("message", "");
            var data = (Map<String, Object>) map.get("data");
            return new ApiResponse(code, msg, data != null ? data : Map.of());
        } catch (IOException e) {
            throw new RBACException(-1, "无法连接 RBAC 服务: " + e.getMessage());
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RBACException(-1, "请求被中断");
        }
    }

    private static void ensureCode(ApiResponse resp, int expected) throws RBACException {
        if (resp.code() != expected) {
            throw new RBACException(resp.code(), resp.message());
        }
    }

    // ================================================================
    // Record 类型
    // ================================================================

    public record LoginResult(String token, String refreshToken, long expiresIn, long userId, String username) {}
    public record RefreshResult(String token, String refreshToken, long expiresIn) {}
    public record VerifyResult(long userId, String username) {}
    public record IntrospectResult(boolean active, long userId, String username) {}
    public record CheckItem(String resource, String action) {}

    // ================================================================
    // 内部工具
    // ================================================================

    /** API 响应 (非 record — 需要自定义 data() 返回类型) */
    static class ApiResponse {
        private final int code;
        private final String message;
        private final Map<String, Object> data;

        ApiResponse(int code, String message, Map<String, Object> data) {
            this.code = code;
            this.message = message;
            this.data = data;
        }

        int code() { return code; }
        String message() { return message; }
        Map<String, Object> data() { return data; }
    }

    /** 构造 Map<String, Object> 避免 Map.of 类型推断问题 */
    private static Map<String, Object> mapOf(String k1, Object v1) {
        var m = new LinkedHashMap<String, Object>();
        m.put(k1, v1);
        return m;
    }

    private static Map<String, Object> mapOf(String k1, Object v1, String k2, Object v2) {
        var m = new LinkedHashMap<String, Object>();
        m.put(k1, v1);
        m.put(k2, v2);
        return m;
    }

    private static String str(Map<String, Object> m, String key) {
        var v = m.get(key);
        return v != null ? v.toString() : "";
    }

    private static long num(Map<String, Object> m, String key) {
        var v = m.get(key);
        if (v instanceof Number n) return n.longValue();
        if (v instanceof String s) return Long.parseLong(s);
        return 0L;
    }

    private static long optNum(Map<String, Object> m, String key, long def) {
        var v = m.get(key);
        if (v instanceof Number n) return n.longValue();
        if (v instanceof String s) {
            try { return Long.parseLong(s); } catch (NumberFormatException e) { return def; }
        }
        return def;
    }

    private static String optStr(Map<String, Object> m, String key, String def) {
        var v = m.get(key);
        return v instanceof String s ? s : (v != null ? v.toString() : def);
    }
}
