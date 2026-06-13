package com.ops.config;

import com.rbac.RBACClient;
import com.rbac.PermissionUtil;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.time.Duration;

/**
 * 应用配置 — 初始化 RBAC 客户端 Bean。
 */
@Configuration
public class AppConfig {

    @Value("${ops.rbac.url:http://localhost:8087/api/v1}")
    private String rbacUrl;

    @Bean
    public RBACClient rbacClient() {
        return new RBACClient(rbacUrl, Duration.ofSeconds(10));
    }
}
