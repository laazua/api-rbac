# 运维管理系统 — api-rbac 跨语言权限集成完整示例

> 演示如何将 **api-rbac (Go)** 作为独立权限管理微服务，与 **Python Flask** 业务系统**完全解耦**地集成。

---

## 目录

- [1. 这个示例解决什么问题](#1-这个示例解决什么问题)
- [2. 核心架构](#2-核心架构)
- [3. 解耦原理：谁管什么](#3-解耦原理谁管什么)
- [4. 鉴权流程详解](#4-鉴权流程详解)
- [5. 运行步骤](#5-运行步骤)
- [6. 代码导读](#6-代码导读)
- [7. 权限模型映射](#7-权限模型映射)
- [8. 前端权限集成（重点）](#8-前端权限集成重点)
- [9. 新增业务模块](#9-新增业务模块)
- [10. 安全最佳实践](#10-安全最佳实践)
- [11. 常见问题排查](#11-常见问题排查)
- [附录: api-rbac 完整鉴权 API](#附录-api-rbac-完整的鉴权-api)

---

## 1. 这个示例解决什么问题

当你用 Python（或 Java/Node.js/PHP）开发业务系统时，**权限管理**往往是最头疼的跨模块问题：

- 每个业务系统都要重复写登录、鉴权、角色管理
- 权限数据散落在各系统中，无法统一管理
- 不同语言实现的服务无法共享权限体系

**这个示例展示的解决方案**：把权限管理完全抽离为一个独立的 Go 微服务（api-rbac），业务系统只需通过 HTTP 调用它即可完成所有鉴权操作。

### 运行效果

```
运维管理系统 (Python Flask :5000)            api-rbac (Go :8087)
──────────────────────────────────         ──────────────────────
用户登录 → POST /api/auth/login ──────────→ 验证密码 → 签发 JWT
查看服务器 → GET /api/servers               (不参与)
  └─ @require_permission("server","read") ─→ 检查用户是否有 server:read 权限
                                            └─ 有 → 放行 / 无 → 403
重启服务器 → POST /api/servers/1/restart     (不参与)
  └─ @require_permission("server","restart")→ 检查用户是否有 server:restart 权限
                                            └─ 有 → 放行 / 无 → 403
```

---

## 2. 核心架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    api-rbac (Go 微服务 :8087)                      │
│                                                                 │
│  职责范围 (仅此而已):                                                │
│  ┌─────────────┐ ┌──────────────┐ ┌────────────────┐             │
│  │ 用户管理     │ │ 角色管理      │ │ 权限管理        │             │
│  │ CRUD + 登录  │ │ CRUD + 绑定   │ │ CRUD + 资源/动作 │            │
│  └─────────────┘ └──────────────┘ └────────────────┘             │
│  ┌─────────────┐ ┌──────────────┐ ┌────────────────┐             │
│  │ JWT 签发     │ │ Token 刷新   │ │ 权限检查 API    │             │
│  │ /auth/login  │ │ /auth/refresh│ │ /auth/check    │             │
│  └─────────────┘ └──────────────┘ └────────────────┘             │
│                                                                 │
│  它完全不知道:                                                     │
│  - "服务器"是什么、有什么属性                                        │
│  - "发布"的流程是什么                                               │
│  - "告警"的级别有哪些                                               │
│  - 任何业务数据的结构和关系                                          │
│                                                                 │
│  它只知道抽象的三元组: (用户ID, resource, action) → true/false       │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │  HTTP REST API (JSON)
                           │  任何编程语言都可以调用
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │ Python   │    │ Java     │    │ Node.js  │
    │ 业务系统  │    │ 业务系统  │    │ 业务系统  │
    └──────────┘    └──────────┘    └──────────┘
```

---

## 3. 解耦原理：谁管什么

### 3.1 职责分界线

| 职责 | 谁负责 | 说明 |
|------|--------|------|
| 用户密码存储与验证 | **api-rbac** | bcrypt 加密存储，业务系统不碰密码 |
| JWT Token 签发与解析 | **api-rbac** | 密钥只存在于 RBAC 服务 |
| 角色-权限-用户绑定 | **api-rbac** | 多对多关系管理 |
| 权限校验 `(用户, 资源, 操作) → bool` | **api-rbac** | `/auth/check` 接口 |
| 服务器 CRUD 逻辑 | **业务系统** | api-rbac 不知道什么是"服务器" |
| 发布流程编排 | **业务系统** | api-rbac 不知道什么是"发布" |
| 告警阈值判断 | **业务系统** | api-rbac 不知道什么是"告警" |
| 业务数据存储 | **业务系统** | 自己的 MySQL/MongoDB 等 |

### 3.2 为什么能做到解耦

api-rbac 的权限模型是**完全抽象的**：

```
权限 = Resource (资源标识) + Action (操作标识)

它不是:  "能否重启 server-01？"
而是:    "用户 X 是否有 resource=server, action=restart 的权限？"

api-rbac 只比对字符串，不关心:
  - server 是什么东西
  - restart 是什么意思
  - 用户 X 是谁（只管 ID）
```

这就像操作系统的文件权限 —— OS 不知道你的文件内容是什么，只管 `rwx` 位。

---

## 4. 鉴权流程详解

### 4.1 登录流程

```
前端 (浏览器)              业务系统 (Flask :5000)           api-rbac (:8087)
──────                     ────────────────              ───────────────
                                                              
  填写用户名密码                                              
  POST /api/auth/login ───→  收到 {account,password}           
                             转发 POST /auth/login ──────────→ 查询用户
                                                              bcrypt.ComparePassword
                                                              签发 JWT + RefreshToken
                             收到 {token,refresh_token,...} ←── 
                             返回给前端                        
  存储 token 到 localStorage                                  
  后续请求带 Authorization                                     
```

> **关键点**：业务系统**从未接触过用户的明文密码**。密码只在浏览器和 api-rbac 之间通过业务系统中转，业务系统只是透传。

### 4.2 权限检查流程

```
前端 (浏览器)              业务系统 (Flask :5000)           api-rbac (:8087)
──────                     ────────────────              ───────────────
                                                              
  POST /api/servers/1/restart                                 
  Authorization: Bearer xxx ──→ @require_permission("server","restart")
                               提取 token                      
                               调 rbac.check_permission(token, "server", "restart")
                               POST /auth/check ──────────────→ 解析 JWT 获取 userID
                                {resource:"server",            ← 从 Redis/DB 加载用户权限
                                 action:"restart"}             通配符匹配
                                                              返回 {allowed: true/false}
                               收到 {allowed: true}  ←──────── 
                               放行，执行业务逻辑              
                               返回重启结果                    
  {code:0, message:"重启成功"}                                 
```

### 4.3 权限被拒绝的流程

```
  POST /api/servers/1/restart                                 
  Authorization: Bearer xxx ──→ @require_permission("server","restart")
                               调 rbac.check_permission(token, "server", "restart")
                               POST /auth/check ──────────────→ 用户仅有 server:read
                                                              没有 server:restart
                               {allowed: false}       ←─────── 
                               返回 403                       
  {code:403, "无权限: server:restart"}                         
  (业务逻辑完全不执行)                                          
```

### 4.4 推荐：使用 Token 自省 (一次调用完成两步)

对于外部服务，推荐用 `/auth/introspect` 接口，将 Token 验证 + 权限检查合并为一次 HTTP 调用：

```
  外部服务收到请求                         api-rbac (:8087)
  ──────────────                          ───────────────
  POST /auth/introspect                   解析 Token (验证签名+过期)
  {token, resource, action} ────────────→ 加载权限并匹配
                                          {active: true/false} ← 

  一次 HTTP 往返，完成鉴权
```

---

## 5. 运行步骤

### 前提条件

- Go 1.21+ (用于运行 api-rbac)
- Python 3.8+ (用于运行运维系统)
- MySQL 5.7+ 或 8.0+ (api-rbac 数据存储)
- Redis (可选，用于权限缓存加速)

### 5.1 启动 api-rbac

```bash
# 进入 api-rbac 项目目录
cd /opt/codes/api-rbac

# 1. 创建数据库
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS api_rbac CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2. 编辑配置文件 (修改数据库连接信息)
vim config/config.yaml

# 3. 编译并启动 (首次运行会提示设置 admin 密码)
make build
./api-rbac
# 输出: 服务启动于 http://0.0.0.0:8087
```

### 5.2 初始化运维系统权限

```bash
# 进入示例目录
cd examples/python-ops-system

# 赋予执行权限
chmod +x setup_permissions.sh

# 运行初始化脚本 (会根据提示输入 admin 密码)
./setup_permissions.sh
```

脚本会自动在 api-rbac 中创建：

| 类型 | 名称 | 详情 |
|------|------|------|
| 权限 | 查看服务器 | `server:read` |
| 权限 | 重启服务器 | `server:restart` |
| 权限 | 停止服务器 | `server:stop` |
| 权限 | 查看发布 | `deployment:read` |
| 权限 | 执行发布 | `deployment:execute` |
| 权限 | 回滚发布 | `deployment:rollback` |
| 权限 | 查看告警 | `alert:read` |
| 权限 | 确认告警 | `alert:ack` |
| 角色 | 运维管理员 | 绑定全部 8 个权限 |
| 角色 | 运维查看者 | 仅绑定 read 类权限 |

### 5.3 创建测试用户

```bash
# 创建一个运维管理员用户
curl -X POST http://localhost:8087/api/v1/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"opsadmin","password":"123456","email":"ops@example.com"}'

# 给该用户分配"运维管理员"角色 (假设角色 ID 为 2)
curl -X POST http://localhost:8087/api/v1/users/2/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role_ids":[2]}'
```

### 5.4 启动运维管理系统

```bash
pip install flask requests
python app.py

# 输出:
# ============================================================
#   运维管理系统 (Python Flask)
#   RBAC 服务: http://localhost:8087/api/v1
# ============================================================
#   * Running on http://0.0.0.0:5000
```

### 5.5 端到端测试

```bash
# ===== 1. 登录 =====
LOGIN=$(curl -s -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account":"opsadmin","password":"123456"}')

echo $LOGIN | python3 -m json.tool
# 响应包含: token, refresh_token, expires_in, user_id, username

TOKEN=$(echo $LOGIN | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

# ===== 2. 查看服务器列表 (需要 server:read) =====
curl -s http://localhost:5000/api/servers \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# ===== 3. 重启服务器 (需要 server:restart) =====
curl -s -X POST http://localhost:5000/api/servers/1/restart \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# ===== 4. 执行发布 (需要 deployment:execute) =====
curl -s -X POST http://localhost:5000/api/deployments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"project":"ops-dashboard","version":"v3.0.0"}' | python3 -m json.tool

# ===== 5. 确认告警 (需要 alert:ack) =====
curl -s -X POST http://localhost:5000/api/alerts/1/ack \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# ===== 6. 测试无权限场景 =====
# 如果用户只有"运维查看者"角色，重启服务器会被拒绝:
# {"code": 403, "message": "无权限: server:restart"}
```

---

## 6. 代码导读

### 6.1 文件结构

```
examples/python-ops-system/
├── README.md              ← 你正在读的文件
├── app.py                 ← 运维管理系统主程序 (~230行)
└── setup_permissions.sh   ← 权限初始化脚本
```

### 6.2 app.py 核心代码分段解读

#### 初始化：一行配置指向 RBAC 服务

```python
from rbac_client import RBACClient

RBAC_URL = "http://localhost:8087/api/v1"  # ← 只需改这一个地址
rbac = RBACClient(RBAC_URL)
```

#### 权限校验装饰器：整个系统只需要这个

```python
def require_permission(resource, action):
    """Flask 装饰器：校验当前请求是否有指定权限"""
    def decorator(f):
        @wraps(f)
        def wrapper(*args, **kwargs):
            # 1. 从请求头提取 JWT Token
            token = request.headers.get("Authorization", "").replace("Bearer ", "")
            if not token:
                return jsonify({"code": 401, "message": "未登录"}), 401

            # 2. 调用 RBAC 服务检查权限
            try:
                if not rbac.check_permission(token, resource, action):
                    return jsonify({"code": 403, "message": f"无权限: {resource}:{action}"}), 403
            except Exception as e:
                return jsonify({"code": 502, "message": f"权限服务异常: {str(e)}"}), 502

            # 3. 权限通过，执行真正的业务逻辑
            return f(*args, **kwargs)
        return wrapper
    return decorator
```

#### 登录接口：转发给 RBAC，业务系统不存密码

```python
@app.route("/api/auth/login", methods=["POST"])
def login():
    data = request.get_json()
    result = rbac.login(data["account"], data["password"])  # ← 转发给 RBAC
    return jsonify({
        "code": 0,
        "data": {
            "token": result["token"],                   # Access Token (2h)
            "refresh_token": result["refresh_token"],   # Refresh Token (7d)
            "expires_in": result["expires_in"],         # 过期秒数
            "user_id": result["user_id"],
            "username": result["username"],
        }
    })
```

#### 业务接口：加一个装饰器就完成鉴权

```python
# 业务代码 100% 是业务逻辑，权限代码只有 @require_permission 一行

@app.route("/api/servers", methods=["GET"])
@require_permission("server", "read")    # ← 仅此一行权限代码
def list_servers():
    """查看服务器列表"""
    return jsonify({"code": 0, "data": servers_db})  # ← 纯业务逻辑

@app.route("/api/servers/<int:server_id>/restart", methods=["POST"])
@require_permission("server", "restart")  # ← 仅此一行
def restart_server(server_id):
    """重启服务器"""
    for s in servers_db:
        if s["id"] == server_id:
            s["status"] = "running"
            return jsonify({"code": 0, "message": f"服务器 {s['name']} 重启成功"})
    return jsonify({"code": 404, "message": "服务器不存在"}), 404
```

### 6.3 setup_permissions.sh 分段解读

脚本分三步在 api-rbac 中初始化权限体系：

```bash
# 第1步: 登录获取管理员 Token
RESP=$(curl -s -X POST "$RBAC_URL/auth/login" \
  -d '{"account":"admin","password":"Admin123456"}')
ADMIN_TOKEN=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

# 第2步: 逐个创建权限 (8个)
# 每条权限定义: resource + action
curl -X POST "$RBAC_URL/permissions" -d '{"name":"重启服务器","resource":"server","action":"restart",...}'
curl -X POST "$RBAC_URL/permissions" -d '{"name":"执行发布","resource":"deployment","action":"execute",...}'
# ...

# 第3步: 创建角色并绑定权限
# "运维管理员" → 绑定全部8个权限
# "运维查看者" → 只绑定 read 类权限
```

---

## 7. 权限模型映射

### 7.1 运维业务 → api-rbac 权限映射表

| 业务操作 | 对应权限 (resource:action) | 说明 |
|----------|---------------------------|------|
| 查看服务器列表 | `server:read` | 列表页 + 详情页 |
| 重启服务器 | `server:restart` | 重启操作 |
| 停止服务器 | `server:stop` | 关机操作 |
| 查看发布记录 | `deployment:read` | 发布历史 |
| 执行新发布 | `deployment:execute` | 创建部署任务 |
| 回滚发布 | `deployment:rollback` | 回退到旧版本 |
| 查看告警列表 | `alert:read` | 告警面板 |
| 确认告警 | `alert:ack` | 处理/关闭告警 |

### 7.2 通配符权限

在 api-rbac 中，`*` 是通配符：

```python
# 超级管理员拥有 *:* → 所有权限检查都返回 True
# 模块管理员拥有 server:* → server 的所有操作都允许
# 只读用户拥有 *:read → 所有资源的查看操作都允许
```

### 7.3 建议的命名规范

```
resource:action 格式

resource:  小写英文单词，用下划线分隔  (server_log, k8s_cluster)
action:    标准 CRUD 动词: read / create / update / delete
           或自定义动词: restart / execute / rollback / ack / download

示例:
  ✅ server:read          ✅ k8s_cluster:deploy
  ✅ user_log:download    ✅ billing_invoice:export
  ❌ Server:Read          ❌ 服务器:查看  (避免大写和中文)
```

---

## 8. 前端权限集成（重点）

> **核心问题**：解耦后的业务模块需要前端页面，如何让前端根据用户权限**动态显示/隐藏菜单和按钮**？

### 8.1 三层权限控制模型

```
┌─────────────────────────────────────────────────────────────┐
│  第三层: 后端 API 鉴权 (安全底线, 不可绕过)                     │
│  ───────────────────────────────────                        │
│  @require_permission("server", "restart")                   │
│  → 调 api-rbac /auth/check → 无权限返回 403                  │
│  即使前端绕过了隐藏按钮，后端也会拒绝请求                          │
├─────────────────────────────────────────────────────────────┤
│  第二层: 前端按钮显隐 (用户体验, 减少无效操作)                    │
│  ───────────────────────────────────                        │
│  v-if="hasPerm('server', 'restart')"                        │
│  → 没有权限就不渲染操作按钮，用户看不到也无法点击                   │
├─────────────────────────────────────────────────────────────┤
│  第一层: 前端菜单显隐 (导航级别控制)                             │
│  ───────────────────────────────                            │
│  v-if="hasAnyPerm('server')"                                │
│  → 用户完全没有服务器权限时，左侧菜单根本不显示"服务器管理"           │
└─────────────────────────────────────────────────────────────┘
```

### 8.2 实现原理

**步骤 1: 登录后立即获取用户全部权限**

```javascript
// 前端登录成功后，调业务后端获取权限
async function loadPermissions() {
  const resp = await fetch('/api/user/permissions', {
    headers: { 'Authorization': `Bearer ${state.token}` },
  });
  const data = await resp.json();
  // 得到: { "server": ["read","restart"], "deployment": ["read"], "alert": ["read","ack"] }
  state.permissions = data.data;
}
```

后端 `/api/user/permissions` 实际是转发调用 api-rbac 的 `/auth/menu` 接口：

```python
# app.py 中的实现
@app.route("/api/user/permissions", methods=["GET"])
def get_my_permissions():
    token = request.headers.get("Authorization", "").replace("Bearer ", "")
    perms = rbac.get_menu_via_get(token)  # ← 调 RBAC 获取权限 map
    return jsonify({"code": 0, "data": perms})
```

**步骤 2: 定义前端辅助函数**

```javascript
// 权限判断 — 整个前端只有这两个核心函数

function hasPerm(resource, action) {
  // 检查通配符 *:*、resource:*、*:action、精确匹配
  const perms = state.permissions;
  if (perms['*'] && perms['*'].includes('*')) return true;
  if (perms[resource] && perms[resource].includes('*')) return true;
  if (perms['*'] && perms['*'].includes(action)) return true;
  return perms[resource] && perms[resource].includes(action);
}

function hasAnyPerm(resource) {
  // 用于判断菜单是否显示
  return state.permissions['*'] || (state.permissions[resource]?.length > 0);
}
```

**步骤 3: 菜单根据权限动态渲染**

```javascript
// 菜单配置：哪些菜单需要什么权限
const menuConfig = [
  { key: 'servers',     label: '🖥️  服务器管理',  permission: 'server'     },
  { key: 'deployments', label: '📦 发布管理',    permission: 'deployment'  },
  { key: 'alerts',      label: '🔔 告警管理',    permission: 'alert'       },
  { key: 'permissions', label: '🔑 我的权限',    permission: null          }, // null = 所有人可见
];

function renderMenu() {
  for (const item of menuConfig) {
    // 需要权限的菜单项 → 检查用户是否至少有一个该模块的权限
    if (item.permission && !hasAnyPerm(item.permission)) {
      continue;  // ← 无权限，跳过不渲染
    }
    // 有权限 → 渲染菜单项
    navMenu.appendChild(createMenuItem(item));
  }
}
```

**步骤 4: 按钮根据权限动态显隐**

```javascript
// 加载服务器列表时，根据权限决定是否渲染操作按钮
async function loadServers() {
  const data = await fetch('/api/servers', ...);
  const canRestart = hasPerm('server', 'restart');  // ← 检查
  const canStop    = hasPerm('server', 'stop');     // ← 检查

  for (const s of data) {
    let actionsHtml = '';
    if (canRestart) actionsHtml += `<button onclick="doRestart(${s.id})">🔄 重启</button>`;
    if (canStop)    actionsHtml += `<button onclick="doStop(${s.id})">⏹️ 停止</button>`;
    // 如果两个权限都没有，这列为空，用户看不到任何操作按钮
  }
}
```

### 8.3 效果演示

**运维管理员登录后** (拥有全部 8 个权限):

```
侧边栏:                          内容区:
┌─────────────┐                 ┌─────────────────────────────────┐
│ 🚀 运维管理  │                 │ 🖥️ 服务器管理                     │
│ 👤 opsadmin  │                 │                                 │
├─────────────┤                 │ ID  名称    IP          状态   操作│
│ 🖥️ 服务器   │ ← 可见          │ 1  web-01  10.0.1.10  running 🔄 ⏹│
│ 📦 发布管理 │ ← 可见          │ 2  web-02  10.0.1.11  stopped 🔄 ⏹│
│ 🔔 告警管理 │ ← 可见          │ 3  db-01   10.0.2.10  running 🔄 ⏹│
│ 🔑 我的权限 │ ← 可见          │                                 │
└─────────────┘                 └─────────────────────────────────┘
```

**运维查看者登录后** (只有 `server:read`, `deployment:read`, `alert:read`):

```
侧边栏:                          内容区:
┌─────────────┐                 ┌─────────────────────────────────┐
│ 🚀 运维管理  │                 │ 🖥️ 服务器管理                     │
│ 👤 viewer    │                 │                                 │
├─────────────┤                 │ ID  名称    IP          状态     │
│ 🖥️ 服务器   │ ← 可见          │ 1  web-01  10.0.1.10  running     │
│ 📦 发布管理 │ ← 可见          │ 2  web-02  10.0.1.11  stopped     │
│ 🔔 告警管理 │ ← 可见          │ 3  db-01   10.0.2.10  running     │
│ 🔑 我的权限 │ ← 可见          │               (无操作按钮) ← 关键!  │
└─────────────┘                 └─────────────────────────────────┘
```

### 8.4 关键安全原则

> ⚠️ **前端显隐只是 UX 优化，真正安全的是后端**。

```
前端: hasPerm('server', 'restart') → false → 隐藏重启按钮
                           │
                           │ 用户可以通过浏览器 DevTools 手动发请求
                           │ POST /api/servers/1/restart
                           ▼
后端: @require_permission("server", "restart")
      → rbac.check_permission(token, "server", "restart")
      → false → 返回 403 ❌ 被拒绝

结论: 即使绕过前端，后端也会拦截。前端控制是"礼貌的引导"，后端是"真正的门锁"。
```

### 8.5 在不同前端框架中的等价写法

**Vue 3 (Composition API)**:

```vue
<template>
  <!-- 菜单显隐 -->
  <el-menu-item v-if="hasAnyPerm('server')" @click="navigate('servers')">
    服务器管理
  </el-menu-item>

  <!-- 按钮显隐 -->
  <el-button v-if="hasPerm('server', 'restart')" @click="doRestart">
    重启
  </el-button>
</template>

<script setup>
import { ref } from 'vue'
const permissions = ref({})

// 加载权限
onMounted(async () => {
  const resp = await fetch('/api/user/permissions', ...)
  permissions.value = (await resp.json()).data
})

const hasPerm = (r, a) => {
  const p = permissions.value
  if (p['*']?.includes('*')) return true
  if (p[r]?.includes('*')) return true
  if (p['*']?.includes(a)) return true
  return p[r]?.includes(a)
}
const hasAnyPerm = (r) => p['*'] || p[r]?.length > 0
</script>
```

**React**:

```jsx
function ServerList() {
  const { permissions } = usePermission();  // 自定义 Hook

  const hasPerm = (r, a) => { /* 同上 */ };
  const hasAnyPerm = (r) => { /* 同上 */ };

  return (
    <div>
      {/* 菜单 */}
      {hasAnyPerm('server') && <NavItem to="/servers">服务器管理</NavItem>}

      {/* 按钮 */}
      {hasPerm('server', 'restart') && <Button onClick={doRestart}>重启</Button>}
    </div>
  );
}
```

### 8.6 前端文件说明

示例项目包含一个完整可运行的前端页面 `static/index.html`：

| 文件 | 说明 |
|------|------|
| `static/index.html` | 纯 HTML+CSS+JS 的 SPA 前端 (~450行)，零依赖，直接可用 |

启动后访问 `http://localhost:5000` 即可看到完整界面。关键代码约 100 行 JS，核心就是 `hasPerm()` 和 `hasAnyPerm()` 两个函数 + `renderMenu()` 动态渲染逻辑。

---

## 9. 新增业务模块

假设要给运维系统增加**"日志管理"**模块，只需两类操作：

### 9.1 在 api-rbac 创建权限

```bash
ADMIN_TOKEN="<你的管理员 token>"

# 创建 2 个权限
curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"查看日志","resource":"log","action":"read","description":"允许查看系统日志"}'

curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"下载日志","resource":"log","action":"download","description":"允许下载日志文件"}'
```

### 9.2 在业务代码添加路由 + 装饰器

```python
# ---- 日志管理 ----

@app.route("/api/logs", methods=["GET"])
@require_permission("log", "read")       # ← 鉴权
def list_logs():
    """查看日志列表"""
    files = os.listdir("/var/log/myapp")
    return jsonify({"code": 0, "data": files})


@app.route("/api/logs/<filename>/download", methods=["GET"])
@require_permission("log", "download")   # ← 鉴权
def download_log(filename):
    """下载日志文件"""
    path = os.path.join("/var/log/myapp", filename)
    if not os.path.exists(path):
        return jsonify({"code": 404, "message": "日志不存在"}), 404
    return send_file(path)
```

**完成。** 不需要改数据库、不需要改用户表、不需要写任何权限判断逻辑。`@require_permission` 装饰器做了全部鉴权工作。

---

## 10. 安全最佳实践

### 10.1 生产环境部署

```yaml
# api-rbac config/config.yaml 生产配置要点

server:
  mode: release              # ← 关闭 Debug 输出

jwt:
  secret: "<32位以上随机字符串>"   # ← 务必修改，所有服务共享
  expire_hour: 2             # ← Access Token 短时效
  refresh_expire_day: 7      # ← Refresh Token 长时效

cors:
  allow_origins:
    - "https://ops.example.com"  # ← 改为具体域名，不要用 *

redis:
  host: 127.0.0.1
  port: 6379                  # ← 启用权限缓存，降低延迟
```

### 10.2 Token 安全

```
✅ 前端存储: localStorage (SPA) 或 httpOnly Cookie (SSR)
✅ 传输: 始终通过 HTTPS
✅ 过期: Access Token 2h，Refresh Token 7d
✅ 刷新: 前端拦截器自动用 refresh_token 换新 access_token
✅ API Key: 服务间调用用 X-API-Key 头部，不用用户 Token

❌ 不要: 将 Token 放在 URL 参数中
❌ 不要: 将 JWT secret 硬编码在业务代码中
❌ 不要: 在日志中打印完整 Token
```

### 10.3 网络隔离

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  前端 (公网)      │────→│  业务系统 (内网)   │────→│  api-rbac (内网)  │
│  :443 HTTPS      │     │  :5000 HTTP       │     │  :8087 HTTP       │
└──────────────────┘     └──────────────────┘     └──────────────────┘

- api-rbac 只监听内网地址，不直接暴露到公网
- 业务系统做反向代理/API Gateway 统一入口
- 前端永远不直接调 api-rbac
```

---

## 11. 常见问题排查

### Q1: 业务系统启动报 "权限服务异常"

```
原因: 业务系统无法连接到 api-rbac
排查:
  1. curl http://localhost:8087/health  确认 RBAC 服务正常
  2. 检查 RBAC_URL 配置是否正确
  3. 检查防火墙/网络策略
```

### Q2: 登录成功但接口返回 401

```
原因: 请求未携带 Token 或 Token 格式错误
排查:
  1. 检查 Header 格式: Authorization: Bearer <token>
  2. 确保是 "Bearer" (首字母大写)，不是 "bearer"
  3. 检查 Token 是否过期 (默认 2h)
```

### Q3: 所有操作返回 "无权限"

```
原因: 用户未被分配正确的角色
排查:
  1. 在 api-rbac 控制台查看用户绑定的角色
  2. 在 api-rbac 控制台查看角色绑定的权限
  3. 检查 resource/action 字符串是否与代码中的完全一致 (区分大小写)
```

### Q4: 新增模块后权限不生效

```
排查:
  1. 确认 api-rbac 中已创建对应的权限 (resource + action)
  2. 确认角色已绑定该权限
  3. 确认用户已分配到该角色
  4. 检查装饰器中 resource/action 与 api-rbac 中拼写完全一致
```

### Q5: Token 频繁过期需要重新登录

```
解决:
  1. 使用 refresh_token 静默刷新 (POST /api/v1/auth/refresh)
  2. 前端在 Token 过期前 (如还剩 5 分钟) 自动调用刷新接口
  3. 示例中的登录接口已返回 refresh_token
```

### Q6: 高并发下权限检查太慢

```
原因: 未启用 Redis 缓存
解决:
  1. 启动 Redis
  2. 在 config.yaml 中配置 redis 连接
  3. 重启 api-rbac，日志显示 "Redis 连接成功，权限缓存已启用"
```

---

## 附录: api-rbac 完整的鉴权 API

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/api/v1/auth/login` | POST | 无 | 登录，返回 Token 对 |
| `/api/v1/auth/refresh` | POST | 无 | 刷新 Token |
| `/api/v1/auth/introspect` | POST | 无 | Token 自省 + 可选权限检查 |
| `/api/v1/auth/check` | POST | Bearer/API-Key | 检查单个权限 |
| `/api/v1/auth/batch-check` | POST | Bearer/API-Key | 批量检查权限 |
| `/api/v1/auth/verify` | POST | Bearer | 验证 Token 有效性 |
| `/api/v1/auth/menu` | GET | Bearer | 获取用户全部权限 |
| `/health` | GET | 无 | 健康检查 |

> 完整 API 文档见项目 Swagger: `http://localhost:8087/swagger/index.html`
