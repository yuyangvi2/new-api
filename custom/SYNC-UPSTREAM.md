# 跟随上游 new-api 更新（操作手册）

本仓库是 `QuantumNous/new-api` 的 fork，目标：**长期跟随上游更新**，同时叠加我们的媒体生成定制。

## 分支模型

| 分支 | 角色 | 规则 |
|------|------|------|
| `main` | 上游干净镜像 | **绝不提交自己的代码**，只用来同步上游 |
| `custom` | 产品分支 | 所有定制都在这；部署也从这出 |

远程：`origin` = 你的 fork；`upstream` = QuantumNous/new-api。

## 日常跟随上游

一键：
```bash
bash custom/sync-upstream.sh
```
它做的事：
1. `git fetch upstream --tags`
2. `main` 快进到 `upstream/main` 并推送 origin
3. 切回 `custom`，把 `main` 合并进来（有冲突会停下让你手动解决）

手动等价：
```bash
git fetch upstream --tags
git checkout main && git merge --ff-only upstream/main && git push origin main
git checkout custom && git merge main      # 解冲突 → 测试 → git push
```

## 减少冲突的纪律（重要）

> 改动越"只增不改"、越隔离，合并上游越无痛。

1. **只增不改**：新功能尽量开**新文件**；我们自己的东西放 `custom/` 命名空间。
2. **新模型适配器**放 `relay/channel/task/<新目录>`（任务型）或 `relay/channel/<新目录>`，照上游适配器范式写——这是 new-api 的官方扩展点，几乎不碰核心。
3. **必须碰核心的只有"注册渠道类型"**那一两处枚举/switch：改动保持极小，并用注释包起来，例如：
   ```go
   // >>> custom: wavespeed media channels
   ...
   // <<< custom
   ```
   这样合并冲突一眼可辨、易解。
4. **DB**：加新表用 GORM AutoMigrate（新文件 `model/xxx.go`），别改上游核心表结构。
5. **配置**全走环境变量（new-api 本就 env 驱动），不硬编码。
6. **前端** `web/`：加页面/路由，不重写核心。
7. 定期（如每周）同步上游，**小步合并**比攒一大坨好解。

## 冲突解决提示

- 冲突基本只会出现在你"碰过的核心文件"上（见纪律 3）。
- 解完跑一遍构建/测试再 push：`make`（见上游 makefile）。
- 真冲突难解时，可单独 `git checkout upstream/main -- <文件>` 取上游版，再把自己的小改动重打上去。
