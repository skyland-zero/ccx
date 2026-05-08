import { describe, expect, it } from 'vitest'
import type { ChannelRecentActivity, ActivitySegment } from '../services/api'
import { expandSparseSegments } from '../services/api'

function makeActivity(overrides: Partial<ChannelRecentActivity> = {}): ChannelRecentActivity {
  return {
    channelIndex: 0,
    segments: {},
    totalSegs: 150,
    rpm: 0,
    tpm: 0,
    ...overrides,
  }
}

describe('expandSparseSegments', () => {
  it('空稀疏对象返回 150 个零值段', () => {
    const result = expandSparseSegments(makeActivity())
    expect(result).toHaveLength(150)
    expect(result[0].requestCount).toBe(0)
    expect(result[149].requestCount).toBe(0)
  })

  it('稀疏 Map 格式正确展开到指定索引', () => {
    const seg: ActivitySegment = { requestCount: 5, successCount: 4, failureCount: 1, inputTokens: 100, outputTokens: 50 }
    const result = expandSparseSegments(makeActivity({ segments: { 3: seg, 99: { ...seg, requestCount: 10 } } }))
    expect(result[3].requestCount).toBe(5)
    expect(result[3].inputTokens).toBe(100)
    expect(result[99].requestCount).toBe(10)
    expect(result[0].requestCount).toBe(0)
  })

  it('旧版数组格式直接返回（兼容路径）', () => {
    const arr: ActivitySegment[] = [
      { requestCount: 1, successCount: 1, failureCount: 0, inputTokens: 10, outputTokens: 5 },
      { requestCount: 2, successCount: 2, failureCount: 0, inputTokens: 20, outputTokens: 10 },
    ]
    const result = expandSparseSegments(makeActivity({ segments: arr }))
    expect(result).toBe(arr) // 同一引用（旧版兼容）
  })

  it('reuse 数组长度匹配时复用原数组引用', () => {
    const reuse: ActivitySegment[] = Array.from({ length: 150 }, () => ({
      requestCount: 0, successCount: 0, failureCount: 0, inputTokens: 0, outputTokens: 0,
    }))
    const activity = makeActivity({ segments: { 0: { requestCount: 3, successCount: 3, failureCount: 0, inputTokens: 0, outputTokens: 0 } } })
    const result = expandSparseSegments(activity, reuse)
    expect(result).toBe(reuse) // 复用原数组
    expect(result[0].requestCount).toBe(3)
    expect(result[1].requestCount).toBe(0)
  })

  it('reuse 数组长度不匹配时创建新数组（totalSegs 变化）', () => {
    const reuse: ActivitySegment[] = Array.from({ length: 100 }, () => ({
      requestCount: 99, successCount: 0, failureCount: 0, inputTokens: 0, outputTokens: 0,
    }))
    const activity = makeActivity({ totalSegs: 150 })
    const result = expandSparseSegments(activity, reuse)
    expect(result).not.toBe(reuse)
    expect(result).toHaveLength(150)
  })

  it('reuse 后正确重置旧值（索引 3 有数据，下次清零）', () => {
    const reuse: ActivitySegment[] = Array.from({ length: 150 }, () => ({
      requestCount: 0, successCount: 0, failureCount: 0, inputTokens: 0, outputTokens: 0,
    }))
    const seg3: ActivitySegment = { requestCount: 7, successCount: 7, failureCount: 0, inputTokens: 50, outputTokens: 20 }
    const result1 = expandSparseSegments(makeActivity({ segments: { 3: seg3 } }), reuse)
    expect(result1[3].requestCount).toBe(7)

    // 第二次调用：索引 3 不在新的稀疏数据中，应清零
    const result2 = expandSparseSegments(makeActivity({ segments: {} }), reuse)
    expect(result2[3].requestCount).toBe(0)
  })

  it('reuse 路径不替换对象引用（防 API 数据污染）', () => {
    const reuse: ActivitySegment[] = Array.from({ length: 150 }, () => ({
      requestCount: 0, successCount: 0, failureCount: 0, inputTokens: 0, outputTokens: 0,
    }))
    const apiSeg: ActivitySegment = { requestCount: 42, successCount: 42, failureCount: 0, inputTokens: 999, outputTokens: 0 }
    expandSparseSegments(makeActivity({ segments: { 5: apiSeg } }), reuse)

    // apiSeg 应保持不变（不被污染）
    expect(apiSeg.requestCount).toBe(42)
    expect(apiSeg.inputTokens).toBe(999)

    // reuse[5] 是我们自己的对象，不是 apiSeg 的引用
    expect(reuse[5]).not.toBe(apiSeg)
    expect(reuse[5].requestCount).toBe(42)
  })

  it('totalSegs 自定义值正确应用', () => {
    const result = expandSparseSegments(makeActivity({ totalSegs: 50 }))
    expect(result).toHaveLength(50)
  })

  it('索引越界的数据被忽略', () => {
    const seg: ActivitySegment = { requestCount: 1, successCount: 1, failureCount: 0, inputTokens: 0, outputTokens: 0 }
    const result = expandSparseSegments(makeActivity({
      segments: { 0: seg, 999: seg, [-1]: seg },
      totalSegs: 10,
    }))
    expect(result).toHaveLength(10)
    expect(result[0].requestCount).toBe(1)
  })
})
