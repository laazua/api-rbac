package com.ops.controller;

import com.rbac.RequirePermission;
import org.springframework.web.bind.annotation.*;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;

/**
 * 服务器管理 — 纯业务逻辑，权限由 @RequirePermission 注解控制。
 *
 * <p>注意: Controller 中没有任何权限判断代码。
 * 所有鉴权由 WebConfig 中的拦截器自动完成。</p>
 */
@RestController
@RequestMapping("/api/servers")
public class ServerController {

    // 模拟数据库
    private final Map<Long, Server> db = new ConcurrentHashMap<>();
    private final AtomicLong idGen = new AtomicLong(3);

    public ServerController() {
        db.put(1L, new Server(1, "web-01", "10.0.1.10", "running"));
        db.put(2L, new Server(2, "web-02", "10.0.1.11", "stopped"));
        db.put(3L, new Server(3, "db-01",  "10.0.2.10", "running"));
    }

    @GetMapping
    @RequirePermission(resource = "server", action = "read")
    public Map<String, Object> list() {
        return Map.of("code", 0, "data", new ArrayList<>(db.values()));
    }

    @PostMapping("/{id}/restart")
    @RequirePermission(resource = "server", action = "restart")
    public Map<String, Object> restart(@PathVariable long id) {
        var s = db.get(id);
        if (s == null) return Map.of("code", 404, "message", "服务器不存在");
        db.put(id, new Server(s.id, s.name, s.ip, "running"));
        return Map.of("code", 0, "message", "服务器 " + s.name + " 重启成功");
    }

    @PostMapping("/{id}/stop")
    @RequirePermission(resource = "server", action = "stop")
    public Map<String, Object> stop(@PathVariable long id) {
        var s = db.get(id);
        if (s == null) return Map.of("code", 404, "message", "服务器不存在");
        db.put(id, new Server(s.id, s.name, s.ip, "stopped"));
        return Map.of("code", 0, "message", "服务器 " + s.name + " 已停止");
    }

    record Server(long id, String name, String ip, String status) {}
}
