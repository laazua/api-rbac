package com.ops.controller;

import com.rbac.RequirePermission;
import org.springframework.web.bind.annotation.*;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 告警管理 — 同样只需 @RequirePermission 注解。
 */
@RestController
@RequestMapping("/api/alerts")
public class AlertController {

    private final Map<Long, Alert> db = new ConcurrentHashMap<>();

    public AlertController() {
        db.put(1L, new Alert(1, "critical", "CPU 使用率 95%", false));
        db.put(2L, new Alert(2, "warning", "磁盘使用率 80%", true));
    }

    @GetMapping
    @RequirePermission(resource = "alert", action = "read")
    public Map<String, Object> list() {
        return Map.of("code", 0, "data", new ArrayList<>(db.values()));
    }

    @PostMapping("/{id}/ack")
    @RequirePermission(resource = "alert", action = "ack")
    public Map<String, Object> ack(@PathVariable long id) {
        var a = db.get(id);
        if (a == null) return Map.of("code", 404, "message", "告警不存在");
        db.put(id, new Alert(a.id, a.level, a.message, true));
        return Map.of("code", 0, "message", "告警已确认");
    }

    record Alert(long id, String level, String message, boolean acked) {}
}
