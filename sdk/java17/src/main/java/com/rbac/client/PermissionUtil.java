package com.rbac.client;

import java.util.*;

/**
 * 权限辅助工具 — 前端/后端通用的权限判断函数。
 *
 * <p>支持通配符: *:*、resource:*、*:action。</p>
 *
 * <pre>{@code
 * var perms = client.getMenu(token);
 * if (PermissionUtil.hasPerm(perms, "server", "restart")) { ... }
 * if (PermissionUtil.hasAnyPerm(perms, "server")) { ... }
 * }</pre>
 */
public final class PermissionUtil {

    private PermissionUtil() {}

    public static boolean hasPerm(Map<String, List<String>> perms, String resource, String action) {
        if (perms == null || perms.isEmpty()) return false;
        // 通配符
        var star = perms.get("*");
        if (star != null && (star.contains("*") || star.contains(action))) return true;
        // 精确匹配
        var res = perms.get(resource);
        return res != null && (res.contains("*") || res.contains(action));
    }

    public static boolean hasAnyPerm(Map<String, List<String>> perms, String resource) {
        if (perms == null || perms.isEmpty()) return false;
        if (perms.containsKey("*")) return true;
        var actions = perms.get(resource);
        return actions != null && !actions.isEmpty();
    }

    public static List<String> getActions(Map<String, List<String>> perms, String resource) {
        var result = new ArrayList<String>();
        if (perms.containsKey("*")) result.addAll(perms.get("*"));
        if (perms.containsKey(resource)) result.addAll(perms.get(resource));
        return result;
    }
}
