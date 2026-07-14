#!/bin/bash
# Remote deployment script for new-api on the Tokyo Tencent Cloud VM.
# It intentionally keeps all new-api runtime resources separate from x2api.
set -euo pipefail

: "${IMAGE:?IMAGE must be injected by the workflow}"
: "${ACR_REGISTRY:?ACR_REGISTRY must be injected by the workflow}"
: "${ACR_USER_B64:?ACR_USER_B64 must be injected by the workflow}"
: "${ACR_PASS_B64:?ACR_PASS_B64 must be injected by the workflow}"

DEPLOY_DIR="${DEPLOY_DIR:-/opt/new-api-deploy}"
CADDY_CTR="${CADDY_CTR:-sub2api-caddy}"
CADDY_NETWORK="${CADDY_NETWORK:-sub2api-deploy_sub2api-network}"
APP_CTR="${APP_CTR:-newapi}"
DOMAIN="${DOMAIN:-www.tokone.ai}"
TS="$(date +%Y%m%d-%H%M%S)"

gen_secret() {
  head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
}

upsert_env() {
  local key="$1"
  local value="$2"
  local tmp=".env.tmp.$$"
  if [ -f .env ]; then
    awk -v k="$key" -v v="$value" '
      BEGIN { done=0 }
      $0 ~ "^" k "=" { print k "=" v; done=1; next }
      { print }
      END { if (!done) print k "=" v }
    ' .env > "$tmp"
  else
    printf '%s=%s\n' "$key" "$value" > "$tmp"
  fi
  mv "$tmp" .env
}

ensure_secret() {
  local key="$1"
  if ! grep -q "^${key}=" .env 2>/dev/null || [ -z "$(grep "^${key}=" .env | tail -1 | cut -d= -f2-)" ]; then
    upsert_env "$key" "$(gen_secret)"
  fi
}

echo ">>> Preflight checks"
docker network inspect "$CADDY_NETWORK" >/dev/null
docker inspect "$CADDY_CTR" >/dev/null

mkdir -p "$DEPLOY_DIR/data" "$DEPLOY_DIR/logs"
cd "$DEPLOY_DIR"
if [ -f docker-compose.yml ]; then
  cp docker-compose.yml "docker-compose.yml.bak.newapi.${TS}"
fi

echo ">>> Login to Aliyun ACR"
ACR_USER="$(printf '%s' "$ACR_USER_B64" | base64 -d)"
ACR_PASS="$(printf '%s' "$ACR_PASS_B64" | base64 -d)"
printf '%s' "$ACR_PASS" | docker login "$ACR_REGISTRY" --username "$ACR_USER" --password-stdin >/dev/null
unset ACR_PASS

echo ">>> Pulling $IMAGE"
docker pull "$IMAGE"

echo ">>> Writing managed new-api compose"
touch .env
chmod 600 .env
ensure_secret PG_PASSWORD
ensure_secret REDIS_PASSWORD
ensure_secret SESSION_SECRET
upsert_env NEWAPI_IMAGE "$IMAGE"
upsert_env CADDY_NETWORK "$CADDY_NETWORK"
upsert_env SESSION_COOKIE_SECURE "true"
upsert_env SESSION_COOKIE_TRUSTED_URL "https://${DOMAIN}"
chmod 600 .env

cat > docker-compose.yml <<'COMPOSE'
name: newapi

services:
  new-api:
    image: ${NEWAPI_IMAGE:?NEWAPI_IMAGE is required}
    container_name: newapi
    restart: unless-stopped
    command: --log-dir /app/logs
    expose:
      - "3000"
    volumes:
      - ./data:/data
      - ./logs:/app/logs
    environment:
      - SQL_DSN=postgresql://newapi:${PG_PASSWORD}@newapi-postgres:5432/new-api
      - REDIS_CONN_STRING=redis://:${REDIS_PASSWORD}@newapi-redis:6379
      - TZ=Asia/Shanghai
      - ERROR_LOG_ENABLED=true
      - BATCH_UPDATE_ENABLED=true
      - SESSION_SECRET=${SESSION_SECRET}
      - SESSION_COOKIE_SECURE=${SESSION_COOKIE_SECURE}
      - SESSION_COOKIE_TRUSTED_URL=${SESSION_COOKIE_TRUSTED_URL}
      - NODE_NAME=newapi-node-1
    depends_on:
      redis:
        condition: service_healthy
      postgres:
        condition: service_healthy
    networks:
      newapi-network:
      caddy-network:
        aliases:
          - newapi
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O - http://localhost:3000/api/status | grep -q '\"success\":true'"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 45s

  redis:
    image: redis:7
    container_name: newapi-redis
    restart: unless-stopped
    command: ["redis-server", "--requirepass", "${REDIS_PASSWORD}", "--save", "60", "1", "--appendonly", "yes"]
    environment:
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - REDISCLI_AUTH=${REDIS_PASSWORD}
      - TZ=Asia/Shanghai
    volumes:
      - redis_data:/data
    networks:
      - newapi-network
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a \"$${REDIS_PASSWORD}\" ping | grep -q PONG"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 5s

  postgres:
    image: postgres:15
    container_name: newapi-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: newapi
      POSTGRES_PASSWORD: ${PG_PASSWORD}
      POSTGRES_DB: new-api
      PGDATA: /var/lib/postgresql/data
      TZ: Asia/Shanghai
    volumes:
      - pg_data:/var/lib/postgresql/data
    networks:
      - newapi-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U newapi -d new-api"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

volumes:
  pg_data:
  redis_data:

networks:
  newapi-network:
    driver: bridge
  caddy-network:
    external: true
    name: ${CADDY_NETWORK}
COMPOSE

echo ">>> Starting database and redis"
docker compose up -d --wait --wait-timeout 180 redis postgres

echo ">>> Recreating new-api application"
if ! docker compose up -d --no-deps --force-recreate --wait --wait-timeout 240 new-api; then
  echo "::error::new-api container failed to become healthy"
  docker compose logs --tail=200 --no-color new-api || true
  docker compose ps || true
  exit 1
fi

echo ">>> Probe new-api container"
for i in $(seq 1 40); do
  if docker exec "$APP_CTR" sh -c "wget -q -O - http://localhost:3000/api/status | grep -q '\"success\":true'" >/dev/null 2>&1; then
    echo "new-api container API OK"
    break
  fi
  if [ "$i" = "40" ]; then
    echo "::error::new-api API probe failed"
    docker compose logs --tail=200 --no-color new-api || true
    exit 1
  fi
  sleep 3
done

echo ">>> new-api status"
docker compose ps
docker compose logs --tail=50 --no-color new-api || true

echo ">>> Locate Caddyfile from $CADDY_CTR"
HOST_CADDY="$(docker inspect "$CADDY_CTR" --format '{{range .Mounts}}{{if eq .Destination "/etc/caddy/Caddyfile"}}{{.Source}}{{end}}{{end}}')"
if [ -z "$HOST_CADDY" ] || [ ! -f "$HOST_CADDY" ]; then
  echo "::error::cannot locate host Caddyfile for $CADDY_CTR"
  exit 1
fi
CADDY_IMAGE="$(docker inspect "$CADDY_CTR" --format '{{.Config.Image}}')"
CADDY_SERVICE="$(docker inspect "$CADDY_CTR" --format '{{ index .Config.Labels "com.docker.compose.service" }}')"
CADDY_WORKDIR="$(docker inspect "$CADDY_CTR" --format '{{ index .Config.Labels "com.docker.compose.project.working_dir" }}')"
cp "$HOST_CADDY" "${HOST_CADDY}.bak.newapi.${TS}"

BEGIN="# >>> new-api ${DOMAIN} (managed by new-api deploy workflow) >>>"
END="# <<< new-api ${DOMAIN} (managed by new-api deploy workflow) <<<"
TMP_CADDY="$(mktemp)"
NEXT_CADDY="$(mktemp)"
ROLLBACK_CADDY="$(mktemp)"
trap 'rm -f "$TMP_CADDY" "$NEXT_CADDY" "$ROLLBACK_CADDY"' EXIT

docker exec "$CADDY_CTR" sh -c 'cat /etc/caddy/Caddyfile' > "$TMP_CADDY"
cp "$TMP_CADDY" "$ROLLBACK_CADDY"
awk -v begin="$BEGIN" -v end="$END" '
  $0 == begin { skip = 1; next }
  $0 == end { skip = 0; next }
  !skip { print }
' "$TMP_CADDY" > "$NEXT_CADDY"

printf '\n%s\n' "$BEGIN" >> "$NEXT_CADDY"
cat >> "$NEXT_CADDY" <<CADDYBLOCK
${DOMAIN} {
	encode zstd gzip
	reverse_proxy ${APP_CTR}:3000 {
		header_up X-Real-IP {remote_host}
		header_up X-Forwarded-For {remote_host}
		header_up X-Forwarded-Proto {scheme}
		header_up X-Forwarded-Host {host}
	}
	request_body {
		max_size 100MB
	}
	log {
		output file /var/log/caddy/newapi-www.log {
			roll_size 50mb
			roll_keep 10
			roll_keep_for 720h
		}
		format json
		level INFO
	}
}
CADDYBLOCK
printf '%s\n' "$END" >> "$NEXT_CADDY"

echo ">>> Validate Caddyfile"
if ! docker run --rm --entrypoint caddy -v "$NEXT_CADDY:/etc/caddy/Caddyfile:ro" "$CADDY_IMAGE" validate --adapter caddyfile --config /etc/caddy/Caddyfile; then
  echo "::error::Caddyfile invalid, rolling back"
  cp "$ROLLBACK_CADDY" "$HOST_CADDY"
  exit 1
fi
cp "$NEXT_CADDY" "$HOST_CADDY"

echo ">>> Apply Caddyfile"
if docker exec -i "$CADDY_CTR" sh -c 'cat > /etc/caddy/Caddyfile' < "$NEXT_CADDY"; then
  echo ">>> Reload Caddy"
  docker exec "$CADDY_CTR" caddy reload --adapter caddyfile --config /etc/caddy/Caddyfile
else
  echo ">>> Caddyfile mount is not writable in-place; recreating Caddy service"
  if [ -z "$CADDY_SERVICE" ] || [ -z "$CADDY_WORKDIR" ] || [ ! -d "$CADDY_WORKDIR" ]; then
    echo "::error::cannot locate Caddy compose service/workdir for recreate"
    cp "$ROLLBACK_CADDY" "$HOST_CADDY"
    exit 1
  fi
  (
    cd "$CADDY_WORKDIR"
    docker compose up -d --no-deps --force-recreate "$CADDY_SERVICE"
  )
fi
docker exec "$CADDY_CTR" caddy validate --adapter caddyfile --config /etc/caddy/Caddyfile

echo ">>> Probe Caddy network to new-api"
docker exec "$CADDY_CTR" sh -c "wget -q -O - -T 5 http://${APP_CTR}:3000/api/status | grep -q '\"success\":true' && echo 'caddy->new-api OK'"
curl -skf --resolve "${DOMAIN}:443:127.0.0.1" "https://${DOMAIN}/api/status" -o /dev/null -w "${DOMAIN} via caddy -> %{http_code}\n" || true

docker image prune -f >/dev/null 2>&1 || true
echo ">>> Deploy OK"
