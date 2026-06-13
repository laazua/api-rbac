package com.ops;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * 运维管理系统 — Spring Boot 入口。
 *
 * <p>访问 http://localhost:8080 查看前端页面。
 * 所有权限管理由 api-rbac 微服务提供，本项目不存储任何用户/角色/权限数据。</p>
 */
@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
