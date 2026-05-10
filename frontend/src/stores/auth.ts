import { defineStore } from 'pinia'

interface AuthState {
  apiKey: string | null
  authError: string
  authAttempts: number
  authLockoutTime: number | null
  isAutoAuthenticating: boolean
  isInitialized: boolean
  authLoading: boolean
  authKeyInput: string
}

/**
 * 认证状态管理 Store
 *
 * 职责：
 * - 管理 API Key 的存储和读取
 * - 管理认证错误和安全状态（失败次数、锁定时间等）
 * - 管理认证 UI 状态（加载中、自动认证等）
 * - 提供响应式的认证状态
 * - 自动持久化到 localStorage
 */
export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    apiKey: null,
    authError: '',
    authAttempts: 0,
    authLockoutTime: null,
    isAutoAuthenticating: true,
    isInitialized: false,
    authLoading: false,
    authKeyInput: '',
  }),

  getters: {
    isAuthenticated: (state): boolean => !!state.apiKey,

    isAuthLocked: (state): boolean => {
      if (!state.authLockoutTime) return false
      return Date.now() < state.authLockoutTime
    },
  },

  actions: {
    setApiKey(key: string | null) {
      this.apiKey = key
      // 同时保存到旧的 localStorage key 以保持兼容性
      if (key) {
        localStorage.setItem('proxyAccessKey', key)
      } else {
        localStorage.removeItem('proxyAccessKey')
      }
    },

    clearAuth() {
      this.apiKey = null
      // 清除旧的 localStorage key
      localStorage.removeItem('proxyAccessKey')
    },

    initializeAuth() {
      // 优先从旧的 localStorage key 读取（兼容性）
      const oldKey = localStorage.getItem('proxyAccessKey')
      if (oldKey) {
        this.apiKey = oldKey
      }

      // 如果没有旧 key，尝试从 Pinia 持久化恢复
      // （由 persistedstate 插件自动处理）
    },

    setAuthError(error: string) {
      this.authError = error
    },

    incrementAuthAttempts() {
      this.authAttempts++
    },

    resetAuthAttempts() {
      this.authAttempts = 0
    },

    setAuthLockout(lockoutTime: Date | null) {
      this.authLockoutTime = lockoutTime ? lockoutTime.getTime() : null
    },

    setAutoAuthenticating(value: boolean) {
      this.isAutoAuthenticating = value
    },

    setInitialized(value: boolean) {
      this.isInitialized = value
    },

    setAuthLoading(value: boolean) {
      this.authLoading = value
    },

    setAuthKeyInput(value: string) {
      this.authKeyInput = value
    },
  },

  // 持久化配置
  persist: {
    key: 'ccx-auth',
    storage: localStorage,
    // 仅持久化必要字段，排除瞬态 UI 状态和敏感输入
    pick: ['apiKey', 'authAttempts', 'authLockoutTime'],
  },
})
