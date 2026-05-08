/// <reference types="vite/client" />
/// <reference types="vuetify" />

declare module 'vuetify/styles' {}

declare global {
  var __APP_UI_LANGUAGE__: string

  interface Window {
    __CCX_RUNTIME_CONFIG__?: {
      uiLanguage?: string
    }
  }
}

export {}
