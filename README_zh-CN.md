# AI Dandelion 后端

AI Dandelion 是面向 AI 辅助业务功能搭建与运行的多服务后端，提供 HTTP 网关、gRPC 服务、认证与权限、AI Agent 执行、功能生成、生成应用运行时支持、文件上传、通知和数据持久化能力。

English documentation: [README.md](README.md)

## 关联仓库

浏览器端位于独立仓库：[gly-hub/ai-dandelion-web](https://github.com/gly-hub/ai-dandelion-web)。前端负责 React 界面、浏览器端登录态、API 调用、流式对话展示和生成应用渲染。

本仓库提供该前端消费的 HTTP API 和实时连接端点。建议将两个仓库克隆到同一目录下进行本地联调：

```text
workspace/
├── ai-dandelion/       # 当前仓库：后端服务与契约
└── ai-dandelion-web/   # React + Vite 前端
```

```bash
git clone https://github.com/gly-hub/ai-dandelion.git
git clone https://github.com/gly-hub/ai-dandelion-web.git
```

## 架构

```text
浏览器（ai-dandelion-web，React + Vite）
        |
        | HTTP / WebSocket
        v
inner-gateway（:8086，Fiber）
        |
        +--> ai-agent       （:50051，gRPC）
        +--> func-operation （:50052，gRPC）
        +--> system         （:50053，gRPC）
```

| 组件 | 职责 |
| --- | --- |
| `inner-gateway/` | HTTP API、认证边界、实时 WebSocket 入口，以及 HTTP 到 gRPC 的适配。 |
| `ai-agent/` | AI 对话、会话、技能、MCP 服务配置和 Agent 执行。 |
| `func-operation/` | 功能生命周期、生成应用产物、发布版本、公共配置和外部 API 集成。 |
| `system/` | 用户、角色、菜单、模型配置、上传、通知和操作日志。 |
| `proto/` | 网关与各 gRPC 服务共享的 protobuf 契约。 |
| `toolbox/` | Agent、认证、存储、事件总线、锁和其他公共能力。 |

## 环境要求

- Go `1.26.3`，或使用 `go.mod` 声明的兼容版本。
- Node.js LTS 与 npm，用于同级前端仓库。
- Redis 与 etcd，用于服务发现和实时/事件能力。
- 示例配置使用 SQLite；MySQL、MongoDB、OpenTelemetry 和 S3 兼容对象存储按需接入。
- 本项目约定使用的 `rtk` 命令包装器。

## 本地联调

### 1. 创建本地服务配置

每个受跟踪的 `configs_example.yaml` 都是模板。启动服务前，将它复制为被 Git 忽略的 `configs_local.yaml`：

```bash
cp ai-agent/config/configs_example.yaml ai-agent/config/configs_local.yaml
cp func-operation/config/configs_example.yaml func-operation/config/configs_local.yaml
cp system/config/configs_example.yaml system/config/configs_local.yaml
cp inner-gateway/config/configs_example.yaml inner-gateway/config/configs_local.yaml
```

根据本机环境修改复制后的数据库路径、基础设施地址、允许的前端来源和凭据。`system` 与 `inner-gateway` 的 JWT 签名密钥必须完全一致。

### 2. 启动后端服务

在不同终端中按以下顺序启动：

```bash
rtk go run ./system -c system/config/configs_local.yaml
rtk go run ./ai-agent -c ai-agent/config/configs_local.yaml
rtk go run ./func-operation -c func-operation/config/configs_local.yaml
rtk go run ./inner-gateway -c inner-gateway/config/configs_local.yaml
```

前端只应访问网关 `http://127.0.0.1:8086`，不应直接调用内部 gRPC 服务。

### 3. 启动前端

进入同级前端目录：

```bash
cd ../ai-dandelion-web
rtk npm install
rtk npm run dev
```

Vite 默认在 `http://localhost:5173` 提供开发页面，并将 HTTP API 路径代理到本网关。默认开发 WebSocket 会在通过代理申请连接凭证后，直连 `ws://127.0.0.1:8086/realtime/ws`。前端界面和构建详情请参考前端仓库的 [README](https://github.com/gly-hub/ai-dandelion-web/blob/main/README_zh-CN.md)。

## 部署关系

前端调用的是根路径 HTTP API。生产环境应在同一公开域名下托管前端构建产物，并将 `/ai-agent`、`/func-operation`、`/system`、`/realtime` 转发到 `inner-gateway`。默认 WebSocket 地址由该公开域名推导；如必须使用独立 WebSocket 地址，请在构建前端时设置 `VITE_REALTIME_WS_URL`。

网关的 CORS 与实时连接 `allowedOrigins` 必须配置为实际部署的前端域名。

## 开发与校验

```bash
# 检查 Go 包解析
rtk env GOCACHE=/private/tmp/ai-dandelion-go-cache go list ./...

# 运行后端测试
rtk env GOCACHE=/private/tmp/ai-dandelion-go-cache go test ./...

# 修改 proto 后重新生成 ai-agent 代码
rtk bash proto/gogo.sh ai-agent

# 仅生成一个 proto 文件
rtk bash proto/gogo.sh ai-agent session.proto
```

修改 API 契约时先改 `proto/`，通过生成脚本更新 `*.pb.go`，不要手工编辑生成文件。网关只负责适配；业务规则放在 `logic`，数据库访问放在 `dao`，数据定义放在 `model`。

## 配置与安全

- `configs_local.yaml` 已被 Git 忽略，正常切换分支不会删除它；不要将凭据写入受 Git 跟踪的文件。
- `configs_example.yaml` 仅用于说明必填结构并使用占位值，不得存放真实凭据。
- 除非已备份本地配置，否则不要执行 `git clean -fdX`，该命令会删除被忽略的文件。
- API Key、对象存储凭据、JWT 签名密钥、桥接令牌和模型 Token 不得提交到版本库；若曾进入 Git 历史，应立即轮换。
- 公开部署时不要使用仅适合开发环境的 Agent 权限模式。
- 生成应用会执行不受信任的业务代码，发布前应审查产物和运行时权限。

## 许可证

当前仓库尚未声明许可证。公开分发前请补齐许可证、安全漏洞报告方式、贡献指南和 CI 工作流。
