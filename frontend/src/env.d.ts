/// <reference types="vite/client" />
/// <reference types="vuetify" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'

  const component: DefineComponent<object, object, any>
  export default component
}

declare module 'vuetify/styles' {}

declare var __APP_UI_LANGUAGE__: string

interface Window {
  __CCX_RUNTIME_CONFIG__?: {
    uiLanguage?: string
  }
}
