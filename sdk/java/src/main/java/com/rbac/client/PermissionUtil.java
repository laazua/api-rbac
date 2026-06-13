package com.rbac.client;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * 权限辅助工具类 — 前端/后端通用的权限判断函数。
 *
 * <p>用于根据 api-rbac 返回的权限 Map 判断用户是否拥有某个权限。
 * 支持通配符 *:*、resource:*、*:action。</p>
 *
 * <pre>{@code
 * // 从 RBAC 获取权限
 * Map<String, List<String>> perms = client.getMenu(token);
 *
 * // 判断权限
 * if (PermissionUtil.hasPerm(perms, "server", "restart")) {
 *     // 有权限
 * }
 *
 * // 判断是否有模块的任何权限 (用于菜单显隐)
 * if (PermissionUtil.hasAnyPerm(perms, "server")) {
 *     // 显示"服务器管理"菜单
 * }
 * }</pre>
 */
public final class PermissionUtil {

    private PermissionUtil() {}

    /**
     * 判断是否拥有指定资源的指定操作权限。
     */
    public static boolean hasPerm(Map<String, List<String>> perms, String resource, String action) {
        if (perms == null || perms.isEmpty()) return false;

        // *:* 通配
        List<String> starActions = perms.get("*");
        if (starActions != null && (starActions.contains("*") || starActions.contains(action))) {
            return true;
        }

        // resource:*
        List<String> resActions = perms.get(resource);
        if (resActions != null && (resActions.contains("*") || resActions.contains(action))) {
            return true;
        }

        return false;
    }

    /**
     * 判断是否至少拥有指定资源的任意一个权限（用于菜单显隐）。
     */
    public static boolean hasAnyPerm(Map<String, List<String>> perms, String resource) {
        if (perms == null || perms.isEmpty()) return false;
        if (perms.containsKey("*")) return true;
        List<String> actions = perms.get(resource);
        return actions != null && !actions.isEmpty();
    }

    /**
     * 从 permMap 中提取某资源的全部 action 列表。
     */
    public static List<String> getActions(Map<String, List<String>> perms, String resource) {
        List<String> result = new ArrayList<>();
        // 通配符的 action 也合并进来
        if (perms.containsKey("*")) result.addAll(perms.get("*"));
        if (perms.containsKey(resource)) result.addAll(perms.get(resource));
        return result;
    }
}
