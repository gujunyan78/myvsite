#!/usr/bin/env bash
# VSITE end-to-end API test script
# Usage: bash scripts/test_api.sh [base_url]
set -u
BASE="${1:-http://127.0.0.1:8080}"
pass=0; fail=0
green() { printf '\033[32m[PASS]\033[0m %s\n' "$1"; pass=$((pass+1)); }
red()   { printf '\033[31m[FAIL]\033[0m %s\n' "$1"; fail=$((fail+1)); }

echo "==== VSITE API test @ $BASE ===="

# 1. Login with default ADMIN/123456
login=$(curl -s -X POST "$BASE/api/admin/login" -H 'Content-Type: application/json' \
  -d '{"username":"ADMIN","password":"123456"}')
code=$(echo "$login" | python3 -c "import sys,json;print(json.load(sys.stdin).get('code'))")
if [ "$code" = "0" ]; then green "admin login"; else red "admin login (resp: $login)"; fi
token=$(echo "$login" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('token',''))")
[ -n "$token" ] && green "got token" || red "no token returned"

# 2. First login must be blocked until password change
who=$(curl -s "$BASE/api/admin/profile" -H "X-Admin-Token: $token")
who_code=$(echo "$who" | python3 -c "import sys,json;print(json.load(sys.stdin).get('code'))")
[ "$who_code" = "403" ] && green "first login blocked until password change" || red "forced password change (code=$who_code)"

# 3. Change default password
chg=$(curl -s -X POST "$BASE/api/admin/password" -H "X-Admin-Token: $token" -H 'Content-Type: application/json' \
  -d '{"old_password":"123456","new_password":"vsite888","username":"ADMIN"}')
chg_code=$(echo "$chg" | python3 -c "import sys,json;print(json.load(sys.stdin).get('code'))")
[ "$chg_code" = "0" ] && green "change default password" || red "change default password (resp: $chg)"

# 4. Login again with the new password
login2=$(curl -s -X POST "$BASE/api/admin/login" -H 'Content-Type: application/json' \
  -d '{"username":"ADMIN","password":"vsite888"}')
token=$(echo "$login2" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('token',''))")
[ -n "$token" ] && green "login with new password" || red "login with new password"

# 5. Access protected API
who=$(curl -s "$BASE/api/admin/profile" -H "X-Admin-Token: $token")
who_code=$(echo "$who" | python3 -c "import sys,json;print(json.load(sys.stdin).get('code'))")
[ "$who_code" = "0" ] && green "access admin profile API" || red "access admin profile API"

# 6. Create creator
blogger=$(curl -s -X POST "$BASE/api/admin/bloggers" -H "X-Admin-Token: $token" -H 'Content-Type: application/json' \
  -d '{"account":"Test Creator 001","description":"Creator created by automated test"}')
bid=$(echo "$blogger" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('id',''))")
[ -n "$bid" ] && green "create creator (id=$bid)" || red "create creator (resp: $blogger)"

# 7. Create video
video=$(curl -s -X POST "$BASE/api/admin/videos" -H "X-Admin-Token: $token" -H 'Content-Type: application/json' \
  -d "{\"blogger_id\":$bid,\"title\":\"Test video title\",\"video_url\":\"https://example.com/test.mp4\",\"cover_url\":\"https://example.com/cover.jpg\",\"description\":\"Test description\"}")
vid=$(echo "$video" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('id',''))")
[ -n "$vid" ] && green "create video (id=$vid)" || red "create video (resp: $video)"

# 8. Public video list
list=$(curl -s "$BASE/api/videos?page_size=3&client_id=test_cid")
total=$(echo "$list" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('total'))")
[ -n "$total" ] && green "public video list (total=$total)" || red "public video list"

# 9. Creator page videos
blist=$(curl -s "$BASE/api/videos?blogger_id=$bid&page_size=5" )
bitems=$(echo "$blist" | python3 -c "import sys,json;print(len(json.load(sys.stdin).get('data',{}).get('items',[])))")
[ "$bitems" -ge 1 ] 2>/dev/null && green "creator video list (count=$bitems)" || red "creator video list (count=$bitems)"

# 10. Like
like=$(curl -s -X POST "$BASE/api/videos/$vid/like" -H 'Content-Type: application/json' -d '{"client_id":"test_cid"}')
liked=$(echo "$like" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('liked'))")
[ "$liked" = "True" ] && green "like video" || red "like video (liked=$liked)"

# 11. Post comment
cmt=$(curl -s -X POST "$BASE/api/videos/$vid/comments" -H 'Content-Type: application/json' -d '{"nickname":"TestUser","content":"This is a test comment"}')
cid=$(echo "$cmt" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('id',''))")
[ -n "$cid" ] && green "post comment (id=$cid)" || red "post comment (resp: $cmt)"

# 12. Public comment list
clist=$(curl -s "$BASE/api/videos/$vid/comments?page=1&page_size=10")
citems=$(echo "$clist" | python3 -c "import sys,json;print(len(json.load(sys.stdin).get('data',{}).get('items',[])))")
[ "$citems" -ge 1 ] 2>/dev/null && green "comment list (count=$citems)" || red "comment list"

# 13. Admin stats
stats=$(curl -s "$BASE/api/admin/stats" -H "X-Admin-Token: $token")
sv=$(echo "$stats" | python3 -c "import sys,json;print(json.load(sys.stdin).get('data',{}).get('videos'))")
[ -n "$sv" ] && green "admin stats (videos=$sv)" || red "admin stats"

# 14. Pages reachable
idx=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
adm=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/admin")
[ "$idx" = "200" ] && green "home page reachable" || red "home page (code=$idx)"
[ "$adm" = "200" ] && green "admin page reachable" || red "admin page (code=$adm)"

# Cleanup test data
curl -s -X DELETE "$BASE/api/admin/videos/$vid" -H "X-Admin-Token: $token" >/dev/null
curl -s -X DELETE "$BASE/api/admin/bloggers/$bid" -H "X-Admin-Token: $token" >/dev/null
echo "==== Result: passed $pass, failed $fail ===="
[ "$fail" -eq 0 ] && exit 0 || exit 1
