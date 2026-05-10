import type { Channel } from '../services/api'
import { normalizeAdvancedChannelOptions } from './channelAdvancedOptions'
import { deduplicateEquivalentBaseUrls } from './baseUrlSemantics'

export interface ChannelFormLike {
  name: string
  serviceType: 'openai' | 'gemini' | 'claude' | 'responses' | ''
  baseUrl: string
  baseUrls: string[]
  website: string
  insecureSkipVerify: boolean
  lowQuality: boolean
  injectDummyThoughtSignature: boolean
  stripThoughtSignature: boolean
  description: string
  apiKeys: string[]
  modelMapping: Record<string, string>
  reasoningMapping: Record<string, 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'>
  textVerbosity: 'low' | 'medium' | 'high' | ''
  fastMode: boolean
  customHeaders: Record<string, string>
  proxyUrl: string
  routePrefix: string
  supportedModels: string[]
  autoBlacklistBalance: boolean
  normalizeMetadataUserId: boolean
  normalizeNonstandardChatRoles?: boolean

}

export function buildChannelPayload(form: ChannelFormLike): Omit<Channel, 'index' | 'latency' | 'status'> {
  const processedApiKeys = form.apiKeys.filter(key => key.trim())
  const advancedOptions = normalizeAdvancedChannelOptions(form.serviceType, {
    reasoningMapping: form.reasoningMapping,
    textVerbosity: form.textVerbosity,
    fastMode: form.fastMode
  })

  const sourceUrls = form.baseUrls.length > 0 ? form.baseUrls : [form.baseUrl]
  const deduplicatedUrls = deduplicateEquivalentBaseUrls(sourceUrls, form.serviceType)

  const channelData: Omit<Channel, 'index' | 'latency' | 'status'> = {
    name: form.name.trim(),
    serviceType: form.serviceType as 'openai' | 'gemini' | 'claude' | 'responses',
    baseUrl: deduplicatedUrls[0] || '',
    website: form.website.trim(),
    insecureSkipVerify: form.insecureSkipVerify,
    lowQuality: form.lowQuality,
    injectDummyThoughtSignature: form.injectDummyThoughtSignature,
    stripThoughtSignature: form.stripThoughtSignature,
    description: form.description.trim(),
    apiKeys: processedApiKeys,
    modelMapping: form.modelMapping,
    reasoningMapping: advancedOptions.reasoningMapping,
    textVerbosity: advancedOptions.textVerbosity,
    fastMode: advancedOptions.fastMode,
    customHeaders: form.customHeaders,
    proxyUrl: form.proxyUrl.trim(),
    routePrefix: form.routePrefix.trim(),
    supportedModels: form.supportedModels,
    autoBlacklistBalance: form.autoBlacklistBalance,
    normalizeMetadataUserId: form.normalizeMetadataUserId,
    normalizeNonstandardChatRoles: !!form.normalizeNonstandardChatRoles,
  }

  if (deduplicatedUrls.length > 1) {
    channelData.baseUrls = deduplicatedUrls
  }

  return channelData
}
