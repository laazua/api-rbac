"""
RBAC 权限管理系统 - Python SDK (韧性增强版)

兼容 Python 3.8+, 零外部依赖。

核心特性:
  - 故障降级: RBAC 不可达时，使用本地缓存权限 (FailMode.CACHE)
  - 熔断保护: 连续失败超过阈值后，直接走本地缓存
  - 安全默认: 默认 FailMode.DENY, 故障时拒绝所有请求
  - 自动恢复: 熔断后定期探测 RBAC, 恢复后切回正常模式

Usage:
    from rbac_client import RBACClient, FailMode

    client = RBACClient("http://localhost:8087/api/v1",
                        fail_mode=FailMode.CACHE,
                        cache_ttl=300)  # 缓存 5 分钟

    # 正常使用, 故障时自动降级到本地缓存
    allowed = client.check_permission(token, "user", "delete")
"""

import base64
import json
import time
import threading
import urllib.request
import urllib.error
from enum import Enum
from typing import Any, Dict, List, Optional, Tuple


class FailMode(Enum):
    """
    故障模式 — RBAC 服务不可达时的行为。

    DENY  (默认, 安全) : 拒绝所有权限检查请求。适合安全敏感场景。
    CACHE (推荐, 韧性): 使用本地缓存的权限数据。适合需要高可用的场景。
    """
    DENY = "deny"
    CACHE = "cache"


class RBACClient:
    """RBAC 服务 HTTP 客户端 (韧性增强版)"""

    def __init__(
        self,
        base_url: str,
        timeout: int = 5,
        fail_mode: FailMode = FailMode.DENY,
        cache_ttl: int = 300,
        circuit_breaker_threshold: int = 5,
        circuit_breaker_recovery: int = 30,
    ):
        """
        Args:
            base_url: RBAC 服务地址
            timeout: 每次 HTTP 请求超时 (秒)
            fail_mode: 故障模式 (DENY=拒绝所有 / CACHE=使用本地缓存)
            cache_ttl: 本地缓存有效期 (秒)，超过后视为过期
            circuit_breaker_threshold: 连续失败多少次后触发熔断
            circuit_breaker_recovery: 熔断后多少秒后尝试恢复
        """
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.fail_mode = fail_mode
        self.cache_ttl = cache_ttl
        self.cb_threshold = circuit_breaker_threshold
        self.cb_recovery = circuit_breaker_recovery

        # ---- 本地缓存 ----
        # 格式: { user_id: (permissions_map, timestamp) }
        # permissions_map: {"resource": ["action1", "action2"], ...}
        self._cache: Dict[int, Tuple[Dict[str, List[str]], float]] = {}
        self._lock = threading.Lock()

        # ---- 熔断器状态 ----
        self._failure_count = 0
        self._circuit_open = False
        self._circuit_opened_at = 0.0

    # ================================================================
    # 公共 API (不变)
    # ================================================================

    def login(self, account: str, password: str) -> Dict:
        resp = self._post("/auth/login", {"account": account, "password": password})
        if resp.get("code") != 0:
            raise RuntimeError(resp.get("message", "登录失败"))
        data = resp["data"]

        # 登录成功后, 立即加载用户权限到本地缓存
        try:
            perms = self.get_menu_via_get(data["token"])
            user_id = data["user_id"]
            with self._lock:
                self._cache[user_id] = (perms, time.time())
        except Exception:
            pass  # 缓存失败不影响登录

        return data

    def refresh(self, refresh_token: str) -> Dict:
        resp = self._post("/auth/refresh", {"refresh_token": refresh_token})
        if resp.get("code") != 0:
            raise RuntimeError(resp.get("message", "刷新失败"))
        return resp["data"]

    def verify(self, token: str) -> Dict:
        """验证 Token — 始终直连 RBAC，永不降级到缓存"""
        if self._is_circuit_open():
            raise RuntimeError("Token 验证不可用: RBAC 服务不可达 (已熔断)")
        try:
            result = self._do_verify(token)
            self._on_success()
            return result
        except Exception:
            self._on_failure()
            raise

    def check_permission(self, token: str, resource: str, action: str) -> bool:
        """检查用户是否有指定权限 (故障时走本地缓存或拒绝)"""
        result = self._call_or_fallback(
            lambda: self._do_check_permission(token, resource, action),
            fallback_value=None,  # None 表示走本地缓存
            error_msg="权限检查失败",
            token=token,
            resource=resource,
            action=action,
        )
        if isinstance(result, bool):
            return result
        # result 为 None → 走 fallback
        return self._check_from_cache(token, resource, action)

    def batch_check(self, token: str, permissions: List[Tuple[str, str]]) -> Dict[str, bool]:
        result = self._call_or_fallback(
            lambda: self._do_batch_check(token, permissions),
            fallback_value=None,
            error_msg="批量权限检查失败",
            token=token,
            permissions=permissions,
        )
        if isinstance(result, dict):
            return result
        # fallback: 逐个从缓存检查
        results = {}
        for res, act in permissions:
            results[f"{res}:{act}"] = self._check_from_cache(token, res, act)
        return results

    def introspect(self, token: str, resource: str = "", action: str = "") -> Dict:
        body = {"token": token}
        if resource:
            body["resource"] = resource
        if action:
            body["action"] = action
        try:
            resp = self._post("/auth/introspect", body)
            self._on_success()
            return resp.get("data", {"active": False})
        except Exception as e:
            self._on_failure()
            # introspect 不走缓存, 直接返回 inactive
            if self._is_circuit_open():
                return {"active": False}
            raise RuntimeError(f"Token 自省失败: {e}")

    def get_menu(self, token: str) -> Dict[str, List[str]]:
        result = self._call_or_fallback(
            lambda: self._do_get_menu(token),
            fallback_value=None,
            error_msg="获取菜单失败",
            token=token,
        )
        if isinstance(result, dict):
            return result
        return self._get_cached_permissions(token) or {}

    def get_menu_via_get(self, token: str) -> Dict[str, List[str]]:
        return self.get_menu(token)

    # ================================================================
    # 韧性层 — 核心
    # ================================================================

    def _call_or_fallback(
        self,
        callable_fn,
        fallback_value,
        error_msg: str,
        **cache_keys,
    ):
        """
        核心韧性方法:
          1. 尝试调 RBAC API
          2. 成功 → 记录成功 + 更新缓存
          3. 失败 → 记录失败 + 检查熔断状态
          4. 熔断或 FailMode=CACHE → 走本地缓存
          5. FailMode=DENY 且无缓存 → 抛异常 (安全拒绝)
        """
        # 熔断状态下直接走 fallback, 避免每次都超时等待
        if self._is_circuit_open():
            if self.fail_mode == FailMode.DENY:
                raise RuntimeError(f"{error_msg}: RBAC 服务不可用 (已熔断)")
            if self.fail_mode == FailMode.CACHE and fallback_value is None:
                return None  # 触发 _check_from_cache
            return fallback_value

        try:
            result = callable_fn()
            self._on_success()
            return result
        except Exception:
            self._on_failure()

            if self.fail_mode == FailMode.DENY:
                raise RuntimeError(f"{error_msg}: RBAC 服务不可用")

            # FailMode.CACHE
            if fallback_value is None:
                return None  # 触发本地缓存路径
            return fallback_value

    def _check_from_cache(self, token: str, resource: str, action: str) -> bool:
        """从本地缓存中检查权限"""
        cached = self._get_cached_permissions(token)
        if cached is None:
            # 缓存中没有 → 安全拒绝
            return False
        return _match_permission(cached, resource, action)

    def _get_cached_permissions(self, token: str) -> Optional[Dict[str, List[str]]]:
        """根据 token 从本地缓存获取权限"""
        # 用 token 的 hash 作为 key (token 中包含 user_id)
        # 更好的方式: 业务系统传入 user_id 或用 token 前缀
        user_id = self._extract_user_id(token)
        if user_id is None:
            return None
        with self._lock:
            entry = self._cache.get(user_id)
            if entry is None:
                return None
            perms, ts = entry
            if time.time() - ts > self.cache_ttl:
                return None  # 缓存过期
            return perms

    def _extract_user_id(self, token: str) -> Optional[int]:
        """从 JWT payload 中提取 user_id (Base64 解码, 无需验证签名)"""
        try:
            # JWT 格式: header.payload.signature
            parts = token.split(".")
            if len(parts) < 2:
                return None
            # Base64Url 解码 payload
            payload = parts[1]
            payload += "=" * (4 - len(payload) % 4)  # 补齐 padding
            decoded = json.loads(base64.urlsafe_b64decode(payload))
            return decoded.get("user_id")
        except Exception:
            return None

    # ---- 熔断器 ----

    def _on_success(self):
        self._failure_count = 0
        self._circuit_open = False

    def _on_failure(self):
        self._failure_count += 1
        if self._failure_count >= self.cb_threshold:
            self._circuit_open = True
            self._circuit_opened_at = time.time()

    def _is_circuit_open(self) -> bool:
        """熔断状态下, 检查是否到了恢复探测时间"""
        if not self._circuit_open:
            return False
        # 到达恢复时间 → 尝试半开状态 (让下一次请求通过)
        if time.time() - self._circuit_opened_at > self.cb_recovery:
            self._circuit_open = False
            self._failure_count = 0
            return False
        return True

    # ================================================================
    # 私有 — HTTP 调用
    # ================================================================

    def _do_check_permission(self, token: str, resource: str, action: str) -> bool:
        headers = {"Authorization": f"Bearer {token}"}
        resp = self._post("/auth/check", {"resource": resource, "action": action}, headers)
        if resp.get("code") != 0:
            if resp.get("code") == 1003:
                return False
            raise RuntimeError(resp.get("message", "权限检查失败"))
        return resp.get("data", {}).get("allowed", False)

    def _do_batch_check(self, token: str, permissions: list) -> Dict[str, bool]:
        headers = {"Authorization": f"Bearer {token}"}
        items = [{"resource": r, "action": a} for r, a in permissions]
        resp = self._post("/auth/batch-check", {"permissions": items}, headers)
        if resp.get("code") != 0:
            raise RuntimeError(resp.get("message", "批量检查失败"))
        return resp.get("data", {}).get("results", {})

    def _do_verify(self, token: str) -> Dict:
        headers = {"Authorization": f"Bearer {token}"}
        resp = self._post("/auth/verify", headers=headers)
        if resp.get("code") != 0:
            raise RuntimeError(resp.get("message", "Token 无效"))
        return resp["data"]

    def _do_get_menu(self, token: str) -> Dict[str, List[str]]:
        import urllib.request
        url = f"{self.base_url}/auth/menu"
        req = urllib.request.Request(url, headers={"Authorization": f"Bearer {token}"})
        with urllib.request.urlopen(req, timeout=self.timeout) as resp:
            result = json.loads(resp.read().decode("utf-8"))
        if result.get("code") != 0:
            raise RuntimeError(result.get("message", "获取菜单失败"))
        perms = result.get("data", {}).get("permissions", {})
        # 更新缓存
        user_id = self._extract_user_id(token)
        if user_id and perms:
            with self._lock:
                self._cache[user_id] = (perms, time.time())
        return perms

    def _post(self, path: str, data: Any = None, headers: Optional[Dict[str, str]] = None) -> Dict:
        url = f"{self.base_url}{path}"
        req_headers = {"Content-Type": "application/json"}
        if headers:
            req_headers.update(headers)

        body = json.dumps(data).encode("utf-8") if data else b""
        req = urllib.request.Request(url, data=body, headers=req_headers, method="POST")

        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            try:
                return json.loads(e.read().decode("utf-8"))
            except Exception:
                raise RuntimeError(f"HTTP {e.code}: {e.reason}")
        except urllib.error.URLError as e:
            raise RuntimeError(f"请求失败: {e.reason}")


# ================================================================
# 权限匹配 (与 api-rbac 服务端逻辑一致)
# ================================================================

def _match_permission(perms: Dict[str, List[str]], resource: str, action: str) -> bool:
    """根据权限 map 判断是否有权限 (支持通配符)"""
    if not perms:
        return False
    # *:*
    if perms.get("*") and ("*" in perms["*"] or action in perms["*"]):
        return True
    # resource:*
    if perms.get(resource) and ("*" in perms[resource] or action in perms[resource]):
        return True
    return False
