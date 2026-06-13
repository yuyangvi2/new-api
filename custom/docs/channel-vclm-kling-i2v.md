# 渠道规格：腾讯云 VCLM 可灵图生视频（自建适配器用）

来源：腾讯文档「图生视频K-测试指引」。这是 new-api 要新增的第一个自定义任务适配器（N4）。

## 服务 / 鉴权
- 服务：**VCLM**（视频创作大模型），endpoint `vclm.tencentcloudapi.com`，**Version `2024-05-23`**
- 鉴权：腾讯云 **TC3-HMAC-SHA256**（SecretId/SecretKey），需主账户加白 + 子账户授权策略 `QcloudVCLMFullAccess`
- 异步任务：提交→轮询查询。建议查询节奏：首次 5 分钟，之后递减。
- 状态：`WAIT`/`RUN`/`FAIL`/`DONE`；DONE 取 `ResultVideoUrl`，FAIL 取 `ErrorCode`/`ErrorMessage`

## Action 1：SubmitImageToVideoJob（提交）
输入（节选关键）：
- `Model`：v1.0=Kling-V1 / v1.5=Kling-V1-5 / v1.6=Kling-V1-6 / v2.0=Kling-V2-Master / v2.1=Kling-V2-1 / v2.1m=Kling-V2-1-master / v2.5=Kling-V2-5-Turbo / v2.6=Kling-V2-6 / v3.0=kling-v3（示例 v1.6）
- `Image`：参考图（URL，<10MB，≥300×300，宽高比 1:2.5~2.5:1，jpg/png）。Image 与 ImageTail 至少一个
- `ImageTail`：尾帧图（URL 或 Base64）
- `Prompt`/`NegativePrompt`：≤2500 字
- `Duration`：秒，默认 5。v1.6/v2.0/v2.5/v2.6 支持 5/10；v3.0 支持 3~15
- `Mode`：std/pro/4k（4k 仅 v3.0）
- `CfgScale`：[0,1]，v2.* 不支持
- `Sound`：on/off（是否生成声音）
- `MultiShot`/`ShotType`/`MultiPrompt[]`：多镜头
- `ElementList[]`：参考主体（最多3）
- `StaticMask`/`DynamicMasks[]`/`CameraControl`：运动笔刷/镜头控制（三选一与 ImageTail 互斥）
- `LogoAdd`/`LogoParam`、`VoiceList[]`、`CallbackUrl`、`ExternalTaskId`

输出：`JobId`、`ExternalTaskId`、`RequestId`

## Action 2：DescribeImageToVideoJob（查询）
输入：`JobId`（或 `ExternalTaskId`）
输出：`Status`、`ErrorCode`、`ErrorMessage`、`ResultVideoUrl`、`VideoId`、`Duration`、`FinalUnitDeduction`、`ExternalTaskId`、`RequestId`

## 对 new-api 的实现要点（N4）
- new-api 自带 `kling` 任务适配器是**快手官方 JWT 鉴权**，与此**不兼容**（这是腾讯云 TC3）。需新增适配器 `relay/channel/task/vclm/`（或 tencentvclm）。
- 复用 new-api `relay/channel/tencent/` 里已有的 **TC3 签名**逻辑。
- 实现 `TaskAdaptor` 接口：Submit 映射到 SubmitImageToVideoJob，FetchTask/ParseTaskResult 映射到 DescribeImageToVideoJob，Status DONE→success(取 ResultVideoUrl)、FAIL→failure。
- 计费：可用 `FinalUnitDeduction` 做 AdjustBillingOnComplete 的实际结算依据，或按 Duration×档位倍率（EstimateBilling）。
- 注册一个新渠道类型（核心最小改动）。
- ⚠️ 改了代码后，运行的容器需换成**自建镜像**（当前北京机跑的是官方 calciumion/new-api 镜像，不含此适配器）→ 需 N7 构建镜像 + 重新部署。

## 可先验证（省得白写）
若当前 tccli 账号(6108661)已加白 VCLM + 有 QcloudVCLMFullAccess，可直接：
`tccli vclm SubmitImageToVideoJob --Model v1.6 --Image <url> --Prompt ...` → 拿 JobId →
`tccli vclm DescribeImageToVideoJob --JobId <id>` → 看真实响应，照着写适配器最稳。
