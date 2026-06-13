package com.rbac.client;

import java.lang.annotation.*;

/**
 * 标注 Controller 方法所需的权限，配合 PermissionInterceptor 使用。
 *
 * <pre>{@code
 * @PostMapping("/api/servers/{id}/restart")
 * @RequirePermission(resource = "server", action = "restart")
 * public void restart(@PathVariable int id) { ... }
 * }</pre>
 */
@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
public @interface RequirePermission {
    String resource();
    String action();
}
