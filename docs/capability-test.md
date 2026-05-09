# 虚拟协议测试单元测试说明

## 相关文件

- **测试文件**: `backend-go/internal/handlers/capability_test_redirect_test.go`
- **实现文件**: `backend-go/internal/handlers/capability_test_redirect.go`
- **快照存储**: `backend-go/internal/handlers/capability_snapshot_store.go`

## 测试覆盖

### 1. TestRunRedirectVerification_SharedActualModel

**测试场景**：多个探测模型重定向到同一个 actualModel

**测试内容**：
- 验证去重逻辑：只测试不同的 actualModel
- 验证初始状态：所有探测模型都有 `queued` 状态
- 验证最终状态：所有探测模型都有测试结果
- 验证共享 actualModel 的模型有相同的测试结果
- 验证 successCount 是所有探测模型的成功数量（不是 actualModel 数量）

**示例**：
```
claude-sonnet-4-6 → glm-5.1-pro
claude-sonnet-4-5-20250929 → glm-5.1-pro  (共享同一个 actualModel)
claude-haiku-4-5-20251001 → glm-5.1
```

只测试 2 个不同的 actualModel，但 3 个探测模型都显示测试结果。

### 2. TestVirtualProtocolSnapshot_InitialState

**测试场景**：虚拟协议快照的初始状态

**测试内容**：
- 验证虚拟协议名称格式：`sourceTab->channelServiceType`
- 验证初始状态为 `idle`
- 验证模型列表只包含被重定向的模型
- 验证每个模型都有 `actualModel` 字段

### 3. TestCountRedirectSuccesses

**测试场景**：成功计数函数

**测试内容**：
- 空结果列表
- 全部成功
- 全部失败
- 部分成功

### 4. TestUpdateCapabilityJobModelResult_VirtualProtocol

**测试场景**：虚拟协议模型结果更新

**测试内容**：
- 验证状态转换：`queued` → `running` → `success`
- 验证生命周期转换：`pending` → `active` → `done`
- 验证结果字段：`latency`、`streamingSupported` 等

## 运行测试

```bash
# 运行虚拟协议相关测试
cd backend-go
go test -v github.com/BenedictKing/ccx/internal/handlers -run "TestRunRedirectVerification_SharedActualModel|TestVirtualProtocolSnapshot_InitialState|TestCountRedirectSuccesses|TestUpdateCapabilityJobModelResult_VirtualProtocol"

# 运行所有能力测试相关测试
go test -v github.com/BenedictKing/ccx/internal/handlers -run "Capability"

# 运行完整测试套件
go test ./...
```

## 测试覆盖的关键逻辑

1. **模型去重**：只测试不同的 actualModel，避免重复请求
2. **结果共享**：共享 actualModel 的探测模型使用相同的测试结果
3. **状态同步**：所有探测模型的状态实时更新
4. **计数准确**：successCount 反映所有探测模型的成功数量

## 已修复的问题

1. **问题**：共享 actualModel 的模型没有正确更新测试结果
   - **修复**：在 `runRedirectVerification` 结束时，为所有探测模型更新结果

2. **问题**：successCount 计数不准确
   - **修复**：改为统计所有探测模型的成功数量，而不是 actualModel 数量

3. **问题**：虚拟协议状态没有正确更新
   - **修复**：在所有模型测试完成后，更新虚拟协议的 Lifecycle 和 Outcome
