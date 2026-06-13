# new-api 异步媒体任务流 & 现状（源码追踪结论）

> 目的：搞清 new-api 出图/出视频的异步任务流水线，确认我们还要做什么。
> 结论：**这条流水线 new-api 已完整实现，且按秒/按分辨率计价是内建能力。多数媒体模型适配器已存在。**

## 一、异步任务流水线（已实现）

```
客户端
  │ POST /{platform}/submit/:action      (router/relay-router.go)
  ▼
controller.RelayTask                      (controller/task.go)
  │  1. TokenAuth + Distribute 选渠道(channel)
  │  2. 适配器 ValidateRequestAndSetAction
  │  3. EstimateBilling → 取倍率{seconds,size...} → 预扣额度
  │  4. BuildRequestURL/Header/Body → DoRequest → DoResponse 拿 taskID
  │  5. AdjustBillingOnSubmit（按上游返回的真实参数修正预扣）
  │  6. 存 model.Task（model/task.go）
  ▼ 立即返回 taskID
后台轮询 goroutine（main.go:136-139，定时）
  │ controller.UpdateTaskBulk / UpdateVideoTaskAll  (controller/task_video.go)
  │  按平台批量 FetchTask → ParseTaskResult → 更新任务状态
  │  到终态时 AdjustBillingOnComplete → 补扣/退款结算
  ▼
客户端 GET /{platform}/fetch/:id → 拿状态 + 产出 URL
```

对应接口（`relay/channel/adapter.go` 的 `TaskAdaptor`）：
- 校验：`ValidateRequestAndSetAction`
- **计费（内建按量）**：`EstimateBilling`（预扣倍率，如 `{"seconds":5,"size":1.666}`）、`AdjustBillingOnSubmit`、`AdjustBillingOnComplete`（终态补扣/退款）
- 请求：`BuildRequestURL/Header/Body` `DoRequest` `DoResponse`
- 轮询：`FetchTask` `ParseTaskResult`
- 输出：`ConvertToOpenAIVideo`（转 OpenAI 兼容）

## 二、已有的媒体适配器（`relay/channel/task/`）

| 适配器 | 模型 |
|------|------|
| `kling` | 快手 Kling 视频 |
| `hailuo` | Minimax 海螺 |
| `sora` | OpenAI Sora |
| `vidu` | Vidu |
| `jimeng` | 字节即梦（Seedance/Seedream）|
| `doubao` | 字节豆包 |
| `gemini` / `vertex` | Google（Veo/图像）|
| `ali` | 阿里（Wan 等）|
| `suno` | 音乐 |

> 样板：`relay/channel/task/kling/adaptor.go`（约 416 行，实现了上面整套接口）。加新模型照它写一个目录即可。

## 三、所以我们真正要做的（缩到很少）

| 事项 | 类型 | 说明 |
|------|------|------|
| 配置渠道(channel) + 填上游 Key + 定价 | **运营配置**（非编码）| 后台点一点，把已支持的模型开起来 |
| 加未覆盖模型的适配器 | 少量编码 | 仿 kling 范式，放 `relay/channel/task/<新>` |
| 产出存腾讯云 COS（7天/预签名）| 编码（可选）| new-api 默认透传上游 URL；要自托管产出才加 |
| 定价费率配置 | 运营配置 | 按张/按秒倍率（接口已支持），填费率即可 |
| 部署：镜像同步阿里云 ACR → 腾讯云 | 运维 | 沿用现有链路 |
| 品牌/前端定制 | 编码（可选）| web/ 加页面，不重写核心 |

**判断**：你的产品 ≈ **80% 运营配置 + 少量适配器/存储编码**，不再是"造流水线"。

## 四、怎么跑起来（需 Docker 的机器，本机无 Docker）

```bash
# 在服务器（x2api 同机即可）
cd new-api
cp .env.example .env        # 配 SQL_DSN(留空=SQLite 单机) / REDIS / 初始管理员等
docker compose up -d        # 上游自带 docker-compose.yml
# 访问 http://<ip>:3000 ，默认管理员见日志或 .env
docker compose logs -f new-api | grep -i 'root\|password\|初始'
```
- 单机最简：`SQL_DSN` 留空走 SQLite，连 PG 都不用。
- 要复用 x2api 的 PG/Redis：把 `SQL_DSN` / `REDIS_CONN_STRING` 指过去（参考之前的复用方案）。

> 本机这台 Mac 没装 Docker，故未在此运行；以上为可在服务器直接执行的命令。
