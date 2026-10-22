#!/bin/bash
# Octop云端管控闭环实测: 邀请码开号→员工redeem→限权→员工调AI→记账到人
# 跑在隔离实例127.0.0.1:8099, 不碰生产8088
set -u
BASE=http://127.0.0.1:8099
PASS=$(cat ~/.octop-cloud/admin-pass.txt)
STEP() { echo; echo "===[ $1 ]==="; }

STEP "1. admin登录拿JWT"
ADMIN_JWT=$(curl -s -m 15 -X POST $BASE/api/auth/login -H 'Content-Type: application/json' \
  -d "{\"username\":\"admin\",\"password\":\"$PASS\"}" | /opt/homebrew/bin/python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])')
echo "JWT长度: ${#ADMIN_JWT}"

STEP "2. 列出角色模板"
curl -s -m 15 $BASE/api/users/roles -H "Authorization: Bearer $ADMIN_JWT" | /opt/homebrew/bin/python3 -m json.tool | head -30

STEP "3. 创建角色: 普通员工(仅experts+tasks)"
curl -s -m 15 -X POST $BASE/api/users/roles -H "Authorization: Bearer $ADMIN_JWT" -H 'Content-Type: application/json' \
  -d '{"user_role_id":"staff","user_role_name":"普通员工","description":"仅可用AI专家和任务, 无管理权限","permissions":["experts","tasks"]}' | /opt/homebrew/bin/python3 -m json.tool

STEP "4. 创建邀请码(挂staff角色, 7天有效)"
INV=$(curl -s -m 15 -X POST $BASE/api/users/invites -H "Authorization: Bearer $ADMIN_JWT" -H 'Content-Type: application/json' \
  -d '{"note":"闭环实测","expires_in_days":7,"role":"staff"}')
echo "$INV" | /opt/homebrew/bin/python3 -m json.tool
CODE=$(echo "$INV" | /opt/homebrew/bin/python3 -c 'import json,sys; print(json.load(sys.stdin)["code"])')
echo "邀请码: $CODE"

STEP "5. 员工redeem开号(模拟员工自助)"
curl -s -m 15 -X POST $BASE/api/auth/invite/redeem -H 'Content-Type: application/json' \
  -d "{\"code\":\"$CODE\",\"username\":\"xiaowang\",\"password\":\"CHANGE_ME_STAFF_PASSWORD\",\"display_name\":\"王小明\"}" > /tmp/redeem.json
/opt/homebrew/bin/python3 -c "
import json
d = json.load(open('/tmp/redeem.json'))
print('user:', d.get('user',{}).get('username'), '| role:', d.get('user',{}).get('role'))
print('员工JWT长度:', len(d.get('access_token','')))
open('/tmp/staff_jwt.txt','w').write(d['access_token'])"

STEP "6. 员工登录验证"
STAFF_JWT=$(cat /tmp/staff_jwt.txt)
curl -s -m 15 $BASE/api/auth/me -H "Authorization: Bearer $STAFF_JWT" | /opt/homebrew/bin/python3 -m json.tool | head -12

STEP "7. 员工调AI(MCP网关, call_octop_expert)"
curl -s -m 90 -X POST $BASE/api/mcp/agents -H "Authorization: Bearer $STAFF_JWT" -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"call_octop_expert","arguments":{"expert_id":"main","message":"用一句话自我介绍"}}}' | tail -c 600

STEP "8. 记账验证(usage_log按人)"
~/.octop/venv/bin/python - <<'EOF'
import sqlite3, pathlib
db = sqlite3.connect(pathlib.Path.home()/'.octop-cloud/octop.db')
rows = db.execute('SELECT user_id, agent_id, model, input_tokens, output_tokens, source, ts FROM usage_log ORDER BY id DESC LIMIT 5').fetchall()
for r in rows: print(r)
EOF

STEP "9. 权限验证: 员工碰管理接口应被拒"
curl -s -m 15 -o /dev/null -w 'GET /api/users(管理接口) -> HTTP %{http_code}\n' $BASE/api/users -H "Authorization: Bearer $STAFF_JWT"
curl -s -m 15 -o /dev/null -w 'GET /api/admin/providers -> HTTP %{http_code}\n' $BASE/api/admin/providers -H "Authorization: Bearer $STAFF_JWT"
