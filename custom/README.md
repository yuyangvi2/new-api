# custom/ — 媒体生成定制（基于 new-api fork）

本目录是我们在 `QuantumNous/new-api` fork 上叠加的**媒体生成（图/视频/音频）聚合中转**定制的命名空间与文档。
代码层面的定制尽量"只增不改"，按 new-api 范式落到上游目录里；本目录放文档、脚本、规划。

- [SYNC-UPSTREAM.md](./SYNC-UPSTREAM.md) — 跟随上游更新的操作手册与减冲突纪律
- [sync-upstream.sh](./sync-upstream.sh) — 一键同步上游
- [docs/](./docs/) — 产品规格、架构映射等

---

## 为什么选 new-api（而非从零自研）

我们要做的「国产图像/视频模型转发中转 + 计费 + 后台」，new-api 已经把**用户/密钥/计费/充值/支付/渠道/后台/前端**全做好了，且**自带异步任务框架**（Midjourney/Suno 就是"提交→轮询"模式）和**字节即梦/阿里/火山/Minimax/腾讯/Replicate** 等适配器。我们只需补媒体特有的少量部分。

## 能力映射（我们要的 → new-api 现成）

| 需求 | new-api 组件 |
|------|-------------|
| 用户/登录/令牌 | `model/user.go` `model/token.go` `controller/` |
| 额度/计费/定价 | `model/pricing.go` `model/topup.go` `controller/billing.go` |
| 充值/支付/回调 | `model/topup.go` `controller/payment_webhook_*.go` |
| 渠道(上游)管理 | `model/channel.go` `controller/channel.go` |
| **异步任务（出图/出视频）** | `relay/relay_task.go` `model/task.go` `controller/midjourney.go` |
| **任务型渠道适配器** | `relay/channel/task/`（扩展点）|
| 字节(即梦)/阿里/火山等适配器 | `relay/channel/{jimeng,ali,volcengine,minimax,...}/` |
| Playground/日志/后台 | `controller/playground.go` `web/` |

## 我们要新增的（媒体特有，尽量新文件）

1. **缺失模型的适配器**：在 `relay/channel/task/<新目录>` 按范式加（Seedance/Wan/Veo 等没覆盖的）。
2. **按张/按秒计价**：new-api 是按 token 计费，额度系统现成，**改计量公式**接入媒体单价/倍率。
3. **产出对象存储**：图/视频结果存腾讯云 COS（7 天 / 预签名），new-api 任务结果原本是 URL 透传。
4. **必要的渠道类型注册**：在上游枚举/switch 加我们的类型（核心改动，最小化 + 注释包裹）。

## 部署

沿用 new-api 既有方式（Go 单体 + PG + Redis + docker-compose），与 x2api 同源、同机可共存。
镜像走 GitHub 构建 → 阿里云 ACR → 腾讯云拉取。
