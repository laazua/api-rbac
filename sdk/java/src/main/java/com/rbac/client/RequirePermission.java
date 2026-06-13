package com.rbac.client;

import java.lang.annotation.*;

/**
 * 标注在 Controller 方法上，声明该方法需要的权限。
 *
 * <p>配合 {@link PermissionInterceptor} 使用，自动完成权限校验。</p>
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

    /** 资源标识，如 "server", "deployment", "alert" */
    String resource();

    /** 操作标识，如 "read", "create", "restart", "delete" */
    String action();
}
