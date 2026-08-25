---
name: generated-app-builder
description: Build complete ai-dandelion func-operation generated apps. Use when Codex is asked to create, finish, repair, or improve a generated app/function under generated_apps, including frontend.js, WASI Go backend source, manifest.json, backend.wasm, database tables, GeneratedAppRenderContext integration, app registration, function binding, and end-to-end verification. Also use when generated app naming, table naming, UUID isolation, or "generated app must be directly usable rather than a text summary" is relevant.
---

# Generated App Builder

## Purpose

Create a complete, usable generated app for `ai-dandelion` func-operation in one pass. A generated app is not done until the user can open the function workspace, see the generated UI, and use real business actions backed by app-scoped data.

## Authority Scope

This skill is only authorized to implement or repair the generated app under the exact `appDir` from the prompt.

Allowed:

- Read the applied product document and applied technical document.
- Read generated-app-builder reference files and relevant runtime contracts.
- Write files only under the exact `appDir`, including `manifest.json`, `frontend.js`, `frontend/*.js`, `backend/*.go`, and `backend.wasm`.
- Run local build/check commands needed to produce or verify `backend.wasm` for that app.

Forbidden:

- Do not modify product documents under `documents/product/`.
- Do not modify technical documents under `documents/technical/`.
- Do not modify files outside `appDir`, including platform Go code, proto files, gateway/backend services, frontend host app, skill files, or global config.
- Do not change applied docs to make implementation easier. If the docs are inconsistent or impossible to implement, stop and report the blocker.
- Do not create sibling generated app folders unless the prompt omits `appDir` and explicitly asks to create a new generated app.

If a user request in the code generation conversation asks to change product requirements or technical design, do not edit those documents. Report that the request belongs to the product or technical step, or implement only what is already covered by the applied documents.

## Product Quality Bar

Build the actual operational feature, not a scaffold, explanation, or document viewer. The page must let users complete the business workflow described by the product and technical docs.

A generated app is unacceptable if it only shows:

- AI conversation summaries.
- Product or technical document excerpts.
- API or data model notes.
- Static cards with no meaningful business action.
- Placeholder text saying the feature is being generated.

The generated app must include real user operations such as create, edit, delete/archive, search/filter, status transition, detail viewing, or another workflow explicitly required by the docs.

## Func-Operation Prompt Contract

The frontend sends a short prompt like:

```text
使用 `generated-app-builder` 技能。
功能名称：...
功能描述：...
产品文档（applied）：generated_apps/<app-id>/documents/product/applied/product-doc.md
研发文档（applied）：generated_apps/<app-id>/documents/technical/applied/technical-doc.md
appId：550e8400-e29b-41d4-a716-446655440000
appDir：generated_apps/550e8400-...
tablePrefix：func_42
完成标签：<func-operation-generated-app-ready function-id="<function-id>" />
失败标签：<func-operation-generated-app-failed function-id="<function-id>" />
```

When you see this shape:

1. Read both applied documents from the given paths before writing any code.
2. Implement strictly per technical doc sections 8-11 (directory, manifest/tables, actions, frontend modules). Do not redesign structure.
3. Modify files only under `appDir`. Do not create other `generated_apps/<name>/` folders.
4. Use `appId` for manifest `id` and folder name. Treat `tablePrefix` as legacy metadata only; new apps must use manifest `dataModels`, `relations`, and `queries`.
5. Rebuild `backend.wasm` after backend changes; confirm it is newer than all `backend/*.go` files.
6. Reply briefly in chat; do not dump full source or doc content.
7. Put `完成标签` alone on the last line when files are complete and verifiable.
8. Put `失败标签` alone on the last line when blocked.
9. Never write completion or failure tags inside source files, documents, or `manifest.json`. Tags are chat-only signals.
10. If information is insufficient, ask focused follow-up questions before generating.

Do not ask the user to run curl, hit backend APIs directly, or call reload/materialize endpoints.

If the repository has `doc/func-operation/interaction-design.md`, use it as the UI behavior reference for page layout, operation flow, preview behavior, user-facing terms, and error recovery. Do not copy that document into the app; apply its conventions to the generated feature.

Before implementing permissions, actions, frontend controls, or backend dispatch, read `references/permissions.md` from this skill and follow its manifest, frontend, backend, and nested UI rules. Before implementing frontend styles, read `references/style-guide.md` from this skill and follow its tokens, layout, component, scoped CSS, and responsive rules.

## Hard Rules

- Always allocate a new feature UUID before creating files. Do not derive identity from the display name alone.
- Func-operation service tables must use `id` as the auto-increment primary key and keep the former string identifier in a separate `uuid` column. This applies to functions, generated apps, and app records.
- Use the UUID only for generated app identity: app folder, manifest `id`, DOM ids/classes where uniqueness matters, and request routing assumptions.
- Do not design business data physical tables in new apps. The platform generates storage from manifest `dataModels`.
- Keep the business name as display text only: manifest `name`, headings, labels, and user-facing copy.
- Never name app folders or logical data models only from the application/function display name. Names such as `book-management` and `app_book_management_books` are collision-prone and should only appear in old examples.
- Generate a complete directory. Do not stop after `manifest.json`, `frontend.js`, or backend source alone.
- When `appDir` is provided in the prompt, treat it as the only writable app root. Scaffold may already exist — replace placeholder imports/UI with real business code.
- The scaffold app may carry `tablePrefix` metadata but should not create placeholder business tables or DDL. Create `dataModels` only when the real business schema is known.
- The generated frontend must be a real interactive business page, not a summary of Agent output, API design, or implementation notes.
- The first screen must be usable. It should show the main records/work queue/dashboard with an obvious primary action, not a hero page or onboarding explanation.
- Every primary action in the product doc must have an implemented frontend control, API wrapper, backend action, validation, success feedback, and error feedback.
- Implement every applicable per-action log marker from Section 16 of the applied technical document. Business execution must be diagnosable from the WASM execution log rather than relying only on request/response payloads.
- Every button-permission-controlled action must be declared in `manifest.actions`; read-only actions must not be declared there. For full frontend/backend permission behavior, follow `references/permissions.md`.
- Never leave scaffold placeholder UI after real generation. Replace "功能页面正在生成" style fallback pages with the actual feature.
- Follow the shared generated-app style guide. Do not invent a one-off visual system, custom palette, marketing page style, or decorative layout unless explicitly requested.
- Frontend code must use `context.invokeData(context.app.id, payload)` or equivalent `appId = context.app.id`; do not hardcode the app id in runtime calls.
- Frontend code must use `context.can(actionKey)` to decide whether a controlled button is rendered. If an action is not listed in `manifest.actions`, treat it as not button-controlled and keep the control visible by default.
- Permission checks must be preserved through nested UI. If a controlled button is rendered inside a modal, drawer, detail panel, tab pane, nested table, or child module, pass `context.can` or an equivalent `can` function into that child renderer and check the exact action key there. Do not only check permissions in the outer list.
- Read-only entry buttons that only open a list, detail view, or management modal must remain visible by default unless their own action is declared in `manifest.actions`. The write buttons inside that nested view still need their own `context.can(...)` checks.
- Controlled action event handlers must also guard with `can(actionKey)` before invoking the backend, so manually triggered DOM events cannot call hidden write operations.
- Backend data access must use platform data capability APIs only: `data_list`, `data_get`, `data_create`, `data_update`, `data_delete`, `data_join_query`, and `data_run_query`.
- Do not generate SQL text, physical table names, table constants, `db_query`, or `db_exec` in new backend code.
- `data_run_query.query` must be a manifest `queries[].name`, never SQL text such as `SELECT ...`.
- Do not hand-edit generated `.pb.go` or unrelated service code unless the request requires platform changes.

## Naming Contract

Create names in this order:

1. Generate UUID: use `uuidgen` or another reliable local source.
2. Set `app_id = <uuid>`.
3. Set folder: `generated_apps/<uuid>/`.
4. Get the auto-increment `id` from the function record. If the model/table still uses a string primary key, first migrate it so `id` is numeric and the old string value is stored in `uuid`.
5. Model business data with logical model names such as `book`, `category`, `borrowRecord`; do not choose or reference physical table names.
6. If the UUID contains uppercase letters, lowercase it before using it in app ids and folders. Keep app ids to `[a-z0-9-]` and logical model/field names to identifier-safe camelCase or snake_case.

To reduce naming mistakes, run the bundled helper when possible:

```bash
python scripts/app_identity.py --uuid 550e8400-e29b-41d4-a716-446655440000 --id 42 --entity books --entity borrow_records
```

Use its `appId`, `folder`, and `tablePrefix` output for identity only. Do not use helper table names in new backend code.

## Required Files

Before creating files, read `references/structure.md` from this skill and follow its directory contract. Before writing manifest actions, frontend buttons, event handlers, API wrappers, or backend dispatch, read `references/permissions.md`. Before writing frontend styles, read `references/style-guide.md`.

Every completed generated app should include:

- `generated_apps/<app_id>/manifest.json`
- `generated_apps/<app_id>/frontend.js`
- `generated_apps/<app_id>/frontend/*.js` for non-trivial frontend modules such as API, state, UI, modal, and styles
- `generated_apps/<app_id>/backend/main.go`
- `generated_apps/<app_id>/backend/platform.go`
- `generated_apps/<app_id>/backend/logger.go`
- `generated_apps/<app_id>/backend/models.go`
- `generated_apps/<app_id>/backend/<business>_handlers.go` or similarly named handler file
- `generated_apps/<app_id>/backend.wasm`

If the current runtime expects a different backend source path, follow the runtime, but still keep all backend source files present and buildable.

## Func-Operation Table Contract

Before generating or testing apps, confirm these model/table shapes:

- `func_operation_functions`: `id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY`, `uuid VARCHAR(36) UNIQUE`.
- `func_operation_generated_apps`: `id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY`, `uuid VARCHAR(120) UNIQUE`.
- `func_operation_app_records`: `id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY`, `uuid VARCHAR(120) UNIQUE`, `app_uuid VARCHAR(120)`.

External API `id` values may still carry UUID strings for frontend compatibility, but database primary keys should be numeric. If an existing local database still has varchar `id` primary keys, `AutoMigrate` may not rewrite them; migrate or rebuild those tables before judging generated folder/table names.

## Manifest Requirements

Use `manifest.json` as the registry contract:

```json
{
  "id": "<uuid>",
  "name": "<用户可见功能名>",
  "version": "v0.1.0",
  "description": "<一句话说明可操作业务能力>",
  "export": "handle",
  "actions": ["book_create", "book_update", "book_delete"],
  "frontendFile": "frontend.js",
  "backendSource": "backend",
  "backendModule": "backend.wasm",
  "dataModels": [
    {
      "name": "book",
      "label": "图书",
      "fields": [
        { "name": "title", "type": "string", "required": true, "maxLength": 120 },
        { "name": "status", "type": "enum", "values": ["在库", "借出", "下架"] }
      ],
      "indexes": ["status"]
    }
  ],
  "relations": [
    {
      "name": "bookCategory",
      "from": "book.categoryId",
      "to": "category.id",
      "type": "manyToOne"
    }
  ],
  "queries": [
    {
      "name": "bookListWithCategory",
      "from": "book",
      "joins": [{ "relation": "bookCategory", "type": "left" }],
      "select": ["book.id", "book.title", "category.name"],
      "orderBy": [{ "field": "book.id", "direction": "desc" }],
      "limit": 50
    }
  ]
}
```

Data manifest rules:

- `actions` must be a JSON string array of action keys only, for example `["class_create", "class_update"]`.
- Never write action objects such as `{ "key": "...", "label": "...", "mode": "..." }` into `manifest.actions`; labels and modes belong in the technical document, not the runtime manifest.
- `dataModels[].name`, field names, relation names, and query names must be stable logical identifiers, not physical table names.
- Include every field the backend reads or writes. Platform-managed fields `id`, `uuid`, `created_at`, and `updated_at` are implicit; do not add them unless the platform contract explicitly changes.
- Use `relations` for multi-model reads. Backend code must reference relation names, not SQL `JOIN ... ON`.
- Use `queries` only for fixed predeclared reads. `data_run_query` must pass the query name exactly.
- Do not include `tables` or DDL for new apps unless repairing a legacy app that explicitly still uses raw SQL.
- `manifest.actions` is only for state-changing, button-permission-controlled actions. Do not include read-only actions such as `list`, `detail`, `query`, `search`, `page`, or `stats`.

## Agent Function Skill Contract

Every new or regenerated function must declare an `agentSkill` object in `manifest.json`. This is the published contract for the platform App Skill; it is not shown directly in the generated page. Do not omit it merely because the first workflow is a list or detail query: users need to ask the web Agent to inspect as well as change function data.

```json
{
  "agentSkill": {
    "name": "图书管理",
    "toolPrefix": "book_management",
    "description": "维护图书馆藏信息。",
    "operations": [
      {
        "key": "list_books",
        "action": "book_list",
        "effect": "read",
        "description": "按关键词、分类和状态查询图书列表",
        "fields": [
          {"key": "keyword", "label": "关键词", "type": "string", "description": "书名、作者或 ISBN"},
          {"key": "category", "label": "分类", "type": "string"},
          {"key": "status", "label": "状态", "type": "enum", "enumValues": ["在架", "下架"]}
        ]
      },
      {
        "key": "get_book",
        "action": "book_detail",
        "effect": "read",
        "description": "查看一本图书的完整资料",
        "fields": [
          {"key": "id", "label": "图书 ID", "type": "integer", "required": true}
        ]
      },
      {
        "key": "create_book",
        "action": "book_create",
        "effect": "create",
        "description": "新增一本图书",
        "autoExecute": true,
        "fields": [
          {"key": "title", "label": "书名", "type": "string", "required": true, "description": "图书标题"},
          {"key": "author", "label": "作者", "type": "string", "required": true}
        ]
      }
    ]
  }
}
```

- `toolPrefix`, operation `key`, action, and every field `key` use stable lowercase snake_case identifiers. The final MCP tool name is `<toolPrefix>__<key>`.
- Each operation must contain `key`, `action`, `effect`, `description`, and a flat `fields` list. Supported `effect` values are `read`, `create`, `update`, `delete`, and `execute`; field types are `string`, `number`, `integer`, `boolean`, and `enum` (with non-empty `enumValues`). Nested object and file fields are not supported.
- MCP tool arguments are flat and use the exact field names. The Function Skill runtime converts them to the generated app envelope `{"action":"<action>","data":toolInput}`. The backend dispatch action and frontend invocation action must therefore match the operation `action` exactly.
- Every non-read operation action must appear in `manifest.actions`. Read-only operations must still be implemented through the normal runtime access checks, but are not listed in `manifest.actions`.
- Only `effect: "create"` may set `autoExecute: true`. Updates, deletes, and execute operations always require the Agent confirmation flow; leave `autoExecute` false for them.
- Build the skill from the applied technical document's App Skill contract, never by selecting only write actions. Required capability coverage is:
  - Every main list, queue, dashboard, or searchable collection has one `effect: "read"` list operation. Include every supported search, filter, sorting, and pagination field as flat optional fields.
  - Every user-visible detail page, drawer, or modal has one `effect: "read"` detail operation with its stable record identifier as a required field.
  - Every user-triggered state-changing action that the Agent is allowed to perform has exactly one non-read operation with matching action and fields. Pure internal helper actions may be omitted only when the technical document marks them as non-exposed.
  - A skill needs at least one read operation. Use unique operation keys and action names; read actions never appear in `manifest.actions`.
- Before building, verify three-way parity: technical document App Skill table -> `manifest.agentSkill.operations` -> frontend API wrapper and backend dispatch. Do not report completion when a listed page/action lacks its skill operation or a skill operation lacks backend dispatch.

## Frontend Requirements

Build `frontend.js` as an ES module exporting `render(container, context)`.

The runtime supports multi-file frontend modules. Keep `frontend.js` as the entrypoint and import browser-loadable relative modules from `./frontend/*.js`. For non-trivial pages, split API calls, state, UI rendering, modal/toast logic, and styles into separate files instead of placing all frontend logic in the entry file.

When using multi-file frontend modules, `frontend.js` must be replaced with the real app entrypoint. Do not leave scaffold imports such as `createInitialRows` or `nextRowId` unless those exact exports still exist. After writing files, check every named import in `frontend.js` and every `frontend/*.js` module against the actual exports before reporting completion.

`frontend/styles.js` must export `injectStyles(root)`, and `frontend.js` must import it as `import { injectStyles } from './frontend/styles.js'`. Do not use alternate names such as `STYLES`, `styles`, `getStyles`, or default exports.

Style consistency is mandatory:

- Use the standard tokens from `references/style-guide.md`.
- Scope all selectors under an app-specific root class.
- Keep operational layouts compact and work-focused.
- Use neutral surfaces, blue primary actions, and semantic status colors.
- Use 6px to 8px border radius.
- Include matching styles for header, toolbar, table/list/cards, buttons, inputs, badges, modal, toast, loading, empty, and error states.
- Do not style generic global selectors without the app root scope.

The page must include:

- Initial data load from backend.
- Create and update form flows when the domain has editable records.
- List/table/card view appropriate to the domain.
- Search, filter, status transition, pagination, or another meaningful workflow when relevant.
- Loading, empty, success, and error states.
- Input validation before invoking backend.
- Escaping for user-provided text before writing HTML.
- Event listeners wired after `container.innerHTML` is assigned.
- Responsive layout that does not overlap on narrow screens.
- Modals, overlays, toasts, and other dynamically appended nodes must receive matching styles. If CSS is scoped as `.app-root .modal`, append those nodes under `.app-root`, not as unscoped siblings of the root container. Otherwise they will render inline instead of as popups.
- Controlled buttons must hide when `context.can(actionKey)` is false.
- Controlled buttons inside modals, drawers, detail panels, tab panes, nested tables, and child modules must receive a propagated `can` function and use the same action-key check before rendering.
- Uncontrolled buttons and read-only interactions must remain available without requiring `context.can(...)`. For example, a "管理" button that only opens a read-only nested list can stay visible even when the user has no create/update/delete permissions inside that nested list.
- Event delegation handlers for controlled write actions must re-check `can(actionKey)` before calling API wrappers.

Recommended frontend shape for normal CRUD/workflow apps:

```text
frontend.js
  create root, inject styles, create api/state, call load, bind events, dispose listeners
frontend/api.js
  list/create/update/delete/status action wrappers using invokeData
frontend/state.js
  normalize rows, filters, selected record, form defaults, pagination
frontend/ui.js
  render shell, toolbar, table/cards, empty state, detail region
frontend/modal.js
  form dialog, confirm dialog, toast helpers
frontend/styles.js
  export injectStyles(root); scoped CSS for all static and dynamic nodes
```

Use stable, boringly reliable controls for operational tools:

- Search input and filter select above the list.
- Table or dense list for records.
- Primary "新建" button.
- Row actions for edit, delete/archive, and status changes.
- Modal or side panel forms.
- Toast or inline alert for save results.

Do not build a marketing landing page, oversized hero, decorative dashboard, or prose-first summary unless the product doc explicitly asks for a read-only informational page.

Use this runtime invocation pattern:

```js
export function render(container, context) {
  const appId = context.app.id

  async function invoke(payload) {
    return context.invokeData(appId, payload)
  }
}
```

`context.invokeData(...)` returns the parsed backend JSON payload directly, for example `{ success, data, error }`. Do not read `result.response` after calling `invokeData`; `response` only exists on the lower-level `context.invoke(...)` result.

Do not expose "generated app", "registry", "binding", "manifest", or internal implementation language in the user-facing page.

## Backend Requirements

Build the backend as TinyGo/WASI-compatible Go:

- Include `//go:build wasip1` in each backend Go source file.
- Provide `main.go` with exported `alloc` and `handle(reqPtr, reqLen uint32) uint64`.
- Split normal apps into `backend/main.go`, `backend/platform.go`, `backend/models.go`, and one or more handler/validator files. Do not put all backend logic into one Go file unless the app is truly tiny.
- Build WASM with `GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../backend.wasm .` from the backend directory. The `-buildmode=c-shared` flag is required so `//go:wasmexport` functions are callable as a WASI reactor.
- Decode a request envelope such as `{ "action": "...", "data": {...} }`.
- Implement action handlers for all frontend operations.
- Include `platform.go` host imports for `data_list`, `data_get`, `data_create`, `data_update`, `data_delete`, `data_join_query`, `data_run_query`, `log`, `result_len`, `result_read`, and `result_store`.
- `platform.go` import signatures must match the host exactly:

```go
//go:wasmimport platform data_list
func hostDataList(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_create
func hostDataCreate(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_update
func hostDataUpdate(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_delete
func hostDataDelete(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_join_query
func hostDataJoinQuery(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_run_query
func hostDataRunQuery(reqPtr, reqLen uint32) uint64

//go:wasmimport platform result_len
func resultLen(handle uint64) uint32

//go:wasmimport platform result_read
func resultRead(handle uint64, outPtr uint32) uint32

//go:wasmimport platform result_store
func resultStore(reqPtr, reqLen uint32) uint64

//go:wasmimport platform log
func hostLog(level uint32, messagePtr uint32, messageLen uint32)
```

- `result_read` must return `uint32` (bytes written). Do not declare it as `void`.
- Return JSON through `storeResponse`, not stdout.
- Keep `backend/logger.go` and its `logDebug`, `logInfo`, `logWarn`, and `logError` helpers when creating or repairing an app. Those helpers call `hostLog`; do not replace them with `fmt.Println`, `println`, or direct stdout/stderr writes.
- Implement the applied technical document's Section 16 table at the listed validation exits, critical branches, data/capability operation boundaries, business failures, and successful outcomes. Generic runtime logs alone do not satisfy this requirement.
- The host automatically correlates every guest log with the outer `request_id`; do not accept, generate, include, or log a request ID in WASM source.
- The host automatically logs each platform `data_*` capability call with its logical model/query, aggregate result, duration, and safe error code. Add the Section 16 business markers around the operation; do not duplicate its low-level data-operation lines or log filters, IDs, records, or raw database errors.
- Log single-line `key=value` messages using stable action names. Use `logWarn` for expected validation/state rejections and `logError` for failed operations/error exits; use `logInfo` for accepted actions, external/data capability boundaries, and final outcomes. Reserve `logDebug` for diagnosis-relevant branch detail.
- Never log passwords, access tokens, cookies, authorization headers, API keys, secrets, raw request/response JSON, complete form payloads, or unredacted personal data. Sanitize and bound error text; prefer stable error codes and aggregate counts or approved non-sensitive identifiers.
- Use data capability wrappers:
  - `dataList(DataListRequest{Model: "book", ...})`
  - `dataCreate(DataWriteRequest{Model: "book", Record: ...})`
  - `dataUpdate(DataWriteRequest{Model: "book", ID: id, Patch: ...})`
  - `dataDelete("book", id)`
  - `dataJoinQuery(DataJoinRequest{From: "book", Joins: ..., Select: ...})`
  - `dataRunQuery(DataRunQueryRequest{Query: "manifestQueryName"})`
- Never concatenate SQL, never create table constants, and never pass SQL text to `dataRunQuery`.
- Add validation for required fields, enum/status values, pagination bounds, and missing IDs.
- Return a consistent shape, for example `{ success, data, rows, total, error }`.

Action parity is mandatory:

- Every action called by `frontend/api.js` must be handled in backend dispatch.
- Every backend action should have at least one frontend path that can call it, unless it is intentionally reserved and documented.
- Response field names must match what frontend state normalization reads.
- List actions should return deterministic ordering and enough fields for the visible list.
- Create/update actions should return the saved row so the frontend can refresh or patch state.

## Build And Registration Workflow

Follow this sequence when creating or repairing an app:

1. Read applied product and technical documents from paths in the prompt.
2. Implement per technical doc sections 8-16; derive entities/actions from Section 10, the per-action logging plan from Section 16, permission behavior from `references/permissions.md`, interaction behavior from Section 12, and style rules from Section 13 plus `references/style-guide.md`.
3. Use prompt `appId`, `appDir`, and `tablePrefix` — do not re-allocate identity unless prompt omits them.
4. Create or update the full app directory and all required source files under `appDir`.
5. Write `manifest.json` with UUID-only id plus structured `dataModels`, `relations`, and `queries` from the technical doc. Do not write DDL or SQL for new apps.
6. Write `frontend.js` and `frontend/*.js` per Section 11.
7. Write backend Go files with matching actions per Section 10 and the planned WASM log markers from Section 16.
8. Build `backend.wasm` from the backend source. Confirm `backend.wasm` is newer than every `backend/*.go` source file.
9. Verify import/export consistency, action parity, and Section 16 logging coverage.
10. Inspect the final frontend for placeholder-only content and missing controls.
11. Output the completion tag from the prompt as the last line.

Do not report completion before steps 4-8 are done. If local service restart is blocked, say what remains and which files were completed.

## Verification Checklist

Before final response, check:

- `manifest.json` exists and `id` matches the UUID folder name.
- `manifest.name` is the business display name, not the UUID.
- Manifest declares `dataModels` for every business model used by backend actions.
- Manifest declares `relations` for every multi-model read used by backend actions.
- Manifest declares `queries` for every `dataRunQuery` call.
- Backend contains no SQL strings (`SELECT`, `INSERT`, `UPDATE`, `DELETE`), no physical table constants, and no `db_query` / `db_exec` imports.
- `backend/platform.go` imports `platform log`, and `backend/logger.go` retains usable `logDebug`, `logInfo`, `logWarn`, and `logError` helpers.
- Every Section 10 action implements the corresponding Section 16 markers for validation rejection, critical business boundaries, failure, and final outcome. Logs are business diagnostics, not only generic invocation messages.
- Log statements contain no credentials, secret material, raw request/response payloads, or unredacted personal data; no WASM source manually includes a `request_id`.
- Every `dataRunQuery` call uses a manifest query name, not SQL text.
- `backend.wasm` exists and was rebuilt after source changes; its modification time must be later than all `backend/*.go` files.
- `frontend.js` exports `render` and uses `context.invokeData`.
- Every named frontend import has a matching export in the imported file.
- `frontend/styles.js` exports a named `injectStyles` function and `frontend.js` imports that exact name.
- `frontend.js` is the real business entrypoint, not the scaffold placeholder entrypoint.
- The frontend has real controls and not just explanatory text.
- The first screen shows the core working view and primary action.
- `frontend/styles.js` follows `references/style-guide.md` tokens and scoped selector rules.
- Create/edit/status/delete/search/filter flows required by the docs are wired end-to-end.
- Loading, empty, validation error, backend error, and success feedback states are implemented.
- Header, toolbar, table/list/card, button, input, badge, modal, toast, empty, loading, and error styles are present when those components are used.
- No unscoped global selectors leak into the host page.
- All expected action strings are implemented in both frontend and backend.
- Every `manifest.actions` controlled button is hidden when `context.can(actionKey)` is false, including buttons in nested modals, drawers, detail panels, tab panes, nested tables, and child modules.
- Read-only entry buttons that are not listed in `manifest.actions` remain visible by default, while controlled write buttons inside the opened view are permission-gated.
- Controlled write action event handlers re-check `can(actionKey)` before invoking backend API wrappers.
- Backend dispatch has no action missing from frontend API wrappers.
- Frontend API wrappers have no action missing from backend dispatch.
- `manifest.agentSkill` exists and has at least one read operation.
- Every primary list/work queue has an `effect: "read"` list tool whose flat fields cover its search, filters, sorting, and pagination inputs.
- Every visible detail flow has an `effect: "read"` detail tool with the required record identifier.
- Every Agent-exposed write action has exactly one matching skill operation; skill action names and fields match backend dispatch and frontend API wrappers.
- Frontend API wrappers should surface empty backend results as a diagnostic error, for example `后端无响应，请确认 backend.wasm 已重建且功能已重新匹配`.
- The app can be listed/loaded by the generated app runtime, or the blocking error is documented.

## Useful Local Context

- Backend repo: current working directory.
- Frontend repo: sibling `../ai-dandelion-web` when available.
- Generated apps root: use the `appDir` value from the prompt; otherwise use `generated_apps`.
- Runtime loader: `func-operation/internal/runtime/generatedapp/service.go`
- Runtime scaffold fallback: `func-operation/internal/runtime/generatedapp/scaffold.go`
- Data capability runtime: `func-operation/internal/dao/data_capability.go`
- Legacy SQL validation: `func-operation/internal/dao/generated_app.go` (do not use for new apps)
- Frontend render context type: `../ai-dandelion-web/src/types.ts`

Commands in this project should use the local `rtk` prefix, for example `go test ./...`, `npm run build`, or `env GOCACHE=/private/tmp/ai-dandelion-go-cache go test ./...`.

## Public Configuration Runtime

Treat confirmed `公共配置` decisions in the approved documents as an implementation contract:

- Declare every exact key in `manifest.json` as `configKeys`, for example `"configKeys": ["country"]`.
- Read option arrays through `await context.config.get('country')`; use `context.config.getMany([...])` when the same page needs multiple keys.
- Render `{ value, label }` options and persist `value`, never the display label.
- Do not hard-code a business option fallback. If the public configuration cannot load or is empty, keep the dependent control unavailable and show a recoverable error.
- Use static arrays or generated data APIs only when the approved document explicitly selected `静态配置` or `业务数据`.

## External API Runtime

For every approved external API action, declare its exact key in `manifest.json`, for example `"externalApis": [{"apiKey":"game.order.submit"}]`. Generated backend code may call the WASM host import `external_api_call` with `{ apiKey, query, body }`, then handle the returned `{ success, statusCode, data }` locally.

Never use browser `fetch`, hard-code a base URL, or pass Authorization, signature, or secret headers. The API client configuration owns the target and all fixed headers. Do not call an API that is not declared in the manifest.
