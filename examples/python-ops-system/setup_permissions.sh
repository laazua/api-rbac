#!/bin/bash
# ============================================================
# 运维管理系统 — 权限初始化脚本
#
# 在 api-rbac 中创建运维管理系统所需的权限和角色。
# 运行前请确保 api-rbac 服务已启动在 localhost:8087。
# ============================================================

set -e

RBAC_URL="http://localhost:8087/api/v1"
ADMIN_TOKEN=""

# 1. 登录获取管理员 Token
echo "=== 1. 登录超级管理员 ==="
RESP=$(curl -s -X POST "$RBAC_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"account":"admin","password":"Admin123456"}')
echo "$RESP" | python3 -m json.tool

ADMIN_TOKEN=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
echo "Token: ${ADMIN_TOKEN:0:20}..."

AUTH="-H \"Authorization: Bearer $ADMIN_TOKEN\" -H \"Content-Type: application/json\""

# 2. 创建权限
echo ""
echo "=== 2. 创建运维系统所需权限 ==="

declare -A PERMS=(
  # 服务器管理
  ["server:read"]="查看服务器|允许查看服务器列表和详情"
  ["server:restart"]="重启服务器|允许重启服务器"
  ["server:stop"]="停止服务器|允许停止服务器"

  # 发布管理
  ["deployment:read"]="查看发布|允许查看发布记录"
  ["deployment:execute"]="执行发布|允许执行发布操作"
  ["deployment:rollback"]="回滚发布|允许回滚发布"

  # 告警管理
  ["alert:read"]="查看告警|允许查看告警列表"
  ["alert:ack"]="确认告警|允许确认/处理告警"
)

for key in "${!PERMS[@]}"; do
  name=$(echo "${PERMS[$key]}" | cut -d'|' -f1)
  desc=$(echo "${PERMS[$key]}" | cut -d'|' -f2)
  resource=$(echo "$key" | cut -d':' -f1)
  action=$(echo "$key" | cut -d':' -f2)

  curl -s -X POST "$RBAC_URL/permissions" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"resource\":\"$resource\",\"action\":\"$action\",\"description\":\"$desc\"}" \
    | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  ✅ {d[\"data\"][\"name\"]} ({resource}:{action})')"
done

# 3. 创建角色
echo ""
echo "=== 3. 创建角色并绑定权限 ==="

# 获取所有权限 ID
PERM_IDS=$(curl -s "$RBAC_URL/permissions?page=1&page_size=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  python3 -c "import sys,json; ids=[str(p['id']) for p in json.load(sys.stdin)['data']['list']]; print(','.join(ids))")

# 运维管理员角色 — 绑定所有运维权限
curl -s -X POST "$RBAC_URL/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"运维管理员","description":"拥有运维系统的全部操作权限"}' > /dev/null

# 获取刚创建的角色 ID（假设为角色列表最后一个）
ROLE_ID=$(curl -s "$RBAC_URL/roles?page=1&page_size=1" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['list'][0]['id'])")

curl -s -X POST "$RBAC_URL/roles/$ROLE_ID/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"permission_ids\":[$PERM_IDS]}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  ✅ 运维管理员角色已创建 (ID: $ROLE_ID), 绑定 {len(json.loads(\"[$PERM_IDS]\"))} 个权限')"

# 只读角色
curl -s -X POST "$RBAC_URL/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"运维查看者","description":"只能查看运维系统信息，不能操作"}' > /dev/null

# 获取只读角色的 ID
VIEWER_ID=$(curl -s "$RBAC_URL/roles?page=1&page_size=1" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['list'][0]['id'])")

# 只绑定 read 权限
READ_IDS=$(curl -s "$RBAC_URL/permissions?page=1&page_size=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  python3 -c "import sys,json; ids=[str(p['id']) for p in json.load(sys.stdin)['data']['list'] if p['action']=='read']; print(','.join(ids))")

curl -s -X POST "$RBAC_URL/roles/$VIEWER_ID/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"permission_ids\":[$READ_IDS]}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  ✅ 运维查看者角色已创建 (ID: $VIEWER_ID)')"

echo ""
echo "========================================"
echo "  ✅ 权限初始化完成！"
echo ""
echo "  创建的角色:"
echo "    - 运维管理员 (全部运维权限)"
echo "    - 运维查看者 (仅 read 权限)"
echo ""
echo "  给用户分配角色:"
echo "    curl -X POST $RBAC_URL/users/<user_id>/roles \\"
echo "      -H 'Authorization: Bearer $ADMIN_TOKEN' \\"
echo "      -d '{\"role_ids\":[$ROLE_ID]}'"
echo "========================================"
