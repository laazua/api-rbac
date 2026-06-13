# RBAC Python SDK

用于从 Python 业务系统集成 RBAC 权限管理微服务。

## 安装

```bash
pip install rbac-client
# 或直接复制 rbac_client.py 到项目中
```

## 快速开始

```python
from rbac_client import RBACClient

client = RBACClient("http://localhost:8087/api/v1")

# 1. 登录
result = client.login("admin", "password")
token = result["token"]
refresh_token = result["refresh_token"]

# 2. 验证 Token
user = client.verify(token)
print(f"当前用户: {user['username']}")

# 3. 检查权限
if client.check_permission(token, "user", "delete"):
    print("有删除用户的权限")
else:
    print("无权限")

# 4. 批量检查
perms = client.batch_check(token, [("user", "read"), ("user", "delete"), ("role", "create")])
print(perms)  # {"user:read": True, "user:delete": False, "role:create": True}

# 5. Token 自省（外部服务使用）
info = client.introspect(token, resource="order", action="read")
if info["active"]:
    print(f"Token 有效，用户: {info['username']}")

# 6. 刷新 Token
if token_expired:
    new_tokens = client.refresh(refresh_token)
    token = new_tokens["token"]
    refresh_token = new_tokens["refresh_token"]
```

## 在 Flask 中使用

```python
from flask import Flask, request, jsonify
from rbac_client import RBACClient

app = Flask(__name__)
rbac = RBACClient("http://rbac-service:8087/api/v1")

def require_permission(resource: str, action: str):
    """权限校验装饰器"""
    def decorator(f):
        from functools import wraps
        @wraps(f)
        def wrapper(*args, **kwargs):
            token = request.headers.get("Authorization", "").replace("Bearer ", "")
            if not token:
                return jsonify({"error": "未提供认证Token"}), 401
            if not rbac.check_permission(token, resource, action):
                return jsonify({"error": "无权限"}), 403
            return f(*args, **kwargs)
        return wrapper
    return decorator

@app.route("/orders/delete", methods=["DELETE"])
@require_permission("order", "delete")
def delete_order():
    # 业务逻辑...
    return jsonify({"success": True})
```

## API 文档

### RBACClient(base_url, timeout=10)

| 方法 | 说明 |
|------|------|
| `login(account, password)` | 登录，返回 Token 对 |
| `refresh(refresh_token)` | 刷新 Token |
| `verify(token)` | 验证 Token 有效性 |
| `check_permission(token, resource, action)` | 检查单个权限 |
| `batch_check(token, permissions)` | 批量检查权限 |
| `introspect(token, resource?, action?)` | Token 自省 |
| `get_menu_via_get(token)` | 获取用户权限菜单 |
