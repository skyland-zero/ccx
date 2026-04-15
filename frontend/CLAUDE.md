# frontend 模块文档

[← 根目录](../CLAUDE.md)

## 模块职责

Vue 3 + Vuetify 3 Web 管理界面：渠道配置、能力测试、模型列表查询、拖拽排序、监控面板、主题切换。

## 启动命令

```bash
bun run dev       # 开发服务器
bun run build     # 生产构建
bun run preview   # 预览构建
```

## 核心组件

| 组件 | 职责 |
|------|------|
| `App.vue` | 根组件，认证、布局、主题、能力测试入口 |
| `ChannelOrchestration.vue` | 渠道编排主界面 |
| `ChannelCard.vue` | 渠道卡片（状态、密钥、指标） |
| `AddChannelModal.vue` | 添加/编辑渠道对话框（模型映射、高级选项、模型过滤等） |
| `CapabilityTestDialog.vue` | 渠道能力测试结果对话框 |

## API 服务

`src/services/api.ts` 封装后端交互，涵盖：

- 渠道 CRUD、排序、状态切换
- 单渠道模型列表查询
- 渠道能力测试
- Dashboard / 指标 / 历史数据查询
- 认证相关请求头管理

## 主题配置

编辑 `src/plugins/vuetify.ts` 中的 `lightTheme` 和 `darkTheme`。

## Vuetify 组件系统

项目使用 **按需导入** 方案，手动注册使用的 Vuetify 组件和指令，显著减小打包体积（首屏 JS 减少约 60%）。

**配置文件**: `src/plugins/vuetify.ts`

**新增组件步骤**:
1. 从 `vuetify/components/V<Component>` 添加导入
2. 在 `createVuetify({ components: { ... } })` 中注册

```typescript
// 1. 导入
import { VBadge } from 'vuetify/components/VBadge'

// 2. 注册
export default createVuetify({
  components: {
    // ... 现有组件
    VBadge,  // 新增
  },
  // ...
})
```

**常见组件路径**:
- 布局: `VApp`, `VMain`, `VContainer`, `VRow`, `VCol` 等
- 表单: `VTextField`, `VSelect`, `VSwitch`, `VBtn` 等
- 反馈: `VAlert`, `VSnackbar`, `VDialog`, `VTooltip` 等

**注意**: 如果模板中使用了未注册的组件，运行时会报 `Unknown custom element` 错误。

## 图标系统

项目使用 **SVG 按需导入** 方案，从 `@mdi/js` 导入单个图标 path，而非完整字体文件，显著减小打包体积。

**配置文件**: `src/plugins/vuetify.ts`

### 重要规则（必须遵守）

- 在模板里写了 `<v-icon>mdi-xxx</v-icon>` **并不代表可直接使用**
- **每新增一个图标，必须同时完成两步**，缺一不可：
  1. 从 `@mdi/js` 添加导入（驼峰命名）
  2. 在 `iconMap` 中添加映射（kebab-case）
- 只写模板、不补 `iconMap`，运行时会走到 `src/plugins/vuetify.ts` 的未找到图标分支，开发环境会报警告，界面可能显示占位文本而不是图标
- 修改前端图标相关代码时，应顺手检查新增的 `mdi-xxx` 是否已在 `iconMap` 中注册

**新增图标步骤**:
1. 从 `@mdi/js` 添加导入（驼峰命名）
2. 在 `iconMap` 中添加映射（kebab-case）

```typescript
// 1. 导入
import { mdiNewIcon } from '@mdi/js'

// 2. 映射
const iconMap = {
  'new-icon': mdiNewIcon,
}
```

**使用方式**: 模板中使用 `mdi-xxx` 格式
```vue
<v-icon>mdi-new-icon</v-icon>
```

**图标查找**: https://pictogrammers.com/library/mdi/

## 构建产物

生产构建输出到 `dist/`，会被嵌入到 Go 后端二进制文件中（`embed.FS`）。
