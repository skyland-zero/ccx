import type { Channel } from '@/services/api'
import { deduplicateEquivalentBaseUrls, type ServiceType } from './baseUrlSemantics'

export type ChannelWatcherAction = 'load-edit-channel' | 'reset-new-form' | 'noop'

export function resolveChannelWatcherAction(params: {
  show: boolean
  newChannel: Channel | null | undefined
  oldChannel: Channel | null | undefined
}): ChannelWatcherAction {
  const { show, newChannel, oldChannel } = params

  if (!show) {
    return 'noop'
  }

  if (newChannel) {
    return 'load-edit-channel'
  }

  if (oldChannel) {
    return 'noop'
  }

  return 'reset-new-form'
}

export function syncBaseUrlsFormState(rawText: string, serviceType: ServiceType): {
  baseUrl: string
  baseUrls: string[]
} {
  const rawUrls = rawText
    .split('\n')
    .map(s => s.trim())
    .filter(Boolean)
  const deduplicated = deduplicateEquivalentBaseUrls(rawUrls, serviceType)

  if (deduplicated.length === 0) {
    return { baseUrl: '', baseUrls: [] }
  }

  if (deduplicated.length === 1) {
    return { baseUrl: deduplicated[0], baseUrls: [] }
  }

  return {
    baseUrl: deduplicated[0],
    baseUrls: deduplicated
  }
}
