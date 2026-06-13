#!/usr/bin/env bash
# 跟随上游 QuantumNous/new-api 更新。详见 custom/SYNC-UPSTREAM.md。
set -euo pipefail
export GIT_SSH_COMMAND='ssh -o ConnectTimeout=15'

echo "==> fetch upstream"
git fetch upstream --tags

echo "==> 同步 main 到 upstream/main"
git checkout main
git merge --ff-only upstream/main
git push origin main

echo "==> 切回 custom，合并 main"
git checkout custom
if git merge main; then
  echo "✅ 合并成功。请构建/测试后执行: git push"
else
  echo "⚠️  有冲突，请手动解决后: git add -A && git commit && git push"
  echo "    冲突通常只在你改过的核心文件上（见 SYNC-UPSTREAM.md 纪律）。"
  exit 1
fi
