# CCX - Go 后端

> 高性能的 Claude / OpenAI Chat / Codex Responses / Gemini API Proxy 后端，Go 语言实现，支持多上游协议转换、智能调度、会话管理与嵌入式 Web 管理界面

## 特性

- ✅ **完整后端能力**：覆盖 Messages、Chat Completions、Responses、Gemini 四类 API 入口
- 🚀 **高性能**：Go 语言实现，性能优于 Node.js 版本
- 📦 **单文件部署**：前端资源嵌入二进制文件，无需额外静态资源服务
- 🔄 **四协议互转**：支持 Claude Messages、OpenAI Chat、Gemini、Responses 间的协议转换
- ⚖️ **智能调度与故障转移**：支持优先级、熔断、Trace 亲和、多密钥自动切换
- 🧪 **能力测试**：支持对单个渠道测试 Messages / Chat / Gemini / Responses 协议兼容性
- 🔍 **模型列表代理查询**：后端代查上游模型列表，避免前端 CORS 与密钥暴露问题
- 🎛️ **模型白名单**：支持 `supportedModels` 精确匹配与通配符前缀过滤
- 🌐 **渠道级代理 / 自定义请求头**：支持 `proxyUrl`、`customHeaders`
- 🖥️ **Web 管理界面**：内置前端管理界面（嵌入式）
- 🛡️ **高可用性**：健康检查、错误处理、超时重试和自动降级

## 支持的上游服务

- ✅ Claude (Anthropic)
- ✅ OpenAI Chat Completions (OpenAI)
- ✅ Codex Responses (OpenAI)
- ✅ Gemini (Google AI)

## 快速开始

### 方式1：下载预编译二进制文件（推荐）

1. 从 [Releases](https://github.com/BenedictKing/ccx/releases) 下载对应平台的二进制文件
2. 创建 `.env` 文件：

```bash
# 复制示例配置
cp .env.example .env

# 编辑配置
nano .env
```

3. 运行服务器：

```bash
# Linux / macOS
./ccx-linux-amd64

# Windows
ccx-windows-amd64.exe
```

### 方式2：从源码构建

#### 前置要求

- Go 1.25 或更高版本
- Node.js 18+ (用于构建前端)

#### 构建步骤

```bash
# 1. 克隆项目
git clone https://github.com/BenedictKing/ccx.git
cd ccx

# 2. 构建前端
cd frontend
npm install
npm run build
cd ..

# 3. 构建 Go 后端（包含前端资源）
cd backend-go
./build.sh

# 构建产物位于 dist/ 目录
```

## 配置说明

### 环境变量配置 (.env)

```env
# ============ 服务器配置 ============
PORT=3000

# 运行环境: development | production
ENV=production

# ============ Web UI 配置 ============
ENABLE_WEB_UI=true

# ============ 访问控制 ============
# 代理访问密钥（代理 API 使用，必须修改！）
PROXY_ACCESS_KEY=your-secure-access-key
# 管理访问密钥（可选；管理界面与 /api/* 使用，未设置时回退到 PROXY_ACCESS_KEY）
ADMIN_ACCESS_KEY=your-admin-access-key

# ============ 日志配置 ============
LOG_LEVEL=info
ENABLE_REQUEST_LOGS=true
ENABLE_RESPONSE_LOGS=true

# ============ 运行时配置 ============
MAX_REQUEST_BODY_SIZE_MB=50
QUIET_POLLING_LOGS=true
```

### 环境模式详解

| 配置项 | development | production |
|--------|-------------|------------|
| **Gin 模式** | DebugMode (详细日志) | ReleaseMode (高性能) |
| **开发端点** | `/admin/dev/info` 开启 | `/admin/dev/info` 关闭 |
| **CORS 策略** | 自动允许所有 localhost 源 | 严格使用 CORS_ORIGIN 配置 |
| **日志输出** | 路由注册、请求详情 | 仅错误和警告 |
| **安全性** | 低（暴露调试信息） | 高（最小信息暴露） |

**建议**：
- 开发测试时使用 `ENV=development`
- 生产部署时务必使用 `ENV=production`

### 渠道配置

服务启动后，可通过 Web 管理界面 (`http://localhost:3000`) 配置上游渠道、API 密钥、模型映射、高级选项、代理与请求头。

也可以直接编辑配置文件 `.config/config.json`。当前按 API 类型分组管理渠道，例如：

```json
{
  "messagesUpstream": [
    {
      "name": "OpenAI Messages Proxy",
      "baseUrl": "https://api.openai.com",
      "apiKeys": ["sk-your-api-key"],
      "serviceType": "openai",
      "supportedModels": ["gpt-5*"],
      "customHeaders": {
        "x-foo": "bar"
      },
      "proxyUrl": "",
      "status": "active"
    }
  ]
}
```

## 管理与调度能力

### 模型发现与过滤

- 支持通过 `/api/{type}/channels/:id/models` 由后端代理查询上游模型列表
- 后端会对 `baseUrl` 做基础校验，默认拦截云元数据地址，降低 SSRF 风险
- 每个渠道可配置 `supportedModels`
  - 空列表表示支持全部模型
  - 支持精确匹配
  - 支持前缀通配符，如 `claude-*`、`gpt-5*`、`gemini-3*`
- 调度器选路时会自动跳过不支持当前请求模型的渠道

### 能力测试

- 支持通过 `/api/{type}/channels/:id/capability-test` 测试单个渠道的协议兼容性
- 覆盖 Messages / Chat / Gemini / Responses 四类协议
- 返回可用性、延迟、流式支持、成功模型和错误分类
- 内置短时缓存，避免频繁触发重复测试

### 渠道级代理与自定义请求头

- `proxyUrl`：为单个渠道配置 HTTP / SOCKS5 代理
- `customHeaders`：为单个渠道附加或覆盖上游请求头

### 渠道状态自动变化

以下场景会触发渠道状态的自动变化：

| 场景 | 触发条件 | 自动行为 |
|------|----------|----------|
| **单 Key 更换自动激活** | 渠道只有 1 个 Key，且更新为不同的 Key | 1. 状态从 `suspended` 变为 `active`<br>2. 重置熔断状态（清除错误计数） |
| **熔断自动恢复** | 渠道熔断后超过恢复时间（默认 15 分钟） | 自动清除熔断标记，渠道恢复可用 |
| **无 Key 自动暂停** | 渠道配置为 `active` 但没有 API Key | 状态自动设为 `suspended` |

**设计说明：**
- 单 Key 更换时自动激活，因为用户明显想要使用新 Key
- 多 Key 场景不会自动激活，避免误操作（用户可能只是添加/删除部分 Key）
- `disabled` 状态不受影响，用户主动禁用的渠道不会被自动激活

### 渠道促销期（Promotion）

促销期机制用于临时提升某个渠道的调度优先级，使其在有限时间内被优先选择。

**核心规则：**
- 促销期内渠道优先级最高，绕过 Trace 亲和性检查
- 同一时间只允许一个渠道处于促销期（设置新渠道会自动清除旧渠道）
- 促销渠道若不健康（熔断/无可用密钥），在本次请求中自动跳过
- 暂停（suspended）渠道时自动清除其促销期

**自动触发场景：**

| 场景 | 触发条件 | 自动行为 |
|------|----------|----------|
| **新建渠道** | 通过 Web UI 添加新渠道 | 自动设置 5 分钟促销期 |

**API 用法：**

```bash
# 设置促销期（所有渠道类型均支持，路径中 type 可选 messages/responses/chat/gemini）
curl -X POST http://localhost:3000/api/messages/channels/0/promotion \
  -H "x-api-key: your-admin-access-key" \
  -H "Content-Type: application/json" \
  -d '{"duration": 300}'   # 单位：秒，0 表示清除

# 清除促销期
curl -X POST http://localhost:3000/api/messages/channels/0/promotion \
  -H "x-api-key: your-admin-access-key" \
  -H "Content-Type: application/json" \
  -d '{"duration": 0}'
```

**适用场景：**
- 新增渠道后临时提升优先级进行测试
- 更换 Key 后验证新 Key 是否正常工作
- 临时将流量切换到特定渠道

## 使用方法

### 访问 Web 管理界面

打开浏览器访问: http://localhost:3000

- 管理界面默认使用 `PROXY_ACCESS_KEY`
- 如果配置了 `ADMIN_ACCESS_KEY`，则管理界面与 `/api/*` 改用独立管理密钥

### API 入口概览

| 端点 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查（无需认证） |
| `/v1/messages` | POST | Claude Messages API |
| `/v1/messages/count_tokens` | POST | Messages Token 计数 |
| `/v1/chat/completions` | POST | OpenAI Chat Completions API |
| `/v1/responses` | POST | Codex Responses API |
| `/v1/responses/compact` | POST | 精简版 Responses API |
| `/v1/models` | GET | 模型列表查询 |
| `/v1beta/models/{model}:generateContent` | POST | Gemini 原生协议 |

### 管理 API 概览

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/messages/channels` | CRUD | Messages 渠道管理 |
| `/api/responses/channels` | CRUD | Responses 渠道管理 |
| `/api/chat/channels` | CRUD | Chat 渠道管理 |
| `/api/gemini/channels` | CRUD | Gemini 渠道管理 |
| `/api/messages/channels/dashboard?type=...` | GET | 统一 dashboard |
| `/api/{type}/channels/:id/models` | POST | 查询单渠道上游模型列表 |
| `/api/{type}/channels/:id/capability-test` | POST | 渠道能力测试 |
| `/api/{type}/channels/:id/promotion` | POST | 渠道促销期管理 |

### API 调用示例

#### Messages

```bash
curl -X POST http://localhost:3000/v1/messages \
  -H "x-api-key: your-proxy-access-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Hello, Claude!"}
    ]
  }'
```

#### Chat Completions

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "x-api-key: your-proxy-access-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

#### 管理 API：能力测试

```bash
curl -X POST http://localhost:3000/api/messages/channels/1/capability-test \
  -H "x-api-key: your-admin-access-key"
```

## 架构对比

| 特性 | TypeScript 版本 | Go 版本 |
|------|----------------|---------|
| 运行时 | Node.js/Bun | Go (编译型) |
| 性能 | 中等 | 高 |
| 内存占用 | 较高 | 较低 |
| 部署 | 需要 Node.js 环境 | 单文件可执行 |
| 启动速度 | 较慢 | 快速 |
| 并发处理 | 事件循环 | Goroutine（原生并发）|

## 目录结构

```
backend-go/
├── main.go                 # 主程序入口
├── go.mod                  # Go 模块定义
├── build.sh                # 构建脚本
├── internal/
│   ├── config/             # 配置管理
│   │   ├── env.go          # 环境变量配置
│   │   └── config.go       # 配置文件管理
│   ├── providers/          # 上游服务适配器
│   │   ├── provider.go     # Provider 接口
│   │   ├── openai.go       # OpenAI 适配器
│   │   ├── gemini.go       # Gemini 适配器
│   │   └── claude.go       # Claude 适配器
│   ├── middleware/         # HTTP 中间件
│   │   ├── cors.go         # CORS 中间件
│   │   └── auth.go         # 认证中间件
│   ├── handlers/           # HTTP 处理器
│   │   ├── health.go       # 健康检查
│   │   ├── config.go       # 配置管理 API
│   │   ├── proxy.go        # 代理处理逻辑
│   │   └── frontend.go     # 前端资源服务
│   └── types/              # 类型定义
│       └── types.go        # 请求/响应类型
└── frontend/dist/          # 嵌入的前端资源（构建时生成）
```

## 性能优化

Go 版本相比 TypeScript 版本的性能优势：

1. **更低的内存占用**：Go 的垃圾回收机制更高效
2. **更快的启动速度**：编译型语言，无需运行时解析
3. **更好的并发性能**：原生 Goroutine 支持
4. **更小的部署包**：单文件可执行，无需 node_modules

## 常见问题

### 1. 如何更新前端资源？

重新构建前端后，运行 `./build.sh` 重新打包。

### 2. 如何禁用 Web UI？

在 `.env` 文件中设置 `ENABLE_WEB_UI=false`

### 3. 支持热重载配置吗？

支持！配置文件（`.config/config.json`）变更会自动重载，无需重启服务器。

### 4. 如何添加自定义上游服务？

实现 `providers.Provider` 接口并在 `providers.GetProvider` 中注册即可。

## 开发

### 🔥 热重载开发模式（新增）

Go 版本现在支持代码热重载，修改代码后自动重新编译和重启！

#### 安装热重载工具

```bash
# 方式一：使用 make（推荐）
make install-air

# 方式二：使用 npm/bun
npm run dev:go:install

# 方式三：直接安装
go install github.com/air-verse/air@latest
```

#### 启动热重载开发模式

```bash
# 方式一：使用 make（推荐）
make dev              # 自动检测并安装 Air，启动热重载

# 方式二：使用 npm/bun
npm run dev:go        # 或 bun run dev:go

# 方式三：直接使用 air
cd backend-go && air
```

**热重载特性：**
- ✅ **自动重启** - 修改 `.go` 文件后自动重新编译和重启
- ✅ **配置监听** - 修改 `.yaml`, `.toml`, `.env` 文件也会触发重启
- ✅ **错误恢复** - 编译错误时保持运行，修复后自动恢复
- ✅ **彩色日志** - 不同类型日志使用不同颜色，便于调试
- ✅ **性能优化** - 1秒延迟编译，避免频繁重启

### 推荐开发流程（智能缓存）

```bash
# 使用 Makefile - 自动管理前端构建缓存
make dev              # 🔥 热重载开发模式（推荐）
make run              # 首次构建前端，后续仅在源文件变更时重新编译
make build            # 构建生产版本
make clean            # 清除所有构建缓存和临时文件

# 手动控制
make build-local      # 构建本地版本
make test             # 运行测试
make test-cover       # 生成测试覆盖率报告
make fmt              # 格式化代码
make lint             # 代码检查
make deps             # 更新依赖
```

**智能缓存机制：**
- ✅ `make run` 自动检测 `frontend/src` 目录文件变更
- ✅ 未变更时跳过编译，**秒级启动**服务器
- ✅ 首次运行或源文件修改后自动重新编译
- ✅ 使用标记文件 `.build-marker` 追踪构建状态

### Air 配置说明

`.air.toml` 文件定义了热重载行为：

```toml
# 监听的文件类型
include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml", "toml", "env"]

# 忽略的目录
exclude_dir = ["assets", "tmp", "vendor", "frontend", "dist"]

# 编译延迟（毫秒）
delay = 1000

# 编译错误时是否停止
stop_on_error = true
```

### 传统开发方式

```bash
# 直接运行（不推荐 - 无版本信息）
go run main.go

# 运行测试
go test ./...

# 格式化代码
go fmt ./...

# 静态检查
go vet ./...
```

### 开发技巧

1. **使用热重载**：`make dev` 启动后，专注于代码编写，无需手动重启
2. **查看日志**：热重载模式下日志有颜色区分，更易阅读
3. **错误处理**：编译错误会显示在控制台，修复后自动重新编译
4. **配置更新**：修改 `.env` 或配置文件也会触发重启

## 版本管理

### 升级版本

只需修改根目录的 `VERSION` 文件：

```bash
# 编辑 VERSION 文件
echo "v1.1.0" > ../VERSION

# 重新构建即可
make build
```

所有构建产物会自动包含新版本号，无需修改代码！

### 查看版本信息

```bash
# 查看项目版本信息
make info

# 启动服务器后查看版本
curl http://localhost:3000/health | jq '.version'

# 输出示例：
# {
#   "version": "v1.0.0",
#   "buildTime": "2025-01-15_10:30:45_UTC",
#   "gitCommit": "abc1234"
# }
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

---

**注意**: 这是 CCX 的 Go 语言重写版本，完整实现了原 TypeScript 版本的所有功能，并提供了更好的性能和部署体验。
