# WaveSpeedAI 产品规格分析报告

> 数据来源：官网页面（首页 / 定价 / FAQ / 退款政策 / 模型详情页，浏览器实测）+ 讨论区评论（Trustpilot、第三方测评、客户证言）+ 竞品第三方资料。
> 编制日期：2026-06-12。价格与模型随时变动，落地前以官方价格页为准。

---

## 目录
1. [产品定位](#一产品定位)
2. [全模型清单](#二全模型清单)
3. [性能与可靠性规格](#三性能与可靠性规格)
4. [定价规格](#四定价规格)
5. [具体模型规格（三个代表）](#五具体模型规格三个代表)
6. [讨论区评论：真实情况对照](#六讨论区评论真实情况对照)
7. [竞品深度横评](#七竞品深度横评)
8. [选型结论](#八选型结论)
9. [信息来源](#九信息来源)

---

## 一、产品定位

WaveSpeed.ai 不是单一创作工具，而是一个 **API 优先的多模型 AI 媒体生成"加速层/聚合平台"**。核心卖点：一个 API Key + 统一接口调用 **1000+ 模型**，覆盖图像、视频、音频、3D、语音、LLM。口号：*"Engineered for Velocity"*。

面向两类用户：
- **开发者**：REST API + Node/Python/cURL SDK + CLI，"ship in minutes, not days"。
- **创作者**：Web 工作台 + 桌面客户端（Win/macOS/Linux）+ Studio，无需写代码。

---

## 二、全模型清单

首页标注 **"Explore All 1000+ models"**，按提供方品牌 + 能力场景两个维度组织。

### 按提供方（33 个品牌专区）

| 提供方 | 代表模型 | 类型 |
|------|------|------|
| **ByteDance** | Seedance 2.0 / 2.0-fast / 1.5 Pro、seed-speech-tts-2.0 | 视频/语音 |
| **Alibaba** | Wan 2.7 / 2.6 / 2.5 / 2.2 / 2.1、Qwen Image 2 / Qwen Image、Qwen3 Max | 视频/图像/LLM |
| **Google** | Veo 3.1 Fast、Nano Banana 2 / Pro、Gemini 3.1 Pro | 视频/图像/LLM |
| **OpenAI** | GPT-Image-2、GPT-Image-1-mini、Whisper、GPT-5.4 | 图像/语音/LLM |
| **Kling（快手）** | Kling O3 (std/pro/4k)、Kling v3.0、v2.6 Pro | 视频 |
| **Minimax** | Hailuo-02 (pro/standard/i2v/t2v) | 视频 |
| **Black Forest Labs** | FLUX Image Tools、FLUX Kontext、Flux 2 Klein | 图像 |
| **Runway** | Gen4 Turbo、Gen4 Aleph、Gen4 Image Turbo、Upscale v1 | 视频 |
| **Luma** | Ray 3.2 (i2v/t2v/reframing/edit) | 视频 |
| **Vidu** | Q3 / Q3 Pro / Q3 Turbo | 视频 |
| **Pixverse** | Pixverse C1 (ref/t2v/i2v/transition) | 视频 |
| **Tencent** | Hunyuan Video (t2v/i2v)、Hunyuan3D v2.1 | 视频/3D |
| **Skywork** | SkyReels v4 / v3 | 视频 |
| **Grok (xAI)** | Grok Imagine Video v1.5、Grok-2 Image | 视频/图像 |
| **Ideogram / Dreamina** | Ideogram v2–v4 | 图像 |
| **Recraft** | Recraft v4.1 / Pro（含矢量图） | 图像 |
| **Stability AI** | Stable Audio 3（music/inpainting/outpainting） | 音频 |
| **Mureka / Sonilo** | Mureka v8/v9、text-to-music、video-to-music | 音乐 |
| **Clarity / Pruna / Bria** | 各类 Upscaler、背景移除、视频增强 | 工具 |
| **HappyHorse** | happyhorse-1.0（video-edit/extend/i2v） | 视频 |
| **第三方 LLM** | Claude Opus/Sonnet 4.6、DeepSeek V4 | LLM |

### 按能力场景（工具聚合，括号为模型数）

| 场景 | 模型数 | 场景 | 模型数 |
|------|---|------|---|
| Best Image Models | 50 | LoRA Generation | 41 |
| Avatar Lipsync | 41 | Speech Generation | 39 |
| Image Editing | 36 | Ultra Selection | 33 |
| Best Video Models | 32 | First/Last Frame Video | 28 |
| Remove Anything | 26 | Video Edit | 21 |
| 3D Creation | 21 | Video Extend | 18 |
| Upscale Image | 17 | Swap Anything | 16 |
| Generate Music | 13 | Content Detection | 11 |
| Object Detection/Seg | 10 | Training Tools | 10 |
| Enhance Videos | 10 | Audio for Video | 8 |
| Motion Control | 8 | | |

### 单价速查（每张图 / 每秒视频，$1 产出）

- **最便宜图像**：Z Image Turbo $0.005（200张）、Flux 2 Klein $0.008（125张）
- **主流图像**：Seedream 4.5 $0.04、Nano Banana 2 $0.07、Nano Banana Pro $0.14
- **最便宜视频**：Wan 2.2 Ultra Fast $0.01/秒（100秒）、InfiniteTalk $0.03/秒
- **主流视频**：Kling 3.0 Std $0.084/秒、Seedance 2.0 Fast / Wan 2.7 $0.10/秒、Veo 3.1 Fast $0.15/秒

> **URL 规律**：`/models/{提供方}/{模型}/{任务}`，如 `bytedance/seedance-2.0/image-to-video`、`alibaba/wan-2.7/image-to-video`、`google/nano-banana-2/text-to-image`。

---

## 三、性能与可靠性规格

| 指标 | 官方宣称 |
|------|------|
| 冷启动延迟 | ~100ms（模型常驻 GPU，"零冷启动"） |
| 图像生成 | 亚秒级 |
| 视频渲染 | 比同类快 **最高 4×** |
| 可用性 SLA | **99.99% uptime**（定价页/标准版另有处写 99.9%，口径略有出入） |
| 安全合规 | **SOC 2 Type II**、端到端加密、可选私有 VPC 部署 |

客户证言侧数据：Novita AI 称视频生成成本降低 **最高 67%**；SocialBook CTO 称从 FAL 迁移"差距是天壤之别"。

> ⚠️ 第三方测评指出官方**无公开 status page**，速度/SLA 数据多来自自家博客与合作方基准，需自行用真实 prompt 验证。

---

## 四、定价规格

按量付费（pay-as-you-go），无月费。

### 图像/视频（部分代表）

| 模型 | 单价 | $1 产出 |
|------|------|------|
| Z Image Turbo | $0.005/图 | 200 张 |
| Flux 2 Klein | $0.008/图 | 125 张 |
| Seedream 4.5 | $0.04/图 | 25 张 |
| Nano Banana 2 | $0.07/图 | 14 张 |
| Nano Banana Pro | $0.14/图 | 7 张 |
| Wan 2.2 Ultra Fast | $0.01/秒 | 100 秒 |
| InfiniteTalk | $0.03/秒 | 33 秒 |
| Kling 3.0 Std | $0.084/秒 | 11.9 秒 |
| Seedance 2.0 Fast / Wan 2.7 | $0.10/秒 | 10 秒 |
| Veo 3.1 Fast | $0.15/秒 | 6.6 秒 |

### LLM（每 1K tokens，输入/输出）

| 模型 | 上下文 | 输入 | 输出 |
|------|---|---|---|
| Claude Opus 4.6 | 200K | $0.015 | $0.075 |
| Claude Sonnet 4.6 | 200K | $0.003 | $0.015 |
| GPT-5.4 | 128K | $0.010 | $0.030 |
| GPT-5.4 Mini | 128K | $0.0004 | $0.0016 |
| Gemini 3.1 Pro | 2M | $0.00125 | $0.005 |
| Qwen3 Max | 128K | $0.0012 | $0.0048 |
| DeepSeek V4 | 128K | $0.0007 | $0.0028 |

### Serverless GPU（按秒计费）

| GPU | VRAM | 每小时 | 每秒 |
|------|---|---|---|
| B200 | 141 GB | $5.98 | $0.00166 |
| H100 | 80 GB | $3.49 | $0.00097 |
| A100 | 80 GB | $1.89 | $0.00053 |
| A100 | 48 GB | $1.39 | $0.00039 |
| RTX 5090 | 24 GB | $0.69 | $0.00019 |
| RTX 4090 | 24 GB | $0.49 | $0.00014 |

### 账户层级与规则

- **层级**（充值解锁更高速率/并发）：Bronze(默认) → Silver($100) → Gold($1,000) → Ultra($10,000)。
- $1 免费额度、无需信用卡。
- 积分**永不过期但不可退现**。
- 服务器/技术故障 **3 小时内自动退款**；参数错误（用户输入）不退。
- 生成内容 **7 天后自动删除**（需自行下载归档）。
- 支付：信用卡 / PayPal / 企业电汇。

---

## 五、具体模型规格（三个代表）

### ① Seedance 2.0 Image-to-Video（ByteDance，视频旗舰）

**定位**：好莱坞级电影感视频，原生音画同步、导演级镜头/灯光控制、超强运动稳定性，基于 Seed 统一多模态架构。

**输入参数**

| 参数 | 必填 | 说明 |
|------|---|------|
| `prompt` | ✅ | 电影化场景描述 |
| `image` | ✅ | 起始帧图 URL |
| `last_image` | ❌ | 结束帧（续接/转场） |
| `duration` | ❌ | 4–15 秒（默认 5），连续可调 |
| `aspect_ratio` | ❌ | 16:9 / 9:16 / 4:3 / 3:4 / 1:1 / 21:9（默认随输入图自适应） |
| `resolution` | ❌ | 480p / 720p(默认) / 1080p |
| `generate_audio` | ❌ | 原生同步音频（默认 true） |
| `enable_web_search` | ❌ | 实时信息检索 |

**关键能力**：单模型处理文/图/音/视；最多 **4 张参考图**保持风格/角色一致；原生音画同步一次生成。

**定价（线性按时长）**

| 分辨率 | 5s | 10s | 15s |
|------|---|---|---|
| 480p | $0.60 | $1.20 | $1.80 |
| 720p | $1.20 | $2.40 | $3.60 |
| 1080p | $3.00 | $6.00 | $9.00 |

计费规则：480p 基准 $0.60/5s；720p = 2×；1080p = 5×（即 720p 的 2.5×）。Playground 默认 $1.2/run。

---

### ② Nano Banana 2 / Text-to-Image（Google = Gemini 3.1 Flash Image，图像旗舰）

**定位**：Pro 级画质 + Flash 速度，512px–4K，文字渲染强、最多 **5 个角色一致性**、整合真实世界知识、无冷启动。

**输入参数**

| 参数 | 必填 | 说明 |
|------|---|------|
| `prompt` | ✅ | 唯一必填项 |
| `aspect_ratio` | ❌ | 14 种：1:1 / 3:2 / 2:3 / 3:4 / 4:3 / 4:5 / 5:4 / 9:16 / 16:9 / 21:9 / 1:4 / 4:1 / 1:8 / 8:1 |
| `resolution` | ❌ | 0.5k / 1k(默认) / 2k / 4k |
| `enable_web_search` | ❌ | +$0.014 |
| `enable_image_search` | ❌ | +$0.014 |
| `output_format` | ❌ | png(默认) / jpeg |
| `enable_sync_mode` | ❌ | 同步返回（仅 API，可能超时） |
| `enable_base64_output` | ❌ | BASE64 输出（仅 API） |

**定价**：0.5k $0.045 / 1k $0.07 / 2k $0.105 / 4k $0.14。2K=1.5×、4K=2× 标准价；web/image search 各 +$0.014。默认 $0.07/run（~14张/$1）。

---

### ③ Wan 2.7 Image-to-Video（Alibaba，视频性价比款）

**定位**：图生视频 720p/1080p，可选音频，支持首尾帧控制、负向提示、可复现 seed，无冷启动。

**输入参数**

| 参数 | 必填 | 说明 |
|------|---|------|
| `image` | ✅ | 起始帧 |
| `prompt` | ✅ | 运动/镜头/氛围描述 |
| `last_image` | ❌ | 结束帧（显著改善运动方向与叙事连贯） |
| `audio` | ❌ | 音轨同步（节奏/情绪/配音） |
| `negative_prompt` | ❌ | 排除元素 |
| `resolution` | ❌ | 720p(默认) / 1080p |
| `duration` | ❌ | 秒（默认 5） |
| `enable_prompt_expansion` | ❌ | 自动优化 prompt |
| `seed` | ❌ | 可复现，-1 为随机 |

**定价**

| 时长 | 720p | 1080p |
|------|---|---|
| 5s | $0.50 | $0.75 |
| 10s | $1.00 | $1.50 |
| 15s | $1.50 | $2.25 |

计费：720p $0.10/秒；1080p $0.15/秒（1.5× 基准）。

### 三个模型的共性结论

1. **统一调用模式**：`POST https://api.wavespeed.ai/api/v3/{provider}/{model}/{task}` → 返回 prediction id → 轮询直到 `completed` → 读 `data.outputs[0]`。每页自带 HTTP/Node/Python 示例。
2. **计费透明可推算**：视频按"基准分辨率 × 倍率 × 秒数"线性计价；图像按分辨率档位。
3. **音频是新一代卖点**：Seedance 2.0 原生音画同步、Wan 2.7 支持音轨输入。
4. **每页含 Schema / Playground / API / History 三视图 + README + FAQ**，文档化程度较高。

---

## 六、讨论区评论：真实情况对照

**正面（社区/客户）**
- 速度与团队响应快被反复点名（SocialBook、Draw Things："FLUX 结果一样，但现在 3 秒内出"）。
- 独家 ByteDance/Alibaba 模型是不可替代的选型理由。
- 视频场景性价比高（成本降幅显著）。

**负面/争议（Trustpilot + 第三方测评）**
1. **计费争议**：有用户反映 Auto-Topup 设 $15 却被扣两次共 $55，无确认页、退款困难。
2. **内容审核突变**：黑五后大量新用户充值后，平台**未提前通知**上线了非常敏感的内容过滤器，逼用户转用 ComfyUI（其节点功能远不如官网）。
3. **输出不稳定**：质量高度依赖选哪个模型，"有的惊艳，有的失望"。
4. **透明度**：无公开 status page / 正式 SLA 文档；积分不可退、按用户错误不退款。
5. **样本量小**：Trustpilot 当时仅 3 条评价，代表性有限。

---

## 七、竞品深度横评

**对比对象：WaveSpeedAI · Fal.ai · Replicate · Runware · Novita AI**（优先第三方资料，对各家自营博客主张降权）

### 对比总表

| 维度 | WaveSpeedAI | Fal.ai | Replicate | Runware | Novita AI |
|---|---|---|---|---|---|
| **模型广度** | 600–1000+ 精选 | 600+ 精选媒体 | **50,000+（最广）** | 40 万+（开源） | 200+ API + GPU |
| **独家/国产** | **字节+阿里首发** | 部分有 | 社区可得 | 基本无 | DeepSeek/Qwen(LLM) |
| **自研引擎** | ✅ | ✅（FLUX 最快） | ❌ | ✅ Sonic | 部分 |
| **冷启动** | 零冷启动(宣称) | 几乎无 | **差(30–60s+)** | 亚秒(宣称) | 自动扩缩 |
| **图像单价** | ~$0.027–0.04/次 | ~$0.025–0.05 | ~$0.03–0.055(最贵) | **$0.0006 起(最低)** | ~$0.0015 |
| **视频** | Seedance 2 ~$0.6/次 | 全模型按秒 | 按 GPU 秒 | 从 $0.14 起 | 有 |
| **LLM** | 弱 | 弱 | 有 | 无 | **强(OpenAI兼容,$0.08起)** |
| **SLA/可靠性** | 99.9%(宣称) | **99.99%+failover+SOC2** | status page透明,客服差 | 资金足,数据少 | **无明确SLA** |
| **DX/SDK** | 统一API | **6 SDK+流式(最佳)** | Cog自部署+生态成熟 | 单一API | OpenAI兼容+HF |
| **ComfyUI/n8n** | API | ✅ 成熟 | ✅ 最成熟 | 可挂载 | ✅ |
| **内容审核** | 可选开关 | **最严最规范** | 自助/自部署 | 披露最少 | 可配置+含未审查 |

### 按需求选型

| 需求 | 首选 |
|------|------|
| 实时/低延迟/移动端 | **Fal.ai**（最快+failover+审核规范，但贵） |
| 大批量极致省钱图像 | **Runware**（$0.0006/图，但只有开源模型） |
| 研究/冷门/自部署 | **Replicate**（5万+模型，但冷启动差、客服差；已被 Cloudflare 收购） |
| **字节/阿里国产图像视频 + 最早拿到** | **WaveSpeedAI**（首发+零冷启动，但第三方数据少需自测） |
| LLM/裸 GPU/微调性价比 | **Novita AI**（OpenAI 兼容，但无 SLA 慎用） |

**实战共识**：这不是单选题。典型生产组合 = WaveSpeed 跑专有/国产 + Runware 跑高并发批量 + Fal 跑实时 + Novita 跑 LLM/GPU。别信营销头条，用真实 workload 在 2–3 家跑基准再定。

### 诚实提醒
1. "亚秒冷启动""快 4×/10×"几乎全是厂商口径，唯一独立基准（HN/venki.dev）只覆盖 Replicate。
2. WaveSpeed 模型数有 "600+" 与 "1000+" 两种说法；大量横评出自其自家博客，对"独家/最快"主张需保持警惕。
3. 价格波动频繁（WaveSpeed 当时对 Seedance 有 10–20% 滚动折扣）。
4. 字节 Seedance 2.0 因好莱坞 IP 风波，2026 初开发者 API 全网延迟铺开（行业性问题）。

---

## 八、选型结论

把三块分析合起来看，WaveSpeed 的产品规格定位很清晰：
- **护城河** = 字节 Seedance/Seedream + 阿里 WAN/Qwen 的**首发时间差** + **零冷启动**，这是 Fal/Replicate/Runware 短期补不齐的。
- **软肋** = 速度/SLA 的强主张几乎全是自家口径，第三方独立基准缺失；可靠性透明度（无 status page）、内容审核稳定性、计费争议是社区实打实的吐槽点。
- **打法建议** = 最适合作为"国产模型接入层"，与一家走量平台（Runware）和一家实时/审核严格平台（Fal）组合使用，而非指望单平台通吃。

**落地前 checklist**：
- ✅ 用真实 prompt 自测目标模型的速度与质量，别信博客数字。
- ✅ 关闭/限制 Auto-Topup，防双扣。
- ✅ 务必 7 天内归档产出。
- ✅ NSFW/边缘内容受审核策略影响大，且策略可能无预警变更。

---

## 九、信息来源

### 官网页面
- [首页](https://wavespeed.ai)
- [定价](https://wavespeed.ai/pricing)
- [FAQ](https://wavespeed.ai/docs/docs-faq)
- [退款政策](https://wavespeed.ai/docs/refund-policy)
- [排障指南](https://wavespeed.ai/docs/troubleshooting-guide)
- 模型详情页（浏览器实测）：`bytedance/seedance-2.0/image-to-video`、`google/nano-banana-2/text-to-image`、`alibaba/wan-2.7/image-to-video`

### 讨论区/评论
- [Trustpilot 用户评价](https://www.trustpilot.com/review/wavespeed.ai)
- [Filmora 测评（隐藏权衡）](https://filmora.wondershare.com/video-editor-review/wavespeed-ai-review.html)
- [Skywork 实测](https://skywork.ai/blog/wavespeed-ai-review-2025/)
- [ai-cmo 测评](https://ai-cmo.net/tools/wavespeed-ai)
- [LinkstartAI 评测](https://www.linkstartai.com/en/agents/wavespeed-ai)

### 竞品横评（第三方为主）
- [apidog 平台横评](https://apidog.com/blog/best-ai-inference-platform-guide-2026/)
- [getdeploying Fal vs Replicate](https://getdeploying.com/fal-ai-vs-replicate)
- [TeamDay AI 价格对比](https://www.teamday.ai/blog/ai-api-pricing-comparison-2026)
- [buildmvpfast 视频价格](https://www.buildmvpfast.com/api-costs/ai-video)
- [DevTk Seedance/Sora/Kling/Veo 价格](https://devtk.ai/en/blog/ai-video-generation-pricing-2026/)
- [IsDown fal 状态](https://isdown.app/status/fal)
- [Trustpilot Replicate](https://www.trustpilot.com/review/replicate.com)
- [HN 冷启动实测](https://news.ycombinator.com/item?id=39411748)
- [TechCrunch Runware $50M A 轮](https://techcrunch.com/2025/12/11/runware-raises-50m-series-a-from-dawn-capital-comcast-ventures-to-become-the-api-for-all-ai/)
- [Cloudflare 收购 Replicate](https://www.cloudflare.com/press/press-releases/2025/cloudflare-to-acquire-replicate-to-build-the-most-seamless-ai-cloud-for-developers/)
- [fal Trust & Safety](https://fal.ai/legal/trust-and-safety)
