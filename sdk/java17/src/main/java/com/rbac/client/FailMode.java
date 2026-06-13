package com.rbac.client;

/**
 * 故障模式 — RBAC 服务不可达时的行为。
 *
 * <p>DENY (默认, 安全): 拒绝所有权限检查请求。适合安全敏感场景。</p>
 * <p>CACHE (推荐, 韧性): 使用本地缓存的权限数据。适合需要高可用的场景。</p>
 */
public enum FailMode {
    DENY,
    CACHE
}
