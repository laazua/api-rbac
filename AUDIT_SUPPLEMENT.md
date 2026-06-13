# api-rbac 补充审计报告 — 多语言 SDK 与韧性层

> **审计日期**: 2026-06-13 | **范围**: SDK 目录 + 韧性中间件 + 示例 | **基础审计**: [AUDIT_REPORT.md](AUDIT_REPORT.md)

本报告是对主审计报告的补充，聚焦于上次审计中未覆盖的 SDK 韧性层与多语言一致性。

---

## 🔴 严重风险

### CRIT-S01: Python SDK `_extract_user_id()` 返回错误用户

**文件**: `sdk/python/rbac_client.py:233-239` | **风险**: 权限混淆

```python
def _extract_user_id(self, token: str) -> Optional[int]:
    """从 JWT token 中提取 user_id"""
    with self._lock:
        for uid in self._cache:
            return uid  # ← 返回第一个缓存的用户, 不是 token 对应的用户!
    return None
```

**影响**: 缓存中存储了多个用户的权限时，`_extract_user_id` 总是返回第一个，导致用户 A 可能匹配到用户 B 的缓存权限。**在多用户场景下，权限检查结果可能是错误的。**

**修复**: 解析 JWT payload 提取真实 user_id:
```python
import base64, json

def _extract_user_id(self, token: str) -> Optional[int]:
    try:
        # JWT payload 是第二个 segment (Base64Url)
        payload = token.split(".")[1]
        # 补齐 Base64 padding
        payload += "=" * (4 - len(payload) % 4)
        decoded = json.loads(base64.urlsafe_b64decode(payload))
        return decoded.get("user_id")
    except Exception:
        return None
```

---

### CRIT-S02: Node.js SDK 完全无韧性保护

**文件**: `sdk/nodejs/src/index.js:189-207` | **风险**: RBAC 宕机时无降级

```javascript
function permissionGuard(client, resource, action) {
  return async (req, res, next) => {
    try {
      const allowed = await client.checkPermission(token, resource, action);
      // ...
    } catch (err) {
      return res.status(502).json({ code: 1005, message: '权限检查服务不可用' });
    }
  };
}
```

**影响**: Node.js 中间件只有 fail-closed (正确), 但完全缺少:
- 本地缓存 (RBAC 宕机时所有请求返回 502)
- 熔断器 (每次请求都等待超时)
- FailMode 配置

**修复**: 为 Node.js SDK 增加韧性层:
```javascript
class ResilientRBACClient extends RBACClient {
  constructor(baseUrl, { failMode = 'DENY', cacheTTL = 300,
      cbThreshold = 5, cbRecovery = 30 } = {}) {
    super(baseUrl);
    this.failMode = failMode;
    this.cacheTTL = cacheTTL * 1000;
    this.cache = new Map();       // token → { perms, ts }
    this.failureCount = 0;
    this.circuitOpen = false;
  }

  async checkPermission(token, resource, action) {
    if (this.circuitOpen && Date.now() - this.circuitSince < this.cbRecovery * 1000) {
      return this.failMode === 'CACHE'
        ? this._checkFromCache(token, resource, action)
        : false;
    }
    try {
      const result = await super.checkPermission(token, resource, action);
      this._onSuccess(); this._populateCache(token);
      return result;
    } catch (e) {
      this._onFailure();
      if (this.failMode === 'CACHE') return this._checkFromCache(token, resource, action);
      throw e;
    }
  }
  // ... (完整实现见修复建议)
}
```

---

## 🟠 高危风险

### HIGH-S01: Java 8 SDK 缺乏韧性特性

**文件**: `sdk/java/src/main/java/com/rbac/client/RBACClient.java` | **风险**: 功能差距

Java 8 SDK 完全没有 FailMode、熔断器、本地缓存。RBAC 宕机时所有请求抛异常，由调用方决定如何处理。虽然调用方可以自己实现降级，但 SDK 应该内置。

**建议**: 为 Java 8 SDK 同步增加 `ResilientRBACClient` (因 Java 8 无 `record`，用普通 POJO):
```java
public class ResilientRBACClient extends RBACClient {
    private final FailMode failMode;
    private final Map<Long, CacheEntry> cache = new ConcurrentHashMap<>();
    // ... (参照 Java 17+ 版实现)
}
```

---

### HIGH-S02: Go ResilientGuard 每个路由创建独立缓存实例

**文件**: `pkg/client/resilient_middleware.go:31-39` | **风险**: 资源浪费 + 缓存不共享

```go
// 每个路由调用都创建独立的 resilientCache 实例
func ResilientGuard(client *RBACClient, failMode FailMode, cacheTTLSec int, resource, action string) gin.HandlerFunc {
    rc := &resilientCache{ ... }  // ← 新实例!
    return func(c *gin.Context) { ... }
}
```

**影响**: 如果 10 个路由都使用 `ResilientGuard`，就有 10 个独立的本地缓存。它们之间不共享权限数据，每个缓存都要独立地调用 `GetMenu` 来填充，浪费内存和网络请求。

**建议**: 缓存应该是全局单例:
```go
var (
    globalCache     = make(map[string]cacheEntry)
    globalCacheMu   sync.Mutex
    globalFailCount int
    globalCircuitOpen bool
    globalCircuitSince time.Time
)

func ResilientGuard(client *RBACClient, failMode FailMode, cacheTTLSec int, ...) gin.HandlerFunc {
    // 使用全局 sharedCache 而非 rc.permCache
}
```

---

### HIGH-S03: Python SDK introspection 熔断时返回假数据

**文件**: `sdk/python/rbac_client.py:158-161` | **风险**: 静默失败

```python
if self._is_circuit_open():
    return {"active": False}  # ← 静默返回 inactive, 完全无提示
```

**影响**: 熔断期间所有 introspection 请求返回 `{active: false}`，外部服务可能认为用户 Token 无效而要求重新登录。实际上 Token 有效，只是 RBAC 宕机了。

**建议**: 增加 warning 字段:
```python
if self._is_circuit_open():
    return {"active": False, "warning": "RBAC 服务不可用 (已熔断)", "degraded": True}
```

---

## 🟡 中危风险

### MED-S01: 跨 SDK FailMode 默认值不统一

| SDK | 默认 FailMode | 默认构造器 |
|-----|---------------|-----------|
| Python | `DENY` ✅ | `RBACClient(url)` → DENY |
| Java 17+ | `DENY` ✅ | `RBACClient(url)` → DENY |
| Java 8 | 无此特性 ❌ | — |
| Node.js | 无此特性 ❌ | — |
| Go | 取决于中间件调用方 | `ResilientGuard` 参数由调用方传入 |

**影响**: 用户体验不一致。Node.js/Java8 用户如果想用韧性，需要手动实现。

**建议**: 所有 SDK 统一提供:
1. `RBACClient(url)` — 默认 DENY (安全)
2. `RBACClient(url, FailMode.CACHE)` — 韧性模式
3. `setFailMode()` — 运行时切换

---

### MED-S02: Python SDK `verify()` 在 CACHE 模式下无降级

**文件**: `sdk/python/rbac_client.py:109-114`

```python
def verify(self, token: str) -> Dict:
    return self._call_or_fallback(
        lambda: self._do_verify(token),
        fallback_value=None,  # ← None 在 CACHE 也抛异常
        error_msg="Token 验证失败",
    )
```

当 `fallback_value=None` 且 `FailMode=CACHE` 时，`_call_or_fallback` 对 `None` 的处理是返回 `None`，导致 `verify()` 返回 `None` 而调用方期望 dict。Token 验证本身不适合缓存降级——如果 RBAC 宕机了，不应该相信缓存说 Token 有效。

**建议**: `verify()` 永远不走缓存，明确标注:
```python
def verify(self, token: str) -> Dict:
    """验证 Token — 始终直连 RBAC，永不降级"""
    if self._is_circuit_open():
        raise RuntimeError("Token 验证不可用: RBAC 服务不可达")
    return self._do_verify(token)
```

---

### MED-S03: Go 示例 `checkFromCache` 与 middleware 重复实现

**文件**: `examples/go-ops-system/main.go:116-140` + `pkg/client/resilient_middleware.go:177-202`

两个文件中 `checkFromCache` 的通配符匹配逻辑**完全相同**，但各写了一遍。如果权限匹配规则改变（如新增 `*:action` 匹配模式），需要改两处。

**建议**: 将 `checkFromCache` 提取为 `pkg/client` 中的公共函数，示例代码引用它。

---

### MED-S04: Python SDK 缓存 key 用 user_id 但实际按 token 查找

**文件**: `sdk/python/rbac_client.py:228-239`

设计不一致：缓存 key 是 `user_id`，但 `check_permission` 传入的是 `token`，需要从 token 提取 user_id 再查缓存。这导致每次都要解析 JWT（费时）或依赖有问题的 `_extract_user_id`。

**建议**: 统一用 `token` 前 32 位 hash 作为缓存 key，或者直接用 `token` 作为 key（Go 中间件已采用此方式）。

---

## 🔵 代码优化建议

### OPT-S01: Node.js SDK 超时处理不完善

`AbortController` 在请求完成后的 `finally` 中 `clearTimeout`，但如果 `response.json()` 也耗时（大响应），整体耗时可能超过 timeout。

### OPT-S02: Java 17+ SDK `batchCheck` 批量请求缺少上限

Java 版 batchCheck 没有像 Python 版那样限制最多 50 个权限项。虽然服务端有限制，但客户端应该提前校验避免无谓的网络请求。

### OPT-S03: 各 SDK 的 `getMenu` 命名不一致

| SDK | 方法名 |
|-----|--------|
| Go | `GetMenu` |
| Python | `get_menu` / `get_menu_via_get` |
| Node.js | `getMenu` |
| Java | `getMenu` |

Python SDK 有 `get_menu` (空实现 `pass`) 和 `get_menu_via_get` (实际实现)。应统一为 `get_menu`。

---

## ✅ 韧性层良好实践

| 实践 | SDK | 说明 |
|------|-----|------|
| Fail-closed 默认 | Python/Java17/Go | FailMode 默认为 DENY |
| 熔断恢复探测 | Python/Java17/Go | 30s 后自动尝试 |
| 登录不受缓存降级 | Python/Java17 | login 必须是实时的 |
| 缓存过期检查 | Python/Java17/Go | 超 TTL 后拒绝使用 |
| 安全拒绝兜底 | Python/Java17/Go | 缓存无数据 → 拒绝 |
| 404/403 不触发熔断 | Python SDK | `check_permission` 正确处理 code=1003 |

---

## 修复优先级 (补充)

| 阶段 | 编号 | 问题 | 工作量 |
|------|------|------|--------|
| **Week 1** | CRIT-S01 | Python `_extract_user_id` 返回错误用户 | 1h |
| | CRIT-S02 | Node.js SDK 增加韧性层 | 3h |
| | MED-S02 | Python `verify()` 永不降级 | 30min |
| | MED-S04 | Python SDK 缓存 key 策略统一 | 1h |
| **Week 2** | HIGH-S02 | Go 中间件缓存全局单例化 | 2h |
| | HIGH-S01 | Java 8 SDK 增加韧性层 | 3h |
| | HIGH-S03 | Python introspection 熔断响应 | 30min |
| | MED-S01 | 跨 SDK FailMode 默认值统一 | 1h |
| **Week 3** | MED-S03 | Go 权限匹配逻辑提取到公共函数 | 1h |
| | OPT-S02 | batchCheck 客户端上限 | 30min |
| | OPT-S03 | SDK getMenu 命名统一 | 30min |
