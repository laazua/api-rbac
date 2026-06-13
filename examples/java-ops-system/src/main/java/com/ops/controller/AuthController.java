package com.ops.controller;

import com.rbac.RBACClient;
import com.rbac.RBACException;
import com.rbac.RequirePermission;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

/**
 * 认证接口 — 全部转发给 api-rbac，业务系统不存储密码。
 *
 * <p>登录/刷新Token 不需要 @RequirePermission 注解 (用户还没有 Token)。</p>
 */
@RestController
@RequestMapping("/api/auth")
public class AuthController {

    private final RBACClient rbac;

    public AuthController(RBACClient rbac) {
        this.rbac = rbac;
    }

    /**
     * 登录 — 直接转发给 api-rbac。
     * 前端拿到 token + refresh_token 后自行存储。
     */
    @PostMapping("/login")
    public Map<String, Object> login(@RequestBody Map<String, String> body) throws RBACException {
        var result = rbac.login(body.get("account"), body.get("password"));
        return Map.of(
                "code", 0,
                "data", Map.of(
                        "token", result.token(),
                        "refresh_token", result.refreshToken(),
                        "expires_in", result.expiresIn(),
                        "user_id", result.userId(),
                        "username", result.username()));
    }

    /** 刷新 Token */
    @PostMapping("/refresh")
    public Map<String, Object> refresh(@RequestBody Map<String, String> body) throws RBACException {
        var result = rbac.refresh(body.get("refresh_token"));
        return Map.of("code", 0, "data", Map.of(
                "token", result.token(),
                "refresh_token", result.refreshToken(),
                "expires_in", result.expiresIn()));
    }

    /** 获取当前用户的全部权限 (前端用于菜单/按钮显隐) */
    @GetMapping("/permissions")
    public Map<String, Object> getPermissions(@RequestHeader("Authorization") String authHeader)
            throws RBACException {
        String token = authHeader.replace("Bearer ", "");
        var perms = rbac.getMenu(token);
        return Map.of("code", 0, "data", perms);
    }
}
