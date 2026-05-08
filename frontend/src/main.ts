import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import vuetify from './plugins/vuetify'
import router from './router'
import App from './App.vue'
import './assets/style.css'
import { useAuthStore } from './stores/auth'
import { usePreferencesStore } from './stores/preferences'
import { applyDocumentLanguage, getRuntimeLocale, type SupportedLocale } from './i18n'

const app = createApp(App)

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)

// TS 6.x Plugin 类型兼容（Vue 3.5 app.use 签名变更）
app.use(pinia as any)
app.use(vuetify as any)
app.use(router as any)

// 初始化 AuthStore（从 localStorage 恢复状态）
const authStore = useAuthStore()
authStore.initializeAuth()

const preferencesStore = usePreferencesStore()
preferencesStore.initializeUILanguage(getRuntimeLocale())
applyDocumentLanguage(preferencesStore.uiLanguage as unknown as SupportedLocale)

app.mount('#app')
