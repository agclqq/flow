#!/usr/bin/env bash
# 通过 API 创建「并行分支 + 聚合节点」示例流水线。
# 推荐使用 Python 版本以避免 shell JSON 转义问题。
exec python3 "$(dirname "$0")/create_parallel_join_pipeline.py" "$@"
