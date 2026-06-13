/**
 * Java 模块系统声明 (JPMS) — 适用于 Java 17+ 模块化项目。
 */
module com.rbac.client {
    exports com.rbac.client;

    // 可选依赖 (仅在项目引入时可用)
    requires static jakarta.servlet;
}
