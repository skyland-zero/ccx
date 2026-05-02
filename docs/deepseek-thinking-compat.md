# DeepSeek Thinking 兼容性备忘

## 背景

DeepSeek Chat Completions 的 thinking 模式会返回 `reasoning_content`。在普通多轮对话中，这段内容可以不参与后续上下文；但如果中间发生了工具调用，DeepSeek 要求后续请求继续回传相关 assistant 消息里的 `reasoning_content`，否则可能返回 400。

部分下游 CLI 或 Agent 客户端不支持 `reasoning_content` 字段，例如更偏 Anthropic Messages 语义的工具调用客户端。此时客户端无法可靠保存并回传 DeepSeek 要求的 thinking 状态。

## 当前判断

`/v1/chat/completions` 的 OpenAI 兼容上游路径以透传为主，客户端显式传入的 `thinking`、`reasoning_effort`、`reasoning_content` 通常不会被主动删除。

真正的兼容风险在于：

- 下游 CLI 不认识 `reasoning_content`，可能在下一轮请求中丢失该字段。
- DeepSeek 在 thinking + tool calls 场景下要求回传该字段。
- 如果 CLI 还会裁剪历史中的 assistant tool call 消息，代理也无法稳定恢复上下文。

## 临时策略

短期先不实现运行时代码变更。对于 CLI/Agent 类渠道，如果出现 DeepSeek thinking 与工具调用不兼容，优先禁用 thinking：

```json
{
  "thinking": {
    "type": "disabled"
  }
}
```

该策略简单稳定，能避免工具调用续轮因缺少 `reasoning_content` 导致 400。代价是这类渠道不能使用 DeepSeek thinking 输出。

## 后续可选方案

如果后续确实需要同时支持 DeepSeek thinking 和工具调用，可以在代理层实现 reasoning bridge：

1. 上游 DeepSeek 返回 `reasoning_content` 且包含 `tool_calls` 时，代理按会话保存 thinking 状态。
2. 返回给下游 CLI 时剥离 `reasoning_content`，避免客户端不支持。
3. 下一轮 CLI 回传 tool result 时，代理在发往 DeepSeek 前，把对应 `reasoning_content` 注入回原 assistant tool call 消息。

实现要求：

- 必须按会话隔离，例如使用 `Conversation_id`、`Session_id`、`X-Claude-Code-Session-Id`、`metadata.user_id` 等统一会话标识。
- 缓存必须设置 TTL，避免长期占用内存和跨会话串话。
- 流式响应需要累积 `delta.reasoning_content`，在工具调用完成时落缓存。
- 不能把 `reasoning_content` 写进普通 `content`，否则会污染用户可见上下文。
- 如果下游已经裁剪掉 assistant tool call 消息，则无法可靠桥接，应回退到禁用 thinking。

## 原则

默认保持 KISS / YAGNI：先记录兼容风险，不引入状态桥接。只有在明确有用户场景需要 DeepSeek thinking + tool calls 且下游 CLI 不支持该字段时，再实现代理层 bridge。
