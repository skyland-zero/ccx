# 独立 Images 渠道实施计划

更新时间：2026-04-25

## 目标

为 OpenAI 兼容的 Images API 新增独立渠道类型，避免图片请求与 Chat 渠道混用指标、熔断、密钥和模型配置。

新增对外入口：

- `POST /v1/images/generations`
- `POST /:routePrefix/v1/images/generations`

后续扩展入口：

- `POST /v1/images/edits`
- `POST /:routePrefix/v1/images/edits`
- `POST /v1/images/variations`
- `POST /:routePrefix/v1/images/variations`

官方参考：

- OpenAI Image generation guide: https://platform.openai.com/docs/guides/image-generation/
- OpenAI Images API reference: https://platform.openai.com/docs/api-reference/images

## 一期范围

一期只实现 `generations`，保持实现简单并先打通 SDK 兼容性。

支持能力：

- 独立 `imagesUpstream` 配置。
- 独立 `ChannelKindImages`。
- 独立 Images metrics、logs、熔断、恢复。
- 复用现有 `UpstreamConfig` 字段：`baseUrl`、`baseUrls`、`apiKeys`、`serviceType`、`modelMapping`、`supportedModels`、`routePrefix`、`customHeaders`、`proxyUrl`。
- 透传 OpenAI 兼容 JSON 请求到上游 `/v1/images/generations`。
- 支持 `#` 结尾 baseURL 约定：`https://host#` 转发到 `/images/generations`，普通 baseURL 转发到 `/v1/images/generations`。
- 保留原始响应结构，成功响应不做转换。
- 失败处理、key failover、channel failover 与 Chat/Responses 保持一致。

一期不做：

- `edits` 的 multipart/form-data。
- `variations`。
- Images 转 Responses 的桥接。
- Images 转 Chat 画图的桥接。
- 图片内容落盘、缓存、URL 代理。
- 前端复杂能力矩阵，只做 Images 渠道基础 CRUD 和状态展示。

## 配置设计

新增 `Config.ImagesUpstream`：

```json
{
  "imagesUpstream": [
    {
      "name": "openai-images",
      "serviceType": "openai",
      "baseUrl": "https://api.openai.com",
      "apiKeys": ["sk-..."],
      "modelMapping": {
        "image-default": "gpt-image-1.5"
      },
      "supportedModels": ["gpt-image-*", "dall-e-*"],
      "status": "active"
    }
  ]
}
```

`serviceType` 一期只需要支持 `openai` 和空值默认 `openai`。预留但不实现 `responses`、`chat`、`gemini` 桥接，避免过早扩大分支复杂度。

## 后端改动

### 1. Config

文件：

- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_loader.go`
- 新增 `backend-go/internal/config/config_images.go`

任务：

- 在 `Config` 中增加 `ImagesUpstream []UpstreamConfig`。
- `GetConfig` 深拷贝 `ImagesUpstream`。
- `applyServiceTypeDefaults` 为 Images 设置默认 `openai`。
- 新增 Images 渠道 CRUD 方法，结构参考 `config_chat.go`。
- 新增 `GetCurrentImagesUpstreamWithIndex`。
- 新增 `GetNextImagesAPIKey`。
- 删除渠道时清理 `images` 类型 failed key cache。

### 2. Scheduler

文件：

- `backend-go/internal/scheduler/channel_scheduler.go`
- `backend-go/internal/handlers/common/*`
- `backend-go/internal/handlers/*metrics*`

任务：

- 增加 `ChannelKindImages`。
- `ChannelScheduler` 增加 `imagesMetricsManager` 和 `imagesChannelLogStore`。
- `NewChannelScheduler` 接收 Images metrics。
- `getMetricsManager` 支持 Images。
- `NormalizedMetricsServiceType` 中 Images 默认 `openai`。
- `setChannelStatusByKind`、`scheduledRecoveryKinds`、`restoreScheduledKeysForKind` 支持 Images。
- Dashboard、logs、metrics 相关通用 handler 支持 `type=images`。

### 3. Images Handler

新增目录：

- `backend-go/internal/handlers/images/`

建议文件：

- `handler.go`
- `channels.go`
- `matrix_test.go`
- `handler_test.go`
- `channels_advanced_test.go`
- `delete_upstream_logs_test.go`

`handler.go` 职责：

- 做代理鉴权。
- 读取请求体，限制 `MaxRequestBodySize`。
- 校验 JSON 和 `model`/`prompt` 必填。
- 提取 `stream`，一期允许透传，但不实现特殊 SSE 转换。
- 提取统一 session id，用于 trace affinity。
- 判断多渠道模式。
- 使用 `TryUpstreamWithAllKeys`。
- 构造上游请求 URL：
  - 普通：`{baseURL}/v1/images/generations`
  - `#` 模式：`{baseURL}/images/generations`
- 应用模型映射。
- 透传 Images 请求参数。
- 设置 `Content-Type: application/json` 和 Bearer 认证。
- 应用自定义请求头。
- 成功时透传 body、status、content-type。

### 4. Main 路由

文件：

- `backend-go/main.go`

任务：

- 初始化 Images metrics manager。
- `NewChannelScheduler` 传入 Images metrics。
- 增加管理 API：
  - `GET /api/images/channels`
  - `POST /api/images/channels`
  - `PUT /api/images/channels/:id`
  - `DELETE /api/images/channels/:id`
  - `POST /api/images/channels/:id/keys`
  - `DELETE /api/images/channels/:id/keys/:apiKey`
  - `POST /api/images/channels/reorder`
  - `PATCH /api/images/channels/:id/status`
  - `POST /api/images/channels/:id/resume`
  - `POST /api/images/channels/:id/promotion`
  - `GET /api/images/channels/metrics`
  - `GET /api/images/channels/:id/logs`
- 增加代理 API：
  - `POST /v1/images/generations`
  - `POST /:routePrefix/v1/images/generations`

### 5. Frontend

目录：

- `frontend/`

任务：

- 在渠道类型导航中加入 `Images`。
- 复用现有 Chat 渠道管理组件能力，接入 `/api/images/channels`。
- Dashboard 支持 `type=images`。
- Metrics、logs、models stats 支持 Images。
- 新增 Images 渠道时默认 `serviceType=openai`。

## 测试计划

后端单测：

- 配置加载时 `imagesUpstream` 默认 `serviceType=openai`。
- `GetConfig` 深拷贝 Images 渠道。
- `AddImagesUpstream` 去重 API key 和 baseURL。
- `UpdateImagesUpstream` 保留历史 key、状态迁移、模型映射。
- `RemoveImagesUpstream` 清理 failed key cache。
- `/v1/images/generations` 缺少 `model` 返回 400。
- `/v1/images/generations` 缺少 `prompt` 返回 400。
- 普通 baseURL 转发到 `/v1/images/generations`。
- `#` baseURL 转发到 `/images/generations`。
- 模型映射生效。
- 多 key failover 生效。
- 多 channel failover 生效。
- routePrefix 只选择匹配渠道。
- 成功响应保持 Images API 原样透传。
- 上游 401/403/429/5xx 失败分类与现有通用逻辑一致。

前端验证：

- Images 渠道可新增、编辑、删除。
- Images key 可新增、删除、置顶、置底。
- Images 渠道状态、promotion、resume 可用。
- Images dashboard、metrics、logs 正常展示。

手工验收：

```bash
curl -X POST "http://localhost:3000/v1/images/generations" \
  -H "Authorization: Bearer $PROXY_ACCESS_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-1.5",
    "prompt": "A small red cube on a white table",
    "n": 1,
    "size": "1024x1024"
  }'
```

期望：

- 返回 OpenAI Images API 兼容 JSON。
- Images metrics 计数增加。
- Chat metrics 不变化。
- Images 渠道日志记录成功请求。

## 后续阶段

### 二期：Images Edits

支持：

- `multipart/form-data` 透传。
- JSON 格式 `images` 引用透传。
- `mask` 透传。
- `POST /v1/images/edits` 和 routePrefix 版本。

注意点：

- 不能使用当前只读 JSON body 的 handler 逻辑。
- 需要保留 multipart boundary。
- 请求体大小限制应考虑图片文件，默认限制可能需要单独配置。

### 三期：Images Variations

支持：

- `POST /v1/images/variations`。
- 仅 OpenAI 兼容透传。
- 默认不做模型转换，避免把非 `dall-e-2` 请求错误路由到不支持的上游。

### 四期：桥接能力

可选能力：

- Images API 转 Responses `image_generation` tool。
- Images API 转 Chat 画图。

建议仅作为明确配置项启用：

```json
{
  "serviceType": "responses",
  "imageBridge": "responses"
}
```

默认不自动桥接，避免请求语义和响应格式不透明。

## 风险与约束

- `edits` 是 multipart 和 JSON 双形态，不能套用 generations 的 JSON 处理路径。
- GPT Image 模型默认返回 base64，部分 DALL·E 模型可返回 URL；代理应保持上游响应原样。
- 图片请求耗时更长，后续可能需要独立超时配置。
- 图片文件体积大，后续需要独立 body size 配置，避免影响文本接口。
- 图片使用量不适合用文本 token 估算，metrics 一期只记录请求成功率、延迟、状态码；usage 字段有则记录，没有则不估算。
- 能力测试应避免默认生成真实图片，防止成本不可控；需要手动触发或使用低成本 prompt。

## 推荐实施顺序

1. 后端 Config + Scheduler 增加 Images 基础类型。
2. 实现 `/v1/images/generations` handler 和后端单测。
3. 接入 main 路由和管理 API。
4. 前端增加 Images 渠道页。
5. 跑 `go test ./internal/...` 和前端 `bun run type-check`。
6. 手工调用 generations 验证真实上游。
7. 观察 metrics/logs 后再进入 edits 阶段。

