# backend-go 模块文档

[← 根目录](../CLAUDE.md)

## 模块职责

Go 后端核心服务：HTTP API、多上游适配、协议转换、多渠道调度、会话管理、指标与日志记录、配置热重载。

## 启动命令

```bash
make dev
make run
make build
make test
make test-cover
make fmt
make lint
make deps
```

说明：实际命令以 `backend-go/Makefile` 为准。

## API 总览

### 代理入口
- `GET /health`
- `POST /v1/messages`
- `POST /v1/messages/count_tokens`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/responses/compact`
- `GET /v1/models`
- `GET /v1/models/:model`
- `POST /v1beta/models/{model}:generateContent`
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `POST /v1/images/variations`

### 管理入口
- `/api/messages/channels/*`
- `/api/chat/channels/*`
- `/api/responses/channels/*`
- `/api/gemini/channels/*`
- `/api/images/channels/*`

### 能力测试
能力测试接口仅适用于 `messages`、`chat`、`responses`、`gemini`。
Images 当前没有 capability-test / snapshot 路由。

## 关键实现点

### 渠道类型
当前正式支持五类渠道：
- `messages`
- `chat`
- `responses`
- `gemini`
- `images`

### Provider 接口
所有上游服务实现 `internal/providers/Provider` 接口：

```go
type Provider interface {
    ConvertToProviderRequest(c *gin.Context, upstream *config.UpstreamConfig, apiKey string) (*http.Request, []byte, error)
    ConvertToClaudeResponse(providerResp *types.ProviderResponse) (*types.ClaudeResponse, error)
    HandleStreamResponse(body io.ReadCloser) (<-chan string, <-chan error, error)
}
```

### 核心模块

| 模块 | 职责 |
| --- | --- |
| `internal/handlers/` | 代理与管理接口处理器 |
| `internal/providers/` | 上游适配 |
| `internal/converters/` | Responses 协议转换 |
| `internal/scheduler/` | 多渠道调度 |
| `internal/session/` | 会话与 Trace 亲和性 |
| `internal/config/` | 配置管理与热重载 |
| `internal/metrics/` | 指标、日志、持久化 |

## 日志规范

所有日志输出使用 `[Component-Action]` 标签格式，禁止使用 emoji。

```go
log.Printf("[Component-Action] 消息内容: %v", value)
```

### Channels logs

渠道日志模型定义于 `internal/metrics/channel_log.go`。

关键字段包括：
- `status`
- `statusCode`
- `requestSource`
- `interfaceType`
- `baseUrl`
- `keyMask`

对于 Images 渠道，还会额外记录：
- `operation`：`generations` / `edits` / `variations`

## 扩展指南

### 添加新上游能力
1. 在 `internal/providers/` 中新增或扩展 provider
2. 在需要时补充 `internal/converters/` 的协议转换
3. 接入对应 handler、调度与前端配置
4. 将指标、日志与模型过滤纳入统一链路

### 调整调度逻辑
- 修改 `internal/scheduler/`
- 如涉及熔断、恢复或日志，联动检查 `internal/metrics/`

## 文档导航

- [../README.md](../README.md) - 项目入口
- [README.md](README.md) - 后端专项文档
- [../ARCHITECTURE.md](../ARCHITECTURE.md) - 架构说明
- [../DEVELOPMENT.md](../DEVELOPMENT.md) - 开发指南
- [../RELEASE.md](../RELEASE.md) - 发布流程
