<template>
  <v-dialog :model-value="modelValue" max-width="800" @update:model-value="$emit('update:modelValue', $event)">
    <v-card>
      <v-card-title class="d-flex align-center justify-space-between">
        <span class="dialog-title">{{ t('channelLogs.title', { channel: channelName }) }}</span>
        <v-btn icon size="small" variant="text" @click="$emit('update:modelValue', false)">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </v-card-title>
      <v-divider />
      <v-card-text class="pa-0 channel-logs-scroll">
        <!-- Loading -->
        <div v-if="isLoading && !logs.length" class="d-flex justify-center py-8">
          <v-progress-circular indeterminate color="primary" />
        </div>

        <!-- Empty -->
        <div v-else-if="!logs.length" class="text-center py-8 text-medium-emphasis">
          <v-icon size="40">mdi-format-list-bulleted</v-icon>
          <div class="text-caption mt-2">{{ t('channelLogs.empty') }}</div>
        </div>

        <!-- Log list -->
        <v-list v-else density="comfortable" class="pa-0">
          <template v-for="(log, i) in logs" :key="i">
            <v-list-item :class="['log-item', { 'bg-error-subtle': log.status === 'failed' }]" @click="toggleExpand(i)">
              <template #prepend>
                <v-chip
                  v-if="log.statusCode > 0"
                  :color="statusColor(log.statusCode)"
                  size="small"
                  variant="flat"
                  class="mr-2 font-weight-bold log-status-chip"
                  :class="{ 'log-status-chip--in-progress': isInProgress(log.status) }"
                >
                  {{ log.statusCode }}
                </v-chip>
                <v-chip
                  v-else-if="isInProgress(log.status)"
                  size="small"
                  variant="flat"
                  class="mr-2 font-weight-bold log-status-chip log-status-chip--placeholder log-status-chip--in-progress"
                >
                  <span class="log-status-chip__placeholder">000</span>
                </v-chip>
                <v-chip v-else size="small" color="default" variant="flat" class="mr-2 font-weight-bold log-status-chip">
                  -
                </v-chip>
              </template>
              <v-list-item-title class="d-flex align-center ga-2 flex-wrap log-summary">
                <span class="text-medium-emphasis log-meta">{{ formatTime(log.timestamp) }}</span>
                <v-chip v-if="log.status" size="small" :color="requestStatusColor(log.status)" variant="tonal" class="text-uppercase">
                  {{ requestStatusText(log.status) }}
                </v-chip>
                <v-chip v-if="log.interfaceType" size="small" :color="interfaceTypeColor(log.interfaceType)" variant="tonal" class="text-uppercase">
                  {{ log.interfaceType }}
                </v-chip>
                <v-chip v-if="log.requestSource === 'capability_test'" size="small" color="warning" variant="tonal">
                  {{ t('channelLogs.sourceCapabilityTest') }}
                </v-chip>
                <span v-if="log.originalModel" class="text-medium-emphasis log-meta">{{ log.originalModel }} →</span>
                <span class="font-weight-medium log-model">{{ log.model }}</span>
                <code class="text-caption bg-surface pa-1 rounded log-inline-code log-key-mask">{{ log.keyMask }}</code>
                <code v-if="log.baseUrl" class="text-caption bg-surface pa-1 rounded log-inline-code log-base-url" :title="log.baseUrl">{{ log.baseUrl }}</code>
                <v-chip v-if="log.isRetry" size="small" color="warning" variant="tonal">{{ t('channelLogs.retry') }}</v-chip>
                <template v-if="calculateDurations(log)">
                  <span v-if="calculateDurations(log)!.connectMs !== null" class="text-medium-emphasis log-meta">
                    连接 {{ formatDurationSeconds(calculateDurations(log)!.connectMs!) }}
                  </span>
                  <span v-if="calculateDurations(log)!.firstByteMs !== null" class="text-medium-emphasis log-meta">
                    首字 {{ formatDurationSeconds(calculateDurations(log)!.firstByteMs!) }}
                  </span>
                  <span v-if="calculateDurations(log)!.totalMs !== null" class="text-medium-emphasis log-meta">
                    总计 {{ formatDurationSeconds(calculateDurations(log)!.totalMs!) }}
                  </span>
                </template>
                <span v-else class="text-medium-emphasis log-meta">{{ formatDurationSeconds(log.durationMs) }}</span>
              </v-list-item-title>
            </v-list-item>
            <!-- 展开的错误详情 -->
            <v-expand-transition>
              <div v-if="expandedIndex === i && log.errorInfo" class="px-4 py-2 log-error-info">
                {{ log.errorInfo }}
              </div>
            </v-expand-transition>
            <v-divider v-if="i < logs.length - 1" />
          </template>
        </v-list>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { api, type ChannelLogEntry } from '../services/api'
import { useI18n } from '../i18n'

const props = defineProps<{
  modelValue: boolean
  channelIndex: number
  channelName: string
  channelType: 'messages' | 'chat' | 'responses' | 'gemini' | 'images'
}>()

defineEmits<{
  (_e: 'update:modelValue', _v: boolean): void
}>()
const { t } = useI18n()

const logs = ref<ChannelLogEntry[]>([])
const isLoading = ref(false)
const autoRefresh = ref(true)
const expandedIndex = ref<number | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

const toggleExpand = (i: number) => {
  expandedIndex.value = expandedIndex.value === i ? null : i
}

const statusColor = (code: number): string => {
  if (code >= 200 && code < 300) return 'success'
  if (code >= 400 && code < 500) return 'warning'
  return 'error'
}

const requestStatusColor = (status: string): string => {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'error'
    case 'cancelled':
    case 'canceled': return 'warning'
    case 'streaming': return 'info'
    case 'first_byte': return 'primary'
    case 'connecting': return 'warning'
    case 'pending': return 'default'
    default: return 'default'
  }
}

const requestStatusText = (status: string): string => {
  switch (status) {
    case 'pending': return '等待中'
    case 'connecting': return '连接中'
    case 'first_byte': return '首字节'
    case 'streaming': return '传输中'
    case 'completed': return '已完成'
    case 'failed': return '失败'
    case 'cancelled':
    case 'canceled': return '已取消'
    default: return status
  }
}

const isInProgress = (status: string): boolean => {
  return ['pending', 'connecting', 'first_byte', 'streaming'].includes(status)
}

const calculateDurations = (log: ChannelLogEntry) => {
  if (!log.startTime) return null

  const start = new Date(log.startTime).getTime()
  const connected = log.connectedAt ? new Date(log.connectedAt).getTime() : null
  const firstByte = log.firstByteAt ? new Date(log.firstByteAt).getTime() : null
  const completed = log.completedAt ? new Date(log.completedAt).getTime() : null

  return {
    connectMs: connected ? connected - start : null,
    firstByteMs: firstByte ? firstByte - start : null,
    totalMs: completed ? completed - start : null
  }
}

const formatDurationSeconds = (durationMs: number): string => {
  const seconds = durationMs / 1000
  return `${Number.parseFloat(seconds.toPrecision(3))}s`
}

const interfaceTypeColor = (type: string): string => {
  switch (type.toLowerCase()) {
    case 'messages': return 'primary'
    case 'chat': return 'success'
    case 'responses': return 'secondary'
    case 'gemini': return 'info'
    default: return 'default'
  }
}

const formatTime = (ts: string): string => {
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const fetchLogs = async () => {
  isLoading.value = true
  try {
    const res = await api.getChannelLogs(props.channelType, props.channelIndex)
    logs.value = res.logs || []
  } catch (e) {
    console.error('Failed to fetch channel logs:', e)
  } finally {
    isLoading.value = false
  }
}

const startPolling = () => {
  stopPolling()
  timer = setInterval(fetchLogs, 3000)
}
const stopPolling = () => { if (timer) { clearInterval(timer); timer = null } }

// 打开时加载，关闭时停止
watch(() => props.modelValue, (open) => {
  if (open) {
    logs.value = []
    expandedIndex.value = null
    fetchLogs()
    if (autoRefresh.value) startPolling()
  } else {
    stopPolling()
  }
})

// 对话框打开状态下切换渠道时重新加载
watch([() => props.channelIndex, () => props.channelType], () => {
  if (props.modelValue) {
    logs.value = []
    expandedIndex.value = null
    fetchLogs()
  }
})

watch(autoRefresh, (v) => {
  if (v && props.modelValue) startPolling()
  else stopPolling()
})

// 对话框打开时自动开始轮询
watch(() => props.modelValue, (open) => {
  if (open && autoRefresh.value) {
    startPolling()
  }
}, { immediate: true })

onUnmounted(() => stopPolling())
</script>

<style scoped>
.auto-refresh-btn :deep(.v-btn__content) {
  font-size: 0.8125rem;
  letter-spacing: 0;
  line-height: 1.5;
}

.channel-logs-scroll {
  max-height: 500px;
  overflow-y: auto;
}

.log-item {
  padding-top: 10px;
  padding-bottom: 10px;
}

.log-status-chip {
  min-width: 52px;
  justify-content: center;
}

.log-status-chip--in-progress {
  position: relative;
  overflow: hidden;
  isolation: isolate;
  animation: log-chip-neon-pulse 1.8s ease-in-out infinite;
}

.log-status-chip--in-progress::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background:
    radial-gradient(circle at center, rgba(var(--v-theme-primary), 0.34) 0%, rgba(var(--v-theme-primary), 0.2) 45%, rgba(var(--v-theme-primary), 0.06) 100%);
  opacity: 0.88;
  z-index: -1;
}

@keyframes log-chip-neon-pulse {
  0%, 100% {
    box-shadow:
      0 0 0 1px rgba(var(--v-theme-primary), 0.28),
      0 0 10px rgba(var(--v-theme-primary), 0.22),
      0 0 18px rgba(var(--v-theme-primary), 0.12);
    filter: saturate(1);
  }
  50% {
    box-shadow:
      0 0 0 1px rgba(var(--v-theme-primary), 0.48),
      0 0 14px rgba(var(--v-theme-primary), 0.36),
      0 0 28px rgba(var(--v-theme-primary), 0.22);
    filter: saturate(1.12);
  }
}

.log-status-chip--placeholder {
  background: transparent !important;
  box-shadow: inset 0 0 0 1px rgba(var(--v-theme-on-surface), 0.12);
}

.log-status-chip__placeholder {
  opacity: 0;
  user-select: none;
}

.log-summary {
  font-size: 0.875rem;
  line-height: 1.6;
}

.log-meta {
  font-size: 0.875rem;
}

.log-inline-code {
  display: inline-block;
  font-family: ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  line-height: 1.3;
  vertical-align: middle;
}

.log-key-mask {
  white-space: nowrap;
}

.log-base-url {
  white-space: nowrap;
}

.log-model {
  font-size: 0.875rem;
}

.log-error-info {
  background: rgba(var(--v-theme-surface-variant), 0.3);
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 0.875rem;
  line-height: 1.6;
}

.bg-error-subtle {
  background: rgba(var(--v-theme-error), 0.05);
}
</style>
