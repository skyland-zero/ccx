# CHANGELOG

## [Unreleased]

### Fixed

- 修复 capability-test 在取消后恢复旧任务时返回过期的 `cancelled` job 快照，避免前端误判任务已结束而停止轮询。
- 为 capability-test 增加取消后恢复场景的 HTTP 回归测试，覆盖恢复响应状态正确性。
