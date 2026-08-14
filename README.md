# AI Dandelion

AI Dandelion is a multi-service platform for building and operating AI-assisted business functions. It combines conversational agents, function generation, generated application runtimes, user and permission management, uploads, and realtime notifications.

For Chinese documentation, see [README_zh-CN.md](README_zh-CN.md).

## Architecture

```text
Browser (React + Vite, ../ai-dandelion-web)
        |
        v
inner-gateway (Fiber HTTP gateway, :8086)
        |
        +--> ai-agent       (gRPC, :50051)
        +--> func-operation (gRPC, :50052)
        +--> system         (gRPC, :50053)
```

- `inner-gateway/`: HTTP API, authentication boundary, realtime WebSocket entry point, and gRPC adaptation.
- `ai-agent/`: AI conversations, sessions, skills, MCP server configuration, and agent execution.
- `func-operation/`: function lifecycle, generated application artifacts, releases, public configuration, and external API integration.
- `system/`: users, roles, menus, model configuration, uploads, notifications, and operation logs.
- `proto/`: protobuf contracts shared by the gateway and gRPC services.
- `toolbox/`: shared agent, authentication, storage, event bus, locking, and utility code.

## Prerequisites

- Go `1.26.3` or a compatible version declared by `go.mod`.
- Node.js LTS and npm for the sibling frontend project.
- Redis and etcd for local service discovery and realtime/event features.
- SQLite is used by the example configuration. MySQL, MongoDB, OpenTelemetry, and S3-compatible object storage can be configured when required.
- The local `rtk` command wrapper used by this project.

## Quick Start

1. Create private local configurations from the committed examples:

   ```bash
   cp ai-agent/config/configs_example.yaml ai-agent/config/configs_local.yaml
   cp func-operation/config/configs_example.yaml func-operation/config/configs_local.yaml
   cp system/config/configs_example.yaml system/config/configs_local.yaml
   cp inner-gateway/config/configs_example.yaml inner-gateway/config/configs_local.yaml
   ```

2. Update the copied files with local database paths, infrastructure endpoints, allowed browser origins, and credentials. Set the same JWT signing secret in `system` and `inner-gateway`.

3. Start the services in this order, each in a separate terminal:

   ```bash
   go run ./system -c system/config/configs_local.yaml
   go run ./ai-agent -c ai-agent/config/configs_local.yaml
   go run ./func-operation -c func-operation/config/configs_local.yaml
   go run ./inner-gateway -c inner-gateway/config/configs_local.yaml
   ```

4. Start the frontend from its sibling directory:

   ```bash
   cd ../ai-dandelion-web
   npm install
   npm run dev
   ```

The example frontend development URL is `http://localhost:5173`; the gateway listens on `http://localhost:8086`.

## Development

```bash
# Resolve all Go packages
env GOCACHE=/private/tmp/ai-dandelion-go-cache go list ./...

# Run the backend test suite
env GOCACHE=/private/tmp/ai-dandelion-go-cache go test ./...

# Regenerate all ai-agent protobuf files after editing proto sources
bash proto/gogo.sh ai-agent

# Regenerate a single protobuf file
bash proto/gogo.sh ai-agent session.proto
```

Frontend validation:

```bash
cd ../ai-dandelion-web
npm run lint
npm run build
```

## Configuration and Security

- `configs_local.yaml` files are intentionally ignored by Git. They persist across normal branch switches and must never contain committed credentials.
- `configs_example.yaml` files document the required keys and use placeholders only. Copy an example rather than editing it with environment-specific values.
- Do not run `git clean -fdX` unless local configuration files are backed up; that command removes ignored files.
- Keep API keys, object-storage credentials, JWT signing secrets, bridge tokens, and model tokens outside version control. Rotate a credential immediately if it has entered Git history.
- Restrict gateway CORS and realtime `allowedOrigins` to the deployed frontend origins. Do not use development-only Agent permission modes in a public deployment.
- Generated applications execute untrusted business code. Treat generated artifacts and their runtime permissions as a deployment boundary and review them before release.

## Repository Workflow

- API contract changes begin in `proto/`; regenerate Go code rather than editing `*.pb.go` manually.
- Keep the gateway as an HTTP-to-gRPC adapter. Put business rules in `logic`, persistence in `dao`, and models in `model`.
- Add focused tests for business logic and agent/runtime behavior alongside changes.
- Before publishing externally, add a license, security-reporting policy, contribution guide, and CI workflow appropriate for the chosen distribution model.

## License

No license has been declared in this repository yet. Choose and add a license before distributing the project publicly.
