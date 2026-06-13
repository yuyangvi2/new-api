# N1 — 服务器部署 new-api（跑通）

目标：在服务器（x2api 同机即可）把 new-api 跑起来、能登录后台。
本机 Mac 无 Docker，以下命令在**服务器**执行。

## 方案 A：独立 PG/Redis（推荐，最快跑通、零耦合）

```bash
# 1. 拉代码（首次）
cd ~ && git clone git@github.com:yuyangvi2/new-api.git   # 或已 clone 则 git pull
cd new-api/custom/deploy

# 2. 配置
cp .env.example .env
openssl rand -hex 32          # 把输出填到 .env 的 SESSION_SECRET
vim .env                      # 改 PG_PASSWORD / REDIS_PASSWORD / SESSION_SECRET

# 3. 启动
docker compose -f docker-compose.server.yml up -d

# 4. 看状态 / 找初始管理员
docker compose -f docker-compose.server.yml ps
docker compose -f docker-compose.server.yml logs -f newapi | grep -iE 'root|password|初始|listen'
```

- 访问：`http://<服务器IP>:3000`
- 首次默认管理员（one-api 系）：用户名 `root`，密码 `123456` → **登录后立刻改密码**。若不对则查上面日志。
- 端口 3000 与 x2api(8080) 不冲突；要走反代/域名，把 compose 端口改 `127.0.0.1:3000:3000` 再用 nginx 转发。

## 方案 B：复用 x2api 的 PG/Redis（省两个容器，耦合 x2api）

1. 在 x2api 的 PG 建库：
   ```bash
   docker exec -it sub2api-postgres psql -U sub2api -c "CREATE DATABASE \"new-api\";"
   ```
2. 让 newapi 容器接入 x2api 网络（`docker network ls | grep sub2api` 确认网络名），
   并把 compose 里的 `SQL_DSN` / `REDIS_CONN_STRING` 指向 x2api 的 `postgres` / `redis` 服务名、Redis 用不同 db（如 `/3`）。
3. 删掉本 compose 里的 postgres/redis 服务，networks 改 external 接 x2api 网络。
> 细节同之前 `wavespeed/aggregator/docker-compose.reuse.yml` 的思路。先跑通建议用方案 A。

## 验收（N1 完成标准）
- [ ] `docker compose ... ps` 三个容器 healthy/running
- [ ] 浏览器能打开 `http://<IP>:3000` 后台并用 root 登录
- [ ] 改掉默认密码

## 常见问题
- **拉镜像慢/失败**：x2api 能跑说明服务器能拉 Docker Hub；若慢，给 Docker daemon 配镜像加速器，或等 N7 把镜像同步到阿里云 ACR 后改 `NEWAPI_IMAGE`。
- **3000 被占**：改 `.env` 的 `NEWAPI_PORT`。
- **数据持久化**：数据在 `pg_data` 卷 + `./data`、`./logs` 目录，别误删。
