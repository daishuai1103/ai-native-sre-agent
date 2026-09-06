#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

ALERT_MESSAGE="Redis 容器内存使用过高，请检查并处理。"
API_URL="${API_URL:-http://localhost:8081/api/alerts}"

if ! docker ps | grep redis >/dev/null 2>&1; then
  echo "Redis container is not running."
  exit 1
fi

response="$(curl -sS -X POST "$API_URL" \
  -H 'Content-Type: application/json' \
  -d "{\"alert\":\"$ALERT_MESSAGE\"}")"

session_id="$(printf '%s' "$response" | sed -n 's/.*"session_id":"\([^"]*\)".*/\1/p')"

docker compose exec -T redis redis-cli --eval /dev/stdin <<'EOF'
for i=1,100000 do redis.call("SET","testkey_"..i,string.rep("x",1000)) end
EOF

if [ -n "$session_id" ]; then
  echo "Alert sent: $session_id"
else
  echo "Alert sent."
fi

echo "Anomaly injected: Redis memory usage spiked. Alert message: $ALERT_MESSAGE"
