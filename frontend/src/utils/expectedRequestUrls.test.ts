import { describe, it, expect } from 'vitest'
import { buildExpectedRequestUrls } from './expectedRequestUrls'

describe('buildExpectedRequestUrls', () => {
  it('应为 responses 渠道上的 gemini 上游生成正确预览 URL', () => {
    const result = buildExpectedRequestUrls('responses', 'gemini', 'https://generativelanguage.googleapis.com')

    expect(result).toHaveLength(1)
    expect(result[0].expectedUrl).toBe(
      'https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent'
    )
  })

  it('应在 baseUrl 已含版本前缀时避免重复追加版本', () => {
    const result = buildExpectedRequestUrls('responses', 'gemini', 'https://generativelanguage.googleapis.com/v1beta')

    expect(result).toHaveLength(1)
    expect(result[0].expectedUrl).toBe(
      'https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent'
    )
  })

  it('应为 responses 渠道上的 claude 上游生成 messages 端点', () => {
    const result = buildExpectedRequestUrls('responses', 'claude', 'https://api.anthropic.com')

    expect(result).toHaveLength(1)
    expect(result[0].expectedUrl).toBe('https://api.anthropic.com/v1/messages')
  })

  it('应为 responses 渠道上的 openai 上游生成 chat completions 端点', () => {
    const result = buildExpectedRequestUrls('responses', 'openai', 'https://api.openai.com')

    expect(result).toHaveLength(1)
    expect(result[0].expectedUrl).toBe('https://api.openai.com/v1/chat/completions')
  })

  it('应为 messages 渠道上的 responses 上游生成 responses 端点', () => {
    const result = buildExpectedRequestUrls('messages', 'responses', 'https://api.openai.com')

    expect(result).toHaveLength(1)
    expect(result[0].expectedUrl).toBe('https://api.openai.com/v1/responses')
  })

  it('应为 chat 渠道上的 responses 上游生成 chat completions 端点', () => {
    const result = buildExpectedRequestUrls('chat', 'responses', 'https://api.openai.com')

    expect(result).toHaveLength(1)
    expect(result[0].expectedUrl).toBe('https://api.openai.com/v1/chat/completions')
  })

  it('应让根域名与默认版本前缀预览到同一请求地址', () => {
    const root = buildExpectedRequestUrls('chat', 'openai', 'https://new.timefiles.online')
    const versioned = buildExpectedRequestUrls('chat', 'openai', 'https://new.timefiles.online/v1')

    expect(root[0].expectedUrl).toBe('https://new.timefiles.online/v1/chat/completions')
    expect(versioned[0].expectedUrl).toBe(root[0].expectedUrl)
  })
})
