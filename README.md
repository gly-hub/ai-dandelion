# AI Dandelion Backend

AI Dandelion is a multi-service backend for building and operating AI-assisted business functions. It provides the HTTP gateway, gRPC services, authentication and authorization, AI-agent execution, function generation, generated-app runtime support, uploads, notifications, and the data layer used by the web application.

Chinese documentation: [README_zh-CN.md](README_zh-CN.md)

demonstration video：
https://www.bilibili.com/video/BV1A78t68EVD/?vd_source=882e22f2f6123549d1c4813f927a00ea

## Related Repository

The browser application lives in a separate repository: [gly-hub/ai-dandelion-web](https://github.com/gly-hub/ai-dandelion-web). It is responsible for the React user interface, browser authentication state, API calls, streaming chat presentation, and generated-app rendering.

The backend owns the public API and realtime endpoints consumed by that frontend. Keep both repositories checked out side by side for local development:

```text
workspace/
├── ai-dandelion/       # this repository: backend services and contracts
└── ai-dandelion-web/   # React + Vite frontend
```

```bash
git clone https://github.com/gly-hub/ai-dandelion.git
git clone https://github.com/gly-hub/ai-dandelion-web.git
```

## Architecture

```text
Browser (ai-dandelion-web, React + Vite)
        |
        | HTTP / WebSocket
        v
inner-gateway (:8086, Fiber)
        |
        +--> ai-agent       (:50051, gRPC)
        +--> func-operation (:50052, gRPC)
        +--> system         (:50053, gRPC)
```

| Component | Responsibility |
| --- | --- |
| `inner-gateway/` | HTTP API, authentication boundary, realtime WebSocket entry point, and HTTP-to-gRPC adaptation. |
| `ai-agent/` | AI conversations, sessions, skills, MCP server configuration, and agent execution. |
| `func-operation/` | Function lifecycle, generated application artifacts, releases, public configuration, and external API integration. |
| `system/` | Users, roles, menus, model configuration, uploads, notifications, and operation logs. |
| `proto/` | Protobuf contracts shared by the gateway and gRPC services. |
| `toolbox/` | Shared agent, authentication, storage, event-bus, locking, and utility code. |

## Prerequisites

- Go `1.26.3`, or a version compatible with `go.mod`.
- Node.js LTS and npm for the sibling frontend repository.
- Redis and etcd for service discovery and realtime/event features.
- SQLite is used by the example configuration. MySQL, MongoDB, OpenTelemetry, and S3-compatible object storage are optional integrations.
- The local `rtk` command wrapper used by this project.

## Local Development

### 1. Create private service configuration

Each tracked `configs_example.yaml` is a template. Copy it to the ignored `configs_local.yaml` before starting a service:

```bash
cp ai-agent/config/configs_example.yaml ai-agent/config/configs_local.yaml
cp func-operation/config/configs_example.yaml func-operation/config/configs_local.yaml
cp system/config/configs_example.yaml system/config/configs_local.yaml
cp inner-gateway/config/configs_example.yaml inner-gateway/config/configs_local.yaml
```

Update the copied files with local database paths, infrastructure endpoints, allowed browser origins, and credentials. `system.auth.access_token_secret` signs access JWTs; `inner-gateway` delegates their validation to `system`.

### 2. Start backend services

Start every service in a separate terminal, in this order:

```bash
rtk go run ./system -c system/config/configs_local.yaml
rtk go run ./ai-agent -c ai-agent/config/configs_local.yaml
rtk go run ./func-operation -c func-operation/config/configs_local.yaml
rtk go run ./inner-gateway -c inner-gateway/config/configs_local.yaml
```

The frontend should connect only to the gateway at `http://127.0.0.1:8086`; it must not call the internal gRPC services directly.

### 3. Start the frontend

In the sibling checkout:

```bash
cd ../ai-dandelion-web
rtk npm install
rtk npm run dev
```

Vite serves the development UI on `http://localhost:5173` and proxies the HTTP API paths to this gateway. The default development WebSocket connects directly to `ws://127.0.0.1:8086/realtime/ws` after requesting its ticket through the proxy. See the frontend repository's [README](https://github.com/gly-hub/ai-dandelion-web/blob/main/README.md) for UI and build details.

## Deployment Relationship

The frontend calls root-relative HTTP API paths. In production, serve the built frontend and route `/ai-agent`, `/func-operation`, `/system`, and `/realtime` to `inner-gateway` on the same public origin. Its default WebSocket URL is derived from that public origin; when a separate WebSocket endpoint is necessary, set the frontend's `VITE_REALTIME_WS_URL` during its build.

Configure the gateway CORS and realtime `allowedOrigins` with the actual deployed frontend origins.

## Development

```bash
# Resolve all Go packages
rtk env GOCACHE=/private/tmp/ai-dandelion-go-cache go list ./...

# Run the backend test suite
rtk env GOCACHE=/private/tmp/ai-dandelion-go-cache go test ./...

# Regenerate all ai-agent protobuf files after editing proto sources
rtk bash proto/gogo.sh ai-agent

# Regenerate a single protobuf file
rtk bash proto/gogo.sh ai-agent session.proto
```

API contract changes begin in `proto/`; regenerate Go code rather than editing `*.pb.go` manually. Keep the gateway as an adapter, business rules in `logic`, database access in `dao`, and database definitions in `model`.

## Configuration and Security

- `configs_local.yaml` files are intentionally ignored by Git. Normal branch switches do not remove them; never place credentials in tracked files.
- `configs_example.yaml` documents the required keys with placeholders only. Do not store real credentials in it.
- Do not run `git clean -fdX` without backing up local configuration, because it removes ignored files.
- Keep API keys, object-storage credentials, JWT signing secrets, bridge tokens, and model tokens out of version control. Rotate a credential immediately if it has entered Git history.
- Do not use development-only Agent permission modes for a public deployment.
- Generated applications execute untrusted business code. Review their artifacts and runtime permissions before release.

## License

No license has been declared in this repository yet. Add a license, security-reporting policy, contribution guide, and CI workflow before public distribution.

## Friend links
[linux.do](linux.do)