# CHANGELOG

## [Unreleased]

### Fixed

- 修复公共 `/v1/models` 与 `/v1/models/:model` 未纳入 Chat 渠道的问题，统一按 `messages → responses → chat` 聚合与回退模型查询，并保留 routePrefix 与已拉黑 key fallback 语义。
- 补充 `/v1/models` Chat 聚合与模型详情回退回归测试，覆盖去重优先级、routePrefix 与已拉黑 key fallback 行为。

- 修复 capability-test 在取消后恢复旧任务时返回过期的 `cancelled` job 快照，避免前端误判任务已结束而停止轮询。
- 为 capability-test 增加取消后恢复场景的 HTTP 回归测试，覆盖恢复响应状态正确性。
- 将 capability-test 的限速、共享结果与运行复用收敛到 upstream identity 维度，并新增 shared snapshot API 与单协议测试交互提示。