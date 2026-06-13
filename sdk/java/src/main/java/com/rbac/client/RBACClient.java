package com.rbac.client;

import java.io.*;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.*;

/**
 * RBAC 权限管理系统 Java SDK — HTTP 客户端。
 *
 * <p>供 Java 业务系统集成 api-rbac 微服务。兼容 Java 8+，纯 JDK 实现，零外部依赖。</p>
 *
 * <h3>快速开始</h3>
 * <pre>{@code
 * RBACClient client = new RBACClient("http://localhost:8087/api/v1");
 *
 * // 1. 登录
 * LoginResult result = client.login("admin", "password");
 * String token = result.getToken();
 *
 * // 2. 检查权限
 * boolean allowed = client.checkPermission(token, "user", "delete");
 *
 * // 3. 批量检查
 * List<CheckItem> items = Arrays.asList(
 *     new CheckItem("user", "read"), new CheckItem("user", "delete"));
 * Map<String, Boolean> perms = client.batchCheck(token, items);
 *
 * // 4. Token 自省
 * IntrospectResult info = client.introspect(token, "order", "read");
 * }</pre>
 */
public class RBACClient {

    private final String baseUrl;
    private final int timeout;

    /**
     * @param baseUrl RBAC 服务地址，如 "http://localhost:8087/api/v1"
     */
    public RBACClient(String baseUrl) {
        this(baseUrl, 10000);
    }

    /**
     * @param baseUrl     RBAC 服务地址
     * @param timeoutMillis 请求超时毫秒数
     */
    public RBACClient(String baseUrl, int timeoutMillis) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.timeout = timeoutMillis;
    }

    // ================================================================
    // 公共 API
    // ================================================================

    /**
     * 用户登录，返回 Token 对。
     */
    public LoginResult login(String account, String password) throws RBACException {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("account", account);
        body.put("password", password);

        JsonNode resp = post("/auth/login", null, body);
        int code = resp.getInt("code");
        if (code != 0) throw new RBACException(code, resp.getString("message"));

        JsonNode data = resp.getObject("data");
        return new LoginResult(
                data.getString("token"),
                data.getString("refresh_token"),
                data.getLong("expires_in"),
                data.getLong("user_id"),
                data.getString("username")
        );
    }

    /** 刷新 Token */
    public RefreshResult refresh(String refreshToken) throws RBACException {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("refresh_token", refreshToken);

        JsonNode resp = post("/auth/refresh", null, body);
        int code = resp.getInt("code");
        if (code != 0) throw new RBACException(code, resp.getString("message"));

        JsonNode data = resp.getObject("data");
        return new RefreshResult(
                data.getString("token"),
                data.getString("refresh_token"),
                data.getLong("expires_in")
        );
    }

    /** 验证 Token */
    public VerifyResult verify(String token) throws RBACException {
        JsonNode resp = postWithToken(token, "/auth/verify", null);
        int code = resp.getInt("code");
        if (code != 0) throw new RBACException(code, resp.getString("message"));

        JsonNode data = resp.getObject("data");
        return new VerifyResult(data.getLong("user_id"), data.getString("username"));
    }

    /** 检查单个权限 */
    public boolean checkPermission(String token, String resource, String action) throws RBACException {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("resource", resource);
        body.put("action", action);

        JsonNode resp = postWithToken(token, "/auth/check", body);
        int code = resp.getInt("code");
        if (code == 1003) return false;
        if (code != 0) throw new RBACException(code, resp.getString("message"));
        return resp.getObject("data").getBoolean("allowed");
    }

    /** 批量检查权限 */
    public Map<String, Boolean> batchCheck(String token, List<CheckItem> permissions) throws RBACException {
        List<Map<String, String>> items = new ArrayList<>();
        for (CheckItem item : permissions) {
            Map<String, String> m = new LinkedHashMap<>();
            m.put("resource", item.resource);
            m.put("action", item.action);
            items.add(m);
        }
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("permissions", items);

        JsonNode resp = postWithToken(token, "/auth/batch-check", body);
        int code = resp.getInt("code");
        if (code != 0) throw new RBACException(code, resp.getString("message"));

        JsonNode results = resp.getObject("data").getObject("results");
        Map<String, Boolean> map = new LinkedHashMap<>();
        for (String key : results.keys()) {
            map.put(key, results.getBoolean(key));
        }
        return map;
    }

    /** Token 自省 */
    public IntrospectResult introspect(String token, String resource, String action) throws RBACException {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("token", token);
        if (resource != null && !resource.isEmpty()) body.put("resource", resource);
        if (action != null && !action.isEmpty()) body.put("action", action);

        JsonNode resp = post("/auth/introspect", null, body);
        JsonNode data = resp.getObject("data");
        return new IntrospectResult(
                data.getBoolean("active"),
                data.optLong("user_id", 0),
                data.optString("username", "")
        );
    }

    /** 获取用户全部权限 */
    public Map<String, List<String>> getMenu(String token) throws RBACException {
        String url = this.baseUrl + "/auth/menu";
        HttpURLConnection conn = null;
        try {
            conn = openConnection(url, "GET");
            conn.setRequestProperty("Authorization", "Bearer " + token);

            int status = conn.getResponseCode();
            InputStream is = (status >= 200 && status < 300) ? conn.getInputStream() : conn.getErrorStream();
            String bodyStr = readAll(is);

            JsonNode resp = JsonNode.parse(bodyStr);
            int code = resp.getInt("code");
            if (code != 0) throw new RBACException(code, resp.getString("message"));

            JsonNode perms = resp.getObject("data").getObject("permissions");
            Map<String, List<String>> map = new LinkedHashMap<>();
            for (String key : perms.keys()) {
                List<String> actions = new ArrayList<>();
                for (JsonNode a : perms.getArray(key)) {
                    actions.add(a.asString());
                }
                map.put(key, actions);
            }
            return map;
        } catch (IOException e) {
            throw new RBACException(-1, "无法连接 RBAC 服务: " + e.getMessage());
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    // ================================================================
    // HTTP 底层 (Java 8 HttpURLConnection)
    // ================================================================

    private JsonNode postWithToken(String token, String path, Map<String, Object> body) throws RBACException {
        Map<String, String> headers = new HashMap<>();
        headers.put("Authorization", "Bearer " + token);
        return post(path, headers, body);
    }

    private JsonNode post(String path, Map<String, String> headers, Map<String, Object> body) throws RBACException {
        String url = this.baseUrl + path;
        String jsonBody = body != null ? JsonNode.toJson(body) : "";

        HttpURLConnection conn = null;
        try {
            conn = openConnection(url, "POST");
            conn.setRequestProperty("Content-Type", "application/json");
            if (headers != null) {
                for (Map.Entry<String, String> e : headers.entrySet()) {
                    conn.setRequestProperty(e.getKey(), e.getValue());
                }
            }
            conn.setDoOutput(true);

            if (body != null) {
                OutputStream os = conn.getOutputStream();
                os.write(jsonBody.getBytes(StandardCharsets.UTF_8));
                os.flush();
                os.close();
            }

            int status = conn.getResponseCode();
            InputStream is = (status >= 200 && status < 300) ? conn.getInputStream() : conn.getErrorStream();
            String responseBody = readAll(is);

            if (responseBody == null || responseBody.isEmpty()) {
                throw new RBACException(-1, "Empty response from RBAC server (HTTP " + status + ")");
            }
            return JsonNode.parse(responseBody);
        } catch (IOException e) {
            throw new RBACException(-1, "无法连接 RBAC 服务: " + e.getMessage());
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    private HttpURLConnection openConnection(String url, String method) throws IOException {
        URL u = new URL(url);
        HttpURLConnection conn = (HttpURLConnection) u.openConnection();
        conn.setRequestMethod(method);
        conn.setConnectTimeout(timeout);
        conn.setReadTimeout(timeout);
        conn.setDoInput(true);
        return conn;
    }

    private static String readAll(InputStream is) throws IOException {
        if (is == null) return null;
        StringBuilder sb = new StringBuilder();
        try (BufferedReader reader = new BufferedReader(new InputStreamReader(is, StandardCharsets.UTF_8))) {
            String line;
            while ((line = reader.readLine()) != null) {
                sb.append(line);
            }
        }
        return sb.toString();
    }

    // ================================================================
    // 结果类
    // ================================================================

    public static class LoginResult {
        private final String token, refreshToken, username;
        private final long expiresIn, userId;
        public LoginResult(String t, String rt, long exp, long uid, String un) {
            this.token = t; this.refreshToken = rt; this.expiresIn = exp; this.userId = uid; this.username = un;
        }
        public String getToken() { return token; }
        public String getRefreshToken() { return refreshToken; }
        public long getExpiresIn() { return expiresIn; }
        public long getUserId() { return userId; }
        public String getUsername() { return username; }
    }

    public static class RefreshResult {
        private final String token, refreshToken;
        private final long expiresIn;
        public RefreshResult(String t, String rt, long exp) { this.token = t; this.refreshToken = rt; this.expiresIn = exp; }
        public String getToken() { return token; }
        public String getRefreshToken() { return refreshToken; }
        public long getExpiresIn() { return expiresIn; }
    }

    public static class VerifyResult {
        private final long userId;
        private final String username;
        public VerifyResult(long uid, String un) { this.userId = uid; this.username = un; }
        public long getUserId() { return userId; }
        public String getUsername() { return username; }
    }

    public static class IntrospectResult {
        private final boolean active;
        private final long userId;
        private final String username;
        public IntrospectResult(boolean a, long uid, String un) { this.active = a; this.userId = uid; this.username = un; }
        public boolean isActive() { return active; }
        public long getUserId() { return userId; }
        public String getUsername() { return username; }
    }

    public static class CheckItem {
        public final String resource;
        public final String action;
        public CheckItem(String resource, String action) { this.resource = resource; this.action = action; }
    }
}
