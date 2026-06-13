#!/bin/bash
# ============================================================
# 运维管理系统 (Java) — 权限初始化脚本
# 在 api-rbac 中创建所需权限和角色
# ============================================================
set -e

RBAC_URL="${RBAC_URL:-http://localhost:8087/api/v1}"

echo "========================================="
echo "  运维管理系统 — 权限初始化"
echo "  RBAC: $RBAC_URL"
echo "========================================="

# 1. 登录
echo ""
echo "=== 1. 登录 ==="
read -sp "管理员密码: " ADMIN_PASS
echo ""

RESP=$(curl -s -X POST "$RBAC_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"account\":\"admin\",\"password\":\"$ADMIN_PASS\"}")

CODE=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','-1'))" 2>/dev/null || echo "-1")
if [ "$CODE" != "0" ]; then
  echo "❌ 登录失败: $RESP"
  exit 1
fi

TOKEN=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
echo "✅ 登录成功"

AUTH="Authorization: Bearer $TOKEN"

# 2. 创建权限
echo ""
echo "=== 2. 创建权限 ==="

create_perm() {
  local name="$1" resource="$2" action="$3" desc="$4"
  local resp=$(curl -s -X POST "$RBAC_URL/permissions" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"resource\":\"$resource\",\"action\":\"$action\",\"description\":\"$desc\"}")
  local code=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',-1))" 2>/dev/null)
  if [ "$code" = "0" ]; then
    echo "  ✅ $name ($resource:$action)"
  elif [ "$code" = "1006" ]; then
    echo "  ⏭️  $name 已存在, 跳过"
  else
    echo "  ❌ $name 创建失败: $resp"
  fi
}

create_perm "查看服务器"  "server"     "read"     "允许查看服务器列表和详情"
create_perm "重启服务器"  "server"     "restart"  "允许重启服务器"
create_perm "停止服务器"  "server"     "stop"     "允许停止服务器"
create_perm "查看发布"    "deployment" "read"     "允许查看发布记录"
create_perm "执行发布"    "deployment" "execute"  "允许执行发布操作"
create_perm "回滚发布"    "deployment" "rollback" "允许回滚发布"
create_perm "查看告警"    "alert"      "read"     "允许查看告警列表"
create_perm "确认告警"    "alert"      "ack"      "允许确认/处理告警"

# 3. 创建角色并绑定权限
echo ""
echo "=== 3. 创建角色 ==="

PERM_IDS=$(curl -s "$RBAC_URL/permissions?page=1&page_size=50" -H "$AUTH" | \
  python3 -c "import sys,json; print(','.join(str(p['id']) for p in json.load(sys.stdin)['data']['list']))")

# 运维管理员
curl -s -X POST "$RBAC_URL/roles" -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"运维管理员","description":"拥有运维系统的全部操作权限"}' > /dev/null 2>&1 || true

sleep 0.5
ROLE_ID=$(curl -s "$RBAC_URL/roles?page=1&page_size=1" -H "$AUTH" | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['list'][0]['id'])")

curl -s -X POST "$RBAC_URL/roles/$ROLE_ID/permissions" -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"permission_ids\":[$PERM_IDS]}" > /dev/null 2>&1
echo "  ✅ 运维管理员 (全部权限) ID=$ROLE_ID"

# 运维查看者
curl -s -X POST "$RBAC_URL/roles" -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"运维查看者","description":"只能查看运维系统信息"}' > /dev/null 2>&1 || true

sleep 0.5
VIEWER_ID=$(curl -s "$RBAC_URL/roles?page=1&page_size=1" -H "$AUTH" | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['list'][0]['id'])")

READ_IDS=$(curl -s "$RBAC_URL/permissions?page=1&page_size=50" -H "$AUTH" | \
  python3 -c "import sys,json; print(','.join(str(p['id']) for p in json.load(sys.stdin)['data']['list'] if p['action']=='read'))")

curl -s -X POST "$RBAC_URL/roles/$VIEWER_ID/permissions" -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"permission_ids\":[$READ_IDS]}" > /dev/null 2>&1
echo "  ✅ 运维查看者 (仅 read) ID=$VIEWER_ID"

echo ""
echo "========================================="
echo "  ✅ 权限初始化完成"
echo ""
echo "  创建的角色:"
echo "    运维管理员 (ID=$ROLE_ID) — 全部运维权限"
echo "    运维查看者 (ID=$VIEWER_ID) — 仅 read 权限"
echo ""
echo "  给用户分配角色示例:"
echo "    curl -X POST $RBAC_URL/users/<user_id>/roles \\"
echo "      -H 'Authorization: Bearer $TOKEN' \\"
echo "      -d '{\"role_ids\":[$ROLE_ID]}'"
echo "========================================="
