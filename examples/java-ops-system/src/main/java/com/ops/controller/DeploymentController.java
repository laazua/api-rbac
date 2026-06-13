package com.ops.controller;

import com.rbac.RequirePermission;
import org.springframework.web.bind.annotation.*;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;

/**
 * 发布管理 — 与服务器管理一样，纯业务逻辑 + @RequirePermission。
 */
@RestController
@RequestMapping("/api/deployments")
public class DeploymentController {

    private final Map<Long, Deployment> db = new ConcurrentHashMap<>();
    private final AtomicLong idGen = new AtomicLong(2);

    public DeploymentController() {
        db.put(1L, new Deployment(1, "web-app", "v2.3.1", "success"));
        db.put(2L, new Deployment(2, "api-service", "v1.5.0", "failed"));
    }

    @GetMapping
    @RequirePermission(resource = "deployment", action = "read")
    public Map<String, Object> list() {
        return Map.of("code", 0, "data", new ArrayList<>(db.values()));
    }

    @PostMapping
    @RequirePermission(resource = "deployment", action = "execute")
    public Map<String, Object> execute(@RequestBody Map<String, String> body) {
        long id = idGen.incrementAndGet();
        var d = new Deployment(id, body.get("project"), body.get("version"), "success");
        db.put(id, d);
        return Map.of("code", 0, "data", d);
    }

    @PostMapping("/{id}/rollback")
    @RequirePermission(resource = "deployment", action = "rollback")
    public Map<String, Object> rollback(@PathVariable long id) {
        var d = db.get(id);
        if (d == null) return Map.of("code", 404, "message", "发布记录不存在");
        db.put(id, new Deployment(d.id, d.project, d.version, "rolled_back"));
        return Map.of("code", 0, "message", "项目 " + d.project + " 已回滚");
    }

    record Deployment(long id, String project, String version, String status) {}
}
