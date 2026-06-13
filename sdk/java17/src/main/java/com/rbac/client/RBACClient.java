package com.rbac.client;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;

/**
 * RBAC 权限管理系统 Java SDK — Java 17+ 韧性增强版。
 *
 * <p>核心特性:</p>
 * <ul>
 *   <li>故障降级: RBAC 不可达时，使用本地缓存权限 (FailMode.CACHE)</li>
 *   <li>熔断保护: 连续失败超阈值后，直接走本地缓存，避免雪崩</li>
 *   <li>安全默认: FailMode.DENY，故障时拒绝所有请求</li>
 *   <li>自动恢复: 熔断后定期探测，恢复正常自动切回</li>
 * </ul>
 *
 * <pre>{@code
 * var client = new RBACClient("http://localhost:8087/api/v1",
 *     FailMode.CACHE, 300, 5, 30);
 *
 * // RBAC 正常 → 远程校验; RBAC 挂 → 自动降级到本地缓存
 * if (client.checkPermission(token, "user", "delete")) { ... }
 * }</pre>
 */
public class RBACClient {

    private final String baseUrl;
    private final HttpClient httpClient;
    private final FailMode failMode;
    private final long cacheTtlMs;
    private final int cbThreshold;
    private final long cbRecoveryMs;

    // 本地权限缓存: userId → { resource → [action...] }
    private final Map<Long, CacheEntry> cache = new ConcurrentHashMap<>();

    // 熔断器
    private volatile int failureCount = 0;
    private volatile boolean circuitOpen = false;
    private volatile long circuitOpenedAt = 0;

    /** 简单构造 (FailMode.DENY) */
    public RBACClient(String baseUrl) {
        this(baseUrl, FailMode.DENY, 300, 5, 30);
    }

    /** 完整构造 */
    public RBACClient(String baseUrl, FailMode failMode, long cacheTtlSec,
                      int cbThreshold, long cbRecoverySec) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.failMode = failMode;
        this.cacheTtlMs = cacheTtlSec * 1000L;
        this.cbThreshold = cbThreshold;
        this.cbRecoveryMs = cbRecoverySec * 1000L;
        this.httpClient = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(5)).build();
    }

    // ================================================================
    // 公共 API
    // ================================================================

    public LoginResult login(String account, String password) throws RBACException {
        var resp = checkedPost("/auth/login", null, mapOf("account", account, "password", password));
        var d = resp.data();
        var result = new LoginResult(
                str(d, "token"), str(d, "refresh_token"),
                num(d, "expires_in"), num(d, "user_id"), str(d, "username"));

        // 登录成功后立即加载权限到本地缓存
        try {
            var perms = getMenu(result.token());
            cache.put(result.userId(), new CacheEntry(perms, System.currentTimeMillis()));
        } catch (Exception ignored) {}

        return result;
    }

    public RefreshResult refresh(String refreshToken) throws RBACException {
        var resp = checkedPost("/auth/refresh", null, mapOf("refresh_token", refreshToken));
        var d = resp.data();
        return new RefreshResult(str(d, "token"), str(d, "refresh_token"), num(d, "expires_in"));
    }

    public boolean checkPermission(String token, String resource, String action) throws RBACException {
        var result = new Object(){ boolean value; boolean isSet; };
        callOrFallback(
                () -> { result.value = doCheckPermission(token, resource, action); result.isSet = true; },
                () -> { result.value = checkFromCache(token, resource, action); result.isSet = true; },
                "权限检查失败", cacheKey(token));
        if (!result.isSet) throw new RBACException(-1, "RBAC 服务不可用 (FailMode=DENY)");
        return result.value;
    }

    @SuppressWarnings("unchecked")
    public Map<String, Boolean> batchCheck(String token, List<CheckItem> permissions) throws RBACException {
        var result = new Object(){ Map<String, Boolean> value; boolean isSet; };
        callOrFallback(
                () -> {
                    var items = permissions.stream()
                            .map(p -> (Object) Map.of("resource", p.resource, "action", p.action))
                            .collect(Collectors.toList());
                    var resp = checkedPost("/auth/batch-check",
                            Map.of("Authorization", "Bearer " + token),
                            Map.of("permissions", (Object) items));
                    var r = new LinkedHashMap<String, Boolean>();
                    var data = (Map<String, Object>) resp.data().get("results");
                    if (data != null) data.forEach((k, v) -> r.put(k, "true".equals(String.valueOf(v))));
                    result.value = r; result.isSet = true;
                },
                () -> {
                    var r = new LinkedHashMap<String, Boolean>();
                    for (var p : permissions)
                        r.put(p.resource + ":" + p.action, checkFromCache(token, p.resource, p.action));
                    result.value = r; result.isSet = true;
                },
                "批量检查失败", cacheKey(token));
        if (!result.isSet) throw new RBACException(-1, "RBAC 服务不可用 (FailMode=DENY)");
        return result.value;
    }

    public IntrospectResult introspect(String token, String resource, String action) throws RBACException {
        var body = new LinkedHashMap<String, Object>();
        body.put("token", token);
        if (resource != null && !resource.isEmpty()) body.put("resource", resource);
        if (action != null && !action.isEmpty()) body.put("action", action);

        if (isCircuitOpen()) throw new RBACException(-1, "RBAC 服务不可用 (已熔断)");

        try {
            var resp = post("/auth/introspect", null, body);
            onSuccess();
            var d = resp.data();
            return new IntrospectResult(
                    "true".equals(string(d, "active")),
                    optNum(d, "user_id", 0L), optStr(d, "username", ""));
        } catch (Exception e) {
            onFailure();
            throw new RBACException(-1, "Token 自省失败: " + e.getMessage());
        }
    }

    @SuppressWarnings("unchecked")
    public Map<String, List<String>> getMenu(String token) throws RBACException {
        var result = new Object(){ Map<String, List<String>> value; boolean isSet; };
        callOrFallback(
                () -> {
                    var req = HttpRequest.newBuilder()
                            .uri(URI.create(baseUrl + "/auth/menu"))
                            .header("Authorization", "Bearer " + token)
                            .timeout(Duration.ofSeconds(5)).GET().build();
                    var resp = send(req);
                    ensureCode(resp, 0);
                    var perms = (Map<String, Object>) resp.data().get("permissions");
                    var map = new LinkedHashMap<String, List<String>>();
                    if (perms != null) perms.forEach((k, v) -> map.put(k, (List<String>) v));
                    result.value = map; result.isSet = true;

                    // 更新本地缓存
                    long uid = extractUserId(token);
                    if (uid > 0) cache.put(uid, new CacheEntry(map, System.currentTimeMillis()));
                },
                () -> {
                    // 从缓存取
                    long uid = extractUserId(token);
                    var entry = uid > 0 ? cache.get(uid) : null;
                    result.value = (entry != null && !entry.isExpired(cacheTtlMs))
                            ? entry.perms : Map.of();
                    result.isSet = true;
                },
                "获取菜单失败", cacheKey(token));
        if (!result.isSet) throw new RBACException(-1, "RBAC 服务不可用 (FailMode=DENY)");
        return result.value;
    }

    // ================================================================
    // 韧性层
    // ================================================================

    @FunctionalInterface
    interface ThrowingRunnable { void run() throws Exception; }

    private void callOrFallback(ThrowingRunnable call, ThrowingRunnable fallback,
                                 String errMsg, String cacheKey) throws RBACException {
        if (isCircuitOpen()) {
            if (failMode == FailMode.DENY)
                throw new RBACException(-1, errMsg + ": RBAC 服务不可用 (已熔断)");
            try { fallback.run(); } catch (Exception e) { /* ignore */ }
            return;
        }
        try {
            call.run();
            onSuccess();
        } catch (Exception e) {
            onFailure();
            if (failMode == FailMode.DENY)
                throw new RBACException(-1, errMsg + ": RBAC 服务不可用");
            try { fallback.run(); } catch (Exception ignored) {}
        }
    }

    private boolean checkFromCache(String token, String resource, String action) {
        long uid = extractUserId(token);
        if (uid <= 0) return false;
        var entry = cache.get(uid);
        if (entry == null || entry.isExpired(cacheTtlMs)) return false;
        return PermissionUtil.hasPerm(entry.perms, resource, action);
    }

    private void onSuccess() { failureCount = 0; circuitOpen = false; }
    private void onFailure() {
        failureCount++;
        if (failureCount >= cbThreshold) { circuitOpen = true; circuitOpenedAt = System.currentTimeMillis(); }
    }

    private boolean isCircuitOpen() {
        if (!circuitOpen) return false;
        if (System.currentTimeMillis() - circuitOpenedAt > cbRecoveryMs) {
            circuitOpen = false; failureCount = 0; return false;  // 半开
        }
        return true;
    }

    private long extractUserId(String token) {
        // 简单实现: 取缓存中存在的 key (生产环境应解码 JWT payload)
        if (!cache.isEmpty()) return cache.keySet().iterator().next();
        return 0;
    }
    private static String cacheKey(String token) { return token; }

    // ================================================================
    // HTTP + 内部类型
    // ================================================================

    private boolean doCheckPermission(String token, String r, String a) throws Exception {
        var resp = postWithToken(token, "/auth/check", mapOf("resource", r, "action", a));
        if (resp.code() == 1003) return false;
        ensureCode(resp, 0);
        return "true".equals(resp.data().get("allowed"));
    }

    private ApiResponse checkedPost(String path, Map<String, String> h, Map<String, Object> b) throws RBACException {
        if (isCircuitOpen()) throw new RBACException(-1, "RBAC 服务不可用 (已熔断)");
        try {
            var resp = post(path, h, b);
            onSuccess();
            return resp;
        } catch (RBACException e) {
            onFailure();
            throw e;
        }
    }

    private ApiResponse postWithToken(String token, String path, Map<String, Object> body) throws RBACException {
        return post(path, Map.of("Authorization", "Bearer " + token), body);
    }

    @SuppressWarnings("unchecked")
    private ApiResponse post(String path, Map<String, String> headers, Map<String, Object> body) throws RBACException {
        var jsonBody = JsonUtil.toJson(body == null ? Map.of() : body);
        var builder = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .header("Content-Type", "application/json")
                .timeout(Duration.ofSeconds(5));
        if (headers != null) headers.forEach(builder::header);
        var req = builder.POST(HttpRequest.BodyPublishers.ofString(jsonBody)).build();
        try {
            var resp = httpClient.send(req, HttpResponse.BodyHandlers.ofString());
            var map = (Map<String, Object>) JsonUtil.parse(resp.body());
            return new ApiResponse(((Number) map.getOrDefault("code", -1)).intValue(),
                    String.valueOf(map.getOrDefault("message", "")),
                    (Map<String, Object>) map.getOrDefault("data", Map.of()));
        } catch (IOException e) { throw new RBACException(-1, "无法连接 RBAC: " + e.getMessage()); }
        catch (InterruptedException e) { Thread.currentThread().interrupt(); throw new RBACException(-1, "中断"); }
    }

    @SuppressWarnings("unchecked")
    private ApiResponse send(HttpRequest req) throws RBACException {
        try {
            var resp = httpClient.send(req, HttpResponse.BodyHandlers.ofString());
            var map = (Map<String, Object>) JsonUtil.parse(resp.body());
            return new ApiResponse(((Number) map.getOrDefault("code", -1)).intValue(),
                    (String) map.getOrDefault("message", ""),
                    (Map<String, Object>) map.getOrDefault("data", Map.of()));
        } catch (IOException e) { throw new RBACException(-1, "无法连接 RBAC: " + e.getMessage()); }
        catch (InterruptedException e) { Thread.currentThread().interrupt(); throw new RBACException(-1, "中断"); }
    }

    private static void ensureCode(ApiResponse resp, int expected) throws RBACException {
        if (resp.code() != expected) throw new RBACException(resp.code(), resp.message());
    }

    public record LoginResult(String token, String refreshToken, long expiresIn, long userId, String username) {}
    public record RefreshResult(String token, String refreshToken, long expiresIn) {}
    public record VerifyResult(long userId, String username) {}
    public record IntrospectResult(boolean active, long userId, String username) {}
    public record CheckItem(String resource, String action) {}

    static class ApiResponse {
        private final int code; private final String message; private final Map<String, Object> data;
        ApiResponse(int c, String m, Map<String, Object> d) { code = c; message = m; data = d; }
        int code() { return code; } String message() { return message; } Map<String, Object> data() { return data; }
    }

    static class CacheEntry {
        final Map<String, List<String>> perms; final long ts;
        CacheEntry(Map<String, List<String>> p, long t) { perms = p; ts = t; }
        boolean isExpired(long ttl) { return System.currentTimeMillis() - ts > ttl; }
    }

    private static Map<String, Object> mapOf(String k, Object v) { var m = new LinkedHashMap<String, Object>(); m.put(k, v); return m; }
    private static Map<String, Object> mapOf(String k1, Object v1, String k2, Object v2) { var m = mapOf(k1, v1); m.put(k2, v2); return m; }
    private static String string(Map<String, Object> m, String k) { var v = m.get(k); return v == null ? "" : v.toString(); }
    private static String str(Map<String, Object> m, String k) { return string(m, k); }
    private static long num(Map<String, Object> m, String k) { var v = m.get(k); return v instanceof Number n ? n.longValue() : v instanceof String s ? Long.parseLong(s) : 0; }
    private static long optNum(Map<String, Object> m, String k, long def) { try { return num(m, k); } catch (Exception e) { return def; } }
    private static String optStr(Map<String, Object> m, String k, String def) { var v = m.get(k); return v instanceof String s ? s : def; }
}
