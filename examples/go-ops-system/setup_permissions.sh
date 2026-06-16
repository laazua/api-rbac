#!/usr/bin/env bash
# ================================================================
# 运维系统权限初始化脚本
#
# 在 api-rbac 中自动创建:
#   - 8 个运维权限 (server + deployment + alert)
#   - 2 个角色 (运维管理员 / 运维查看者)
#
# 用法:
#   1. 确保 api-rbac 已启动在 localhost:8087
#   2. chmod +x setup_permissions.sh && ./setup_permissions.sh
# ================================================================

set -e

RBAC_URL="http://localhost:8087/api/v1"
ADMIN_ACCOUNT="${ADMIN_ACCOUNT:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

echo "========================================"
echo "  运维系统权限初始化"
echo "  RBAC 服务: $RBAC_URL"
echo "========================================"
echo ""

# ================================================================
# Step 1: 管理员登录
# ================================================================
echo "📡 Step 1: 管理员登录 ..."
LOGIN_RESP=$(curl -s -X POST "$RBAC_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"account\":\"$ADMIN_ACCOUNT\",\"password\":\"$ADMIN_PASSWORD\"}")

ADMIN_TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['token'])" 2>/dev/null)

if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
  echo "❌ 管理员登录失败！请确认:"
  echo "   1. api-rbac 服务已启动在 $RBAC_URL"
  echo "   2. 管理员密码正确 (默认 Admin123456)"
  echo ""
  echo "   可设置环境变量: ADMIN_ACCOUNT=xxx ADMIN_PASSWORD=xxx"
  echo "   响应: $LOGIN_RESP"
  exit 1
fi
echo "✅ 管理员登录成功"

# ================================================================
# Step 2: 创建权限
# ================================================================
echo ""
echo "📡 Step 2: 创建运维权限 ..."

declare -A PERMS=(
  ["查看服务器"]="server:read"
  ["创建服务器"]="server:create"
  ["重启服务器"]="server:restart"
  ["停止服务器"]="server:stop"
  ["删除服务器"]="server:delete"
  ["查看发布"]="deployment:read"
  ["执行发布"]="deployment:execute"
  ["回滚发布"]="deployment:rollback"
  ["查看告警"]="alert:read"
  ["确认告警"]="alert:ack"
)

PERM_IDS=()

create_perm() {
  local name="$1"
  local resource="$2"
  local action="$3"

  RESP=$(curl -s -X POST "$RBAC_URL/permissions" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"resource\":\"$resource\",\"action\":\"$action\",\"description\":\"$name\"}")

  CODE=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',-1))" 2>/dev/null)
  if [ "$CODE" = "0" ]; then
    ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
    PERM_IDS+=("$ID")
    echo "  ✅ $name ($resource:$action) — ID=$ID"
  elif [ "$CODE" = "1006" ]; then
    echo "  ⏭️  $name 已存在, 跳过"
  else
    echo "  ⚠️  $name 创建异常: $RESP"
  fi
}

for NAME in "${!PERMS[@]}"; do
  IFS=':' read -r RESOURCE ACTION <<< "${PERMS[$NAME]}"
  create_perm "$NAME" "$RESOURCE" "$ACTION"
done

# ================================================================
# Step 3: 创建角色
# ================================================================
echo ""
echo "📡 Step 3: 创建运维角色 ..."

# 3a. 获取所有已存在的权限 ID (用于绑定)
echo "  获取所有权限列表..."
ALL_PERMS_RESP=$(curl -s "$RBAC_URL/permissions?page=1&page_size=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

# 提取运维相关权限的 ID
OPS_PERM_IDS=$(echo "$ALL_PERMS_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
perms = data.get('data', {}).get('list', [])
ids = [str(p['id']) for p in perms if p['resource'] in ('server', 'deployment', 'alert')]
print(','.join(ids))
" 2>/dev/null)

READ_PERM_IDS=$(echo "$ALL_PERMS_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
perms = data.get('data', {}).get('list', [])
ids = [str(p['id']) for p in perms if p['resource'] in ('server', 'deployment', 'alert') and p['action'] == 'read']
print(','.join(ids))
" 2>/dev/null)

if [ -z "$OPS_PERM_IDS" ]; then
  echo "  ⚠️  未找到运维权限，跳过角色创建"
else
  # 3b. 创建 "运维管理员" 角色 (拥有全部运维权限)
  echo "  创建角色: 运维管理员 (所有运维权限)..."
  ROLE_RESP=$(curl -s -X POST "$RBAC_URL/roles" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"运维管理员\",\"description\":\"拥有运维系统全部权限\"}")

  ADMIN_ROLE_ID=$(echo "$ROLE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)

  if [ -n "$ADMIN_ROLE_ID" ] && [ "$ADMIN_ROLE_ID" != "null" ]; then
    # 分配权限
    IFS=',' read -ra IDS <<< "$OPS_PERM_IDS"
    PERM_JSON="[$(echo "${IDS[*]}" | sed 's/ /,/g')]"
    curl -s -X POST "$RBAC_URL/roles/$ADMIN_ROLE_ID/permissions" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"permission_ids\":$PERM_JSON}" > /dev/null
    echo "  ✅ 运维管理员 (ID=$ADMIN_ROLE_ID) — 已绑定 ${#IDS[@]} 个权限"
  else
    echo "  ⏭️  运维管理员角色已存在"
  fi

  # 3c. 创建 "运维查看者" 角色 (仅 read 权限)
  echo "  创建角色: 运维查看者 (仅查看权限)..."
  ROLE_RESP=$(curl -s -X POST "$RBAC_URL/roles" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"运维查看者\",\"description\":\"仅有运维系统查看权限\"}")

  VIEWER_ROLE_ID=$(echo "$ROLE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)

  if [ -n "$VIEWER_ROLE_ID" ] && [ "$VIEWER_ROLE_ID" != "null" ] && [ -n "$READ_PERM_IDS" ]; then
    IFS=',' read -ra IDS <<< "$READ_PERM_IDS"
    PERM_JSON="[$(echo "${IDS[*]}" | sed 's/ /,/g')]"
    curl -s -X POST "$RBAC_URL/roles/$VIEWER_ROLE_ID/permissions" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"permission_ids\":$PERM_JSON}" > /dev/null
    echo "  ✅ 运维查看者 (ID=$VIEWER_ROLE_ID) — 已绑定 ${#IDS[@]} 个权限"
  else
    echo "  ⏭️  运维查看者角色已存在"
  fi
fi

# ================================================================
# Step 4: 创建测试用户并分配角色
# ================================================================
echo ""
echo "📡 Step 4: 创建测试用户 ..."

create_user() {
  local username="$1"
  local password="$2"
  local email="$3"
  local role_name="$4"

  # 获取角色 ID
  ROLE_RESP=$(curl -s "$RBAC_URL/roles?page=1&page_size=100" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

  ROLE_ID=$(echo "$ROLE_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
roles = data.get('data', {}).get('list', [])
for r in roles:
    if r['name'] == '$role_name':
        print(r['id'])
        break
" 2>/dev/null)

  if [ -z "$ROLE_ID" ]; then
    echo "  ⚠️  角色 '$role_name' 未找到，跳过用户 $username"
    return
  fi

  # 创建用户
  USER_RESP=$(curl -s -X POST "$RBAC_URL/users" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$username\",\"password\":\"$password\",\"email\":\"$email\"}")

  USER_ID=$(echo "$USER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)

  if [ -n "$USER_ID" ] && [ "$USER_ID" != "null" ]; then
    # 分配角色
    curl -s -X POST "$RBAC_URL/users/$USER_ID/roles" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"role_ids\":[$ROLE_ID]}" > /dev/null
    echo "  ✅ $username (密码: $password) — 已分配角色: $role_name"
  else
    echo "  ⏭️  $username 可能已存在"
    # 尝试获取已有用户并分配角色
    USER_RESP=$(curl -s "$RBAC_URL/users?page=1&page_size=100&keyword=$username" \
      -H "Authorization: Bearer $ADMIN_TOKEN")
    EXISTING_ID=$(echo "$USER_RESP" | python3 -c "
import sys, json
data = json.load(sys.stdin)
users = data.get('data', {}).get('list', [])
for u in users:
    if u['username'] == '$username':
        print(u['id'])
        break
" 2>/dev/null)
    if [ -n "$EXISTING_ID" ]; then
      curl -s -X POST "$RBAC_URL/users/$EXISTING_ID/roles" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"role_ids\":[$ROLE_ID]}" > /dev/null
      echo "  ✅ 已有用户 $username (ID=$EXISTING_ID) — 已分配角色: $role_name"
    fi
  fi
}

create_user "opsadmin" "123456" "opsadmin@example.com" "运维管理员"
create_user "opsviewer" "123456" "opsviewer@example.com" "运维查看者"

# ================================================================
# Step 5: 注册运维系统模块 (让 api-rbac 门户可以入口)
# ================================================================
echo ""
echo "📡 Step 5: 注册运维系统模块 ..."

OPS_URL="${OPS_URL:-http://localhost:8083}"

create_module() {
  local code="$1"
  local name="$2"
  local url="$3"
  local icon="$4"

  RESP=$(curl -s -X POST "$RBAC_URL/modules" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"code\":\"$code\",\"url\":\"$url\",\"icon\":\"$icon\",\"description\":\"运维管理系统 (Go+Vue)\",\"sort\":10}")

  CODE=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',-1))" 2>/dev/null)
  if [ "$CODE" = "0" ]; then
    ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
    echo "  ✅ $name (code=$code, ID=$ID)"
    echo "     入口地址: $url"
  elif [ "$CODE" = "1006" ]; then
    echo "  ⏭️  模块 '$name' 已存在"
    # 更新 URL (可能上次不同)
    MOD_ID=$(curl -s "$RBAC_URL/modules?page=1&page_size=100" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | \
      python3 -c "import sys,json; data=json.load(sys.stdin); mods=data.get('data',{}).get('list',[]); [print(m['id']) for m in mods if m['code']=='$code']" 2>/dev/null)
    if [ -n "$MOD_ID" ]; then
      curl -s -X PUT "$RBAC_URL/modules/$MOD_ID" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$url\"}" > /dev/null
      echo "  ✅ 已更新模块 '$name' 入口地址为: $url"
    fi
  else
    echo "  ⚠️  模块创建异常: $RESP"
  fi
}

create_module "ops" "运维管理系统" "$OPS_URL" "🚀"

# 获取新创建的模块 ID
OPS_MODULE_ID=$(curl -s "$RBAC_URL/modules?page=1&page_size=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  python3 -c "import sys,json; data=json.load(sys.stdin); mods=data.get('data',{}).get('list',[]); [print(m['id']) for m in mods if m['code']=='ops']" 2>/dev/null)

# 回填权限的 module_id (让角色页面按模块分组时能看到权限)
if [ -n "$OPS_MODULE_ID" ]; then
  echo ""
  echo "📡 Step 5b: 回填权限的 module_id ..."
  OPS_PERM_IDS=$(curl -s "$RBAC_URL/permissions?page=1&page_size=100" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | \
    python3 -c "
import sys, json
perms = json.load(sys.stdin)['data']['list']
ids = [str(p['id']) for p in perms if p['resource'] in ('server','deployment','alert')]
print(','.join(ids))
" 2>/dev/null)

  for PID in $(echo "$OPS_PERM_IDS" | tr ',' ' '); do
    curl -s -X PUT "$RBAC_URL/permissions/$PID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"module_id\":$OPS_MODULE_ID}" > /dev/null
  done
  echo "  ✅ 已将 ${#OPS_PERM_IDS} 个运维权限关联到模块 (ID=$OPS_MODULE_ID)"
fi

# 给角色绑定模块 (用户才能通过门户看到)
if [ -n "$OPS_MODULE_ID" ]; then
  echo ""
  echo "📡 Step 5c: 绑定模块到角色 ..."

  # 获取运维管理员角色 ID
  ADMIN_ROLE_ID=$(curl -s "$RBAC_URL/roles?page=1&page_size=100" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | \
    python3 -c "import sys,json; data=json.load(sys.stdin); roles=data.get('data',{}).get('list',[]); [print(r['id']) for r in roles if r['name']=='运维管理员']" 2>/dev/null)

  if [ -n "$ADMIN_ROLE_ID" ]; then
    curl -s -X POST "$RBAC_URL/roles/$ADMIN_ROLE_ID/modules" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"module_ids\":[$OPS_MODULE_ID]}" > /dev/null
    echo "  ✅ 运维管理员 → 已绑定运维管理系统模块"
  fi

  # 获取运维查看者角色 ID
  VIEWER_ROLE_ID=$(curl -s "$RBAC_URL/roles?page=1&page_size=100" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | \
    python3 -c "import sys,json; data=json.load(sys.stdin); roles=data.get('data',{}).get('list',[]); [print(r['id']) for r in roles if r['name']=='运维查看者']" 2>/dev/null)

  if [ -n "$VIEWER_ROLE_ID" ]; then
    curl -s -X POST "$RBAC_URL/roles/$VIEWER_ROLE_ID/modules" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"module_ids\":[$OPS_MODULE_ID]}" > /dev/null
    echo "  ✅ 运维查看者 → 已绑定运维管理系统模块"
  fi
fi

# ================================================================
# 完成
# ================================================================
echo ""
echo "========================================"
echo "  ✅ 权限初始化完成！"
echo "========================================"
echo ""
echo "测试账号 (在 api-rbac 前端登录):"
echo "  运维管理员: opsadmin / 123456 (全部运维权限)"
echo "  运维查看者: opsviewer / 123456 (仅查看权限)"
echo ""
echo "模块入口: $OPS_URL"
echo ""
echo "启动运维系统:"
echo "  cd examples/go-ops-system"
echo "  go run .                              # 后端 :8083"
echo "  cd web && npm install && npm run dev   # 前端 :5173 (开发模式)"
echo ""
echo "使用流程:"
echo "  1. 访问 api-rbac 前端 → 登录 (opsadmin)"
echo "  2. 门户页面 → 点击「运维管理系统」模块卡片"
echo "  3. 自动以 iframe 方式嵌入运维系统 (Token 自动传递)"
echo "========================================"
