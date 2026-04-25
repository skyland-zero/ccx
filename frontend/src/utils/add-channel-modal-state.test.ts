import { describe, expect, it } from 'vitest'

import type { Channel } from '@/services/api'
import {
  filterValidSupportedModelPatterns,
  isValidSupportedModelPattern,
  resolveChannelWatcherAction,
  syncBaseUrlsFormState
} from './add-channel-modal-state'

const sampleChannel: Channel = {
  index: 1,
  name: 'existing-channel',
  serviceType: 'openai',
  baseUrl: 'https://example.com/v1',
  apiKeys: ['sk-test'],
}

describe('resolveChannelWatcherAction', () => {
  it('新增模式打开时返回重置表单动作', () => {
    expect(resolveChannelWatcherAction({
      show: true,
      newChannel: null,
      oldChannel: null,
    })).toBe('reset-new-form')
  })

  it('编辑模式切入时返回回填动作', () => {
    expect(resolveChannelWatcherAction({
      show: true,
      newChannel: sampleChannel,
      oldChannel: null,
    })).toBe('load-edit-channel')
  })

  it('同一渠道静默保存后仅更新基线，不重置本地草稿', () => {
    expect(resolveChannelWatcherAction({
      show: true,
      newChannel: {
        ...sampleChannel,
        name: 'existing-channel-updated',
        baseUrl: 'https://example.com/v2'
      },
      oldChannel: sampleChannel,
    })).toBe('noop')
  })

  it('编辑态 channel 被清空时保持 noop，避免误切快速添加', () => {
    expect(resolveChannelWatcherAction({
      show: true,
      newChannel: null,
      oldChannel: sampleChannel,
    })).toBe('noop')
  })

  it('弹窗关闭时始终忽略 channel 变化', () => {
    expect(resolveChannelWatcherAction({
      show: false,
      newChannel: sampleChannel,
      oldChannel: null,
    })).toBe('noop')
  })
})

describe('syncBaseUrlsFormState', () => {
  it('应在当前 serviceType 语义下去重，但不要求回写原始文本', () => {
    expect(syncBaseUrlsFormState('https://host\nhttps://host/v1', 'openai')).toEqual({
      baseUrl: 'https://host',
      baseUrls: []
    })
  })

  it('应保留原始文本，便于后续按最终 serviceType 重算', () => {
    expect(syncBaseUrlsFormState('https://host/v1\nhttps://host', 'openai')).toEqual({
      baseUrl: 'https://host',
      baseUrls: []
    })

    expect(syncBaseUrlsFormState('https://host/v1\nhttps://host', 'gemini')).toEqual({
      baseUrl: 'https://host/v1',
      baseUrls: ['https://host/v1', 'https://host']
    })
  })
})

describe('isValidSupportedModelPattern', () => {
  it('支持精确、前缀、后缀、包含和排除规则', () => {
    expect(isValidSupportedModelPattern('gpt-4o')).toBe(true)
    expect(isValidSupportedModelPattern('gpt-4*')).toBe(true)
    expect(isValidSupportedModelPattern('*image')).toBe(true)
    expect(isValidSupportedModelPattern('*image*')).toBe(true)
    expect(isValidSupportedModelPattern('!*image*')).toBe(true)
  })

  it('拒绝非法中间通配和空规则', () => {
    expect(isValidSupportedModelPattern('foo*bar')).toBe(false)
    expect(isValidSupportedModelPattern('**')).toBe(false)
    expect(isValidSupportedModelPattern('')).toBe(false)
    expect(isValidSupportedModelPattern('   ')).toBe(false)
    expect(isValidSupportedModelPattern('!')).toBe(false)
    expect(isValidSupportedModelPattern('!!gpt-4*')).toBe(false)
  })
})

describe('filterValidSupportedModelPatterns', () => {
  it('过滤非法规则并保留合法规则顺序', () => {
    expect(filterValidSupportedModelPatterns([' gpt-4* ', 'foo*bar', '!*image*'])).toEqual({
      validPatterns: ['gpt-4*', '!*image*'],
      hasInvalidPatterns: true
    })
  })

  it('全部合法时不标记错误', () => {
    expect(filterValidSupportedModelPatterns(['gpt-4*', '*image*'])).toEqual({
      validPatterns: ['gpt-4*', '*image*'],
      hasInvalidPatterns: false
    })
  })
})
