import { computed } from 'vue'

import { usePreferencesStore } from '@/stores/preferences'

import {
  applyDocumentLanguage,
  createTranslator,
  DEFAULT_LOCALE,
  getDocumentLanguage,
  getRuntimeLocale,
  isSupportedLocale,
  normalizeLocale,
  resolveInitialLocale,
  translate,
} from './core'

export {
  applyDocumentLanguage,
  createTranslator,
  DEFAULT_LOCALE,
  getDocumentLanguage,
  getRuntimeLocale,
  isSupportedLocale,
  normalizeLocale,
  resolveInitialLocale,
  translate,
}
export type { SupportedLocale } from './messages'

export function useI18n() {
  const preferencesStore = usePreferencesStore()
  const locale = computed(() => normalizeLocale(preferencesStore.uiLanguage as unknown as string))

  const translateKey = (key: Parameters<ReturnType<typeof createTranslator>>[0], params?: Parameters<ReturnType<typeof createTranslator>>[1]) => {
    return translate(locale.value, key, params)
  }

  const setLocale = (nextLocale: typeof locale.value) => {
    preferencesStore.setUILanguage(nextLocale)
    applyDocumentLanguage(nextLocale)
  }

  return {
    locale,
    t: translateKey,
    setLocale,
  }
}
