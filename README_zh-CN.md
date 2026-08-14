# AI Dandelion

AI Dandelion 是一个面向 AI 辅助业务功能搭建与运行的平台，包含对话 Agent、功能生成、生成应用运行时、用户与权限、文件上传和实时通知等能力。

For English documentation, see [README.md](README.md).

## 架构

```text
浏览器（React + Vite，../ai-dandelion-web）
        |
        v
inner-gateway（Fiber HTTP 网关，:8086）
        |
        +--> ai-agent       （gRPC，:50051）
        +--> func-operation （gRPC，:50052）
        +--> system         （gRPC，:50053）
```

- `inner-gateway/`：HTTP API、认证边界、实时 WebSocket 入口与 gRPC 适配。
- `ai-agent/`：AI 对话、会话、技能、MCP 服务配置和 Agent 执行。
- `func-operation/`：功能生命周期、生成应用产物、发布版本、公共配置与外部 API 集成。
- `system/`：用户、角色、菜单、模型配置、上传、通知和操作日志。
- `proto/`：网关与各 gRPC 服务共享的 protobuf 契约。
- `toolbox/`：Agent、认证、存储、事件总线、锁与其他公共能力。

## 环境要求

- Go `1.26.3`，或使用 `go.mod` 声明的兼容版本。
- Node.js LTS 与 npm，用于同级前端项目。
- Redis 与 etcd，提供本地服务发现和实时/事件能力。
- 示例配置使用 SQLite；按需要可接入 MySQL、MongoDB、OpenTelemetry 和 S3 兼容对象存储。
- 本项目约定使用的 `rtk` 命令包装器。

## 本地启动

1. 从示例创建本地私有配置：

   ```bash
   cp ai-agent/config/configs_example.yaml ai-agent/config/configs_local.yaml
   cp func-operation/config/configs_example.yaml func-operation/config/configs_local.yaml
   cp system/config/configs_example.yaml system/config/configs_local.yaml
   cp inner-gateway/config/configs_example.yaml inner-gateway/config/configs_local.yaml
   ```

2. 按本机环境修改数据库路径、基础设施地址、允许的前端来源和凭据。`system` 与 `inner-gateway` 的 JWT 签名密钥必须保持一致。

3. 分别在独立终端按以下顺序启动服务：

   ```bash
   go run ./system -c system/config/configs_local.yaml
   go run ./ai-agent -c ai-agent/config/configs_local.yaml
   go run ./func-operation -c func-operation/config/configs_local.yaml
   go run ./inner-gateway -c inner-gateway/config/configs_local.yaml
   ```

4. 启动同级前端项目：

   ```bash
   cd ../ai-dandelion-web
   npm install
   npm run dev
   ```

默认前端开发地址为 `http://localhost:5173`，网关地址为 `http://localhost:8086`。

## 开发与校验

```bash
# 检查 Go 包解析
env GOCACHE=/private/tmp/ai-dandelion-go-cache go list ./...

# 运行后端测试
env GOCACHE=/private/tmp/ai-dandelion-go-cache go test ./...

# 修改 proto 后重新生成 ai-agent 代码
bash proto/gogo.sh ai-agent

# 仅生成一个 proto 文件
bash proto/gogo.sh ai-agent session.proto
```

前端校验：

```bash
cd ../ai-dandelion-web
npm run lint
npm run build
```

## 配置与安全

- `configs_local.yaml` 已被 Git 忽略，正常切换分支不会删除它们；不要把凭据写入受 Git 跟踪的文件。
- `configs_example.yaml` 仅用于说明配置结构，包含占位值。请复制示例后填写本地配置，不要直接修改示例文件保存真实信息。
- 除非已备份本地配置，否则不要执行 `git clean -fdX`，该命令会删除被忽略的文件。
- API Key、对象存储凭据、JWT 签名密钥、桥接令牌和模型 Token 不得提交到版本库；若曾进入 Git 历史，应立即轮换。
- 网关 CORS 和实时连接的 `allowedOrigins` 必须限制为实际部署的前端域名；公开部署时不要使用仅适合开发环境的 Agent 权限模式。
- 生成应用会执行不受信任的业务代码，应在发布前审查产物和运行时权限。

## 协作约定

- 修改 API 契约时先改 `proto/`，通过生成脚本更新 `*.pb.go`，不要手工编辑生成文件。
- 网关只负责 HTTP 到 gRPC 的适配；业务规则放在 `logic`，数据库访问放在 `dao`，数据模型放在 `model`。
- 业务和 Agent/运行时变更应补充有针对性的测试。
- 对外发布前请补齐许可证、安全漏洞报告方式、贡献指南和适合项目的 CI 工作流。

## 许可证

当前仓库尚未声明许可证。公开分发前请确定并添加合适的许可证。
