/**
 * Vuetify 4.x + vue-tsc 6.x 类型扩展
 *
 * vue-tsc 6.x 无法自动加载 Vuetify 的 ComponentCustomProperties 扩展，
 * 因此需要在此文件中显式声明 $vuetify 属性。
 */
import type { Ref } from 'vue'

declare module 'vue' {
  interface ComponentCustomProperties {
    $vuetify: {
      display: {
        xs: boolean
        sm: boolean
        md: boolean
        lg: boolean
        xl: boolean
        xxl: boolean
        smAndUp: boolean
        mdAndUp: boolean
        lgAndUp: boolean
        xlAndUp: boolean
        smAndDown: boolean
        mdAndDown: boolean
        lgAndDown: boolean
        xlAndDown: boolean
        name: string
        width: number
        height: number
        mobile: boolean
        mobileBreakpoint: number | string
        thresholds: {
          xs: number
          sm: number
          md: number
          lg: number
          xl: number
          xxl: number
        }
      }
      theme: {
        global: {
          name: string
          current: {
            dark: boolean
            colors: Record<string, string>
          }
        }
        name: string
        current: {
          dark: boolean
          colors: Record<string, string>
        }
      }
    }
  }
}

export {}
