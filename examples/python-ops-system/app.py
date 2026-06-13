"""
运维管理系统 — Python Flask 实现
完整演示如何将 api-rbac 作为独立的权限管理微服务集成

运行方式:
    pip install flask requests
    # 确保 api-rbac 服务已启动
    python app.py
"""

import sys
import os

# 引入 Python SDK
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../sdk/python'))
from rbac_client import RBACClient, FailMode

from flask import Flask, request, jsonify, send_from_directory
from functools import wraps

app = Flask(__name__, static_folder='static', static_url_path='')

# ============================================================
# 配置：指向独立运行的 api-rbac 服务 (韧性模式)
# ============================================================
RBAC_URL = "http://localhost:8087/api/v1"
rbac = RBACClient(
    RBAC_URL,
    timeout=5,                          # 单次请求超时 5s
    fail_mode=FailMode.CACHE,           # 故障时用本地缓存, 不拒绝请求
    cache_ttl=300,                      # 缓存有效期 5 分钟
    circuit_breaker_threshold=5,        # 连续 5 次失败触发熔断
    circuit_breaker_recovery=30,        # 熔断后 30s 尝试恢复
)


# ============================================================
# 权限校验装饰器 — 内置韧性降级
# ============================================================
def require_permission(resource, action):
    """
    Flask 装饰器：校验当前请求是否有指定权限。
    RBAC 可用时 → 正常远程校验
    RBAC 不可用时 → 自动降级到本地缓存 (FailMode.CACHE)
                    若缓存中无该用户数据 → 安全拒绝 (fail-closed)
    """
    def decorator(f):
        @wraps(f)
        def wrapper(*args, **kwargs):
            token = request.headers.get("Authorization", "").replace("Bearer ", "")
            if not token:
                return jsonify({"code": 401, "message": "未登录"}), 401

            try:
                if not rbac.check_permission(token, resource, action):
                    return jsonify({"code": 403, "message": f"无权限: {resource}:{action}"}), 403
            except RuntimeError as e:
                # RBAC 完全不可达 + FailMode.DENY 或缓存缺失 → 安全拒绝
                return jsonify({"code": 502, "message": f"权限服务不可用: {str(e)}"}), 502

            return f(*args, **kwargs)
        return wrapper
    return decorator


# ============================================================
# 0. 前端入口 — 所有页面由前端 SPA 渲染，后端只提供 API
# ============================================================
@app.route("/")
def index():
    """返回前端 SPA 页面"""
    return send_from_directory("static", "index.html")


# ============================================================
# 1. 登录 — 转发到 api-rbac，业务系统不存密码
# ============================================================
@app.route("/api/auth/login", methods=["POST"])
def login():
    data = request.get_json()
    try:
        result = rbac.login(data["account"], data["password"])
        # 登录成功，返回 Token 对给前端
        return jsonify({
            "code": 0,
            "data": {
                "token": result["token"],
                "refresh_token": result["refresh_token"],
                "expires_in": result["expires_in"],
                "user_id": result["user_id"],
                "username": result["username"],
            }
        })
    except Exception as e:
        return jsonify({"code": 401, "message": str(e)}), 401


@app.route("/api/auth/refresh", methods=["POST"])
def refresh_token():
    data = request.get_json()
    try:
        result = rbac.refresh(data["refresh_token"])
        return jsonify({"code": 0, "data": result})
    except Exception as e:
        return jsonify({"code": 401, "message": str(e)}), 401


# ============================================================
# 2. 服务器管理 — 纯业务逻辑，权限完全走 api-rbac
# ============================================================

# 模拟业务数据库
servers_db = [
    {"id": 1, "name": "web-01", "ip": "10.0.1.10", "status": "running"},
    {"id": 2, "name": "web-02", "ip": "10.0.1.11", "status": "stopped"},
    {"id": 3, "name": "db-01", "ip": "10.0.2.10", "status": "running"},
]


@app.route("/api/servers", methods=["GET"])
@require_permission("server", "read")
def list_servers():
    """查看服务器列表 — 需要 server:read 权限"""
    return jsonify({"code": 0, "data": servers_db})


@app.route("/api/servers/<int:server_id>/restart", methods=["POST"])
@require_permission("server", "restart")
def restart_server(server_id):
    """重启服务器 — 需要 server:restart 权限"""
    for s in servers_db:
        if s["id"] == server_id:
            s["status"] = "running"
            return jsonify({"code": 0, "message": f"服务器 {s['name']} 重启成功"})
    return jsonify({"code": 404, "message": "服务器不存在"}), 404


@app.route("/api/servers/<int:server_id>/stop", methods=["POST"])
@require_permission("server", "stop")
def stop_server(server_id):
    """停止服务器 — 需要 server:stop 权限"""
    for s in servers_db:
        if s["id"] == server_id:
            s["status"] = "stopped"
            return jsonify({"code": 0, "message": f"服务器 {s['name']} 已停止"})
    return jsonify({"code": 404, "message": "服务器不存在"}), 404


# ============================================================
# 3. 发布管理
# ============================================================

deployments_db = [
    {"id": 1, "project": "web-app", "version": "v2.3.1", "status": "success"},
    {"id": 2, "project": "api-service", "version": "v1.5.0", "status": "failed"},
]


@app.route("/api/deployments", methods=["GET"])
@require_permission("deployment", "read")
def list_deployments():
    return jsonify({"code": 0, "data": deployments_db})


@app.route("/api/deployments", methods=["POST"])
@require_permission("deployment", "execute")
def create_deployment():
    """执行发布 — 需要 deployment:execute 权限"""
    data = request.get_json()
    new_id = len(deployments_db) + 1
    deployments_db.append({
        "id": new_id,
        "project": data["project"],
        "version": data["version"],
        "status": "pending",
    })
    # 模拟：业务系统自己的部署逻辑...
    deployments_db[-1]["status"] = "success"
    return jsonify({"code": 0, "data": deployments_db[-1]})


@app.route("/api/deployments/<int:deploy_id>/rollback", methods=["POST"])
@require_permission("deployment", "rollback")
def rollback_deployment(deploy_id):
    """回滚发布 — 需要 deployment:rollback 权限"""
    for d in deployments_db:
        if d["id"] == deploy_id:
            d["status"] = "rolled_back"
            return jsonify({"code": 0, "message": f"项目 {d['project']} 已回滚"})
    return jsonify({"code": 404, "message": "发布记录不存在"}), 404


# ============================================================
# 4. 告警管理
# ============================================================

alerts_db = [
    {"id": 1, "level": "critical", "message": "CPU 使用率 95%", "acked": False},
    {"id": 2, "level": "warning", "message": "磁盘使用率 80%", "acked": True},
]


@app.route("/api/alerts", methods=["GET"])
@require_permission("alert", "read")
def list_alerts():
    return jsonify({"code": 0, "data": alerts_db})


@app.route("/api/alerts/<int:alert_id>/ack", methods=["POST"])
@require_permission("alert", "ack")
def ack_alert(alert_id):
    """确认告警 — 需要 alert:ack 权限"""
    for a in alerts_db:
        if a["id"] == alert_id:
            a["acked"] = True
            return jsonify({"code": 0, "message": "告警已确认"})
    return jsonify({"code": 404, "message": "告警不存在"}), 404


# ============================================================
# 5. 当前用户权限查询
# ============================================================
@app.route("/api/user/permissions", methods=["GET"])
def get_my_permissions():
    """获取当前用户的权限列表，前端用于菜单/按钮显隐"""
    token = request.headers.get("Authorization", "").replace("Bearer ", "")
    if not token:
        return jsonify({"code": 401, "message": "未登录"}), 401
    try:
        perms = rbac.get_menu_via_get(token)
        return jsonify({"code": 0, "data": perms})
    except Exception as e:
        return jsonify({"code": 502, "message": str(e)}), 502


# ============================================================
# 启动
# ============================================================
if __name__ == "__main__":
    print("=" * 60)
    print("  运维管理系统 (Python Flask)")
    print(f"  RBAC 服务: {RBAC_URL}")
    print("=" * 60)
    print()
    print("  端点示例:")
    print("    POST /api/auth/login         — 登录 (转发到 RBAC)")
    print("    GET  /api/servers             — 服务器列表 (需要 server:read)")
    print("    POST /api/servers/1/restart   — 重启服务器 (需要 server:restart)")
    print("    GET  /api/deployments         — 发布列表 (需要 deployment:read)")
    print("    POST /api/deployments         — 执行发布 (需要 deployment:execute)")
    print("    GET  /api/alerts              — 告警列表 (需要 alert:read)")
    print()
    app.run(host="0.0.0.0", port=5000, debug=True)
