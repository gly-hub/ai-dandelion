---
name: technical-doc-builder
description: Build implementation-ready technical design documents for ai-dandelion func-operation features, including page layout, interaction flow, frontend modules, backend actions, manifest dataModels/relations/queries, states, validation, and WASM observability. Use when deriving implementation design from an applied product doc before code generation, when output must align with generated-app-builder contracts, or when writing to generated_apps documents with a completion tag.
---

# Technical Doc Builder

## Purpose

Turn an applied product document into an implementation-ready technical design. Make all interaction, data, action, and file contracts explicit so code generation can implement without inventing the product. Stop before writing code or generated app files.

## Authority Scope

This skill is only authorized to create or update the technical draft document specified by `draft 文件`.

Allowed:

- Read the applied product document from `documents/product/applied/product-doc.md`.
- Read generated-app-builder reference files such as `references/structure.md` and `references/style-guide.md` from that skill when generated-app-builder is injected.
- Write exactly one file: `generated_apps/<app-id>/documents/technical/draft/technical-doc.md`, or the exact `draft 文件` path from the prompt.

Forbidden:

- Do not modify product documents under `documents/product/`.
- Do not modify generated app code, frontend files, backend files, `manifest.json`, `backend.wasm`, or platform source code.
- Do not run build commands, regenerate WASM, edit config, edit proto files, or call backend APIs.
- Do not perform code repair even if the user asks in the technical conversation. Instead, describe the technical requirement and leave implementation to the generated-app/code generation session.
- Do not change applied technical docs directly; only the draft file may be written.

If a user request in the technical conversation asks for product scope changes, send it back to the product step. If it asks for code changes, capture the requirement in the technical contract only when appropriate, but do not edit code.

## Func-Operation Prompt Contract

The frontend sends a short prompt like:

```text
使用 `technical-doc-builder` 技能。
功能名称：...
产品文档（applied）：generated_apps/<app-id>/documents/product/applied/product-doc.md
draft 文件：generated_apps/<app-id>/documents/technical/draft/technical-doc.md
完成标签：<func-operation-document-ready function-id="<function-id>" doc-type="technical" />
待继续标签：<func-operation-continue function-id="<function-id>" conversation="technical" />
```

When you see this shape:

1. Read the applied product document from the given path first.
2. Read generated-app-builder `references/structure.md` and `references/style-guide.md` when available; align the technical contract with both.
3. Before writing the document, decide whether implementation-critical facts are missing. If missing facts would change data models, actions, layout, validation, or generated-app contracts, ask concise follow-up questions instead of writing the draft.
4. Draft the technical document strictly from the product doc and generated-app contracts; do not expand scope.
5. Write the full document to the exact `draft 文件` path only after the implementation contract is clear enough (overwrite if exists).
6. Reply with 1-3 sentence summary only; do not paste the full document in chat.
7. Put `完成标签` alone on the last line when successful.
8. Put `待继续标签` alone on the last line only when the work remains unfinished and a later AI turn must continue it because of a turn, time, or execution boundary.
9. When waiting for user clarification or reporting a blocker, do not emit either tag.
10. Never write either tag inside `draft 文件` or any document body. Tags are chat-only signals.

## Document Storage

```text
generated_apps/<app-id>/documents/technical/draft/technical-doc.md
generated_apps/<app-id>/documents/technical/applied/technical-doc.md
```

## Hard Rules

- Read applied product doc before drafting.
- Read generated-app-builder structure and style references before drafting when they exist.
- No final code, generated files, or migration scripts.
- Sections 2, 4, and 8-16 must be concrete enough that `generated-app-builder` can implement without redesigning layout, structure, data models, action names, frontend states, visual style, or observability.
- Overwrite draft with the latest complete document; never append partial content.
- Do not produce generic architecture prose. Every page, component, action, state, and table must map to the requested feature.
- Do not design a generated page that only renders document summaries, AI output, API notes, or implementation plans.
- If the repository has `doc/func-operation/interaction-design.md`, follow its layout and workflow conventions for usage/design mode, preview behavior, recovery actions, and user-facing terminology.
- Use the standard generated-app style guide when planning frontend layout. If available, refer to generated-app-builder `references/style-guide.md` for visual tokens, component sizes, scoped CSS, and responsive behavior.
- Use one stable action naming scheme across Section 4, Section 10, Section 11, and Section 12. Do not mix names such as `createBook`, `book_create`, and `create` for the same operation.
- Define a concrete WASM logging contract in Section 16 for every backend action. It must identify the diagnostic markers, severity, permitted fields, and sensitive-field redaction needed for incident investigation.
- Do not leave placeholders such as `<business>`, `<entity>`, `待生成`, `根据情况`, or `TODO` in the final technical document.
- Do not write unresolved questions, "待确认问题", assumptions needing confirmation, or implementation TBDs into the final technical document. Clarify first, then produce a complete implementation contract.

## Quality Bar

A good technical doc must let the next agent implement the app by following contracts rather than making product decisions. It must specify:

- Exact page layout and regions.
- Frontend component/module responsibilities.
- User actions, request payloads, response shapes, and state updates.
- Which actions are button-permission-controlled versus read-only defaults.
- Business data models, relations, and named query capabilities.
- Validation and error handling.
- Empty/loading/success/failure UI states.
- Standard visual style and CSS module responsibilities.
- Acceptance checks for the generated app.

## Generated-App Handoff Contract

The technical document is the direct handoff to `generated-app-builder`. It must include a complete mapping across these layers:

| Layer | Must Define | Used By |
| --- | --- | --- |
| Product flow | Page, action, state, recovery path | `frontend.js`, `frontend/ui.js` |
| Frontend action | Control, handler, API wrapper, state update | `frontend.js`, `frontend/api.js`, `frontend/state.js` |
| Backend action | Action string, request, response, validation, data capability intent, permission mode | `backend/main.go`, handler files |
| App Skill | Tool key, action, effect, flat fields, exposure and confirmation policy | `manifest.json`, Function Skill runtime |
| Data schema | Entity, dataModel, relation, query, indexes, enum values | `manifest.json`, `backend/models.go` |
| Style contract | Root class, namespace, components, responsive behavior | `frontend/styles.js` |
| Observability | Per-action log markers, levels, permitted fields, redaction | `backend/logger.go`, handlers, runtime execution logs |

If non-critical details are absent, choose a conservative default and state the chosen behavior directly in the relevant section. If any row cannot be filled without changing scope or implementation contracts, ask the user before writing the draft. Do not leave implementation-critical gaps in Sections 8-16.

## Required Structure

1. 模块拆分
2. 页面与组件设计
3. 数据模型
4. API 设计
5. 状态流转与校验规则
6. 异常与空状态处理
7. 实施步骤
8. 代码目录与文件清单（对接 `generated-app-builder`）
9. Manifest 与表结构约定
10. 后端 Action 清单
11. 前端模块拆分
12. 前端交互流程与状态合同
13. 样式规范与视觉一致性
14. 验收与自检清单
15. App Skill 契约
16. 可观测性与 WASM 日志

Prefer actionable detail over abstract advice.

## Required Output Template

Use this section shape in the generated technical document. The content can be concise, but each table must be complete enough for code generation.

```markdown
# <功能名称> 技术设计

## 1. 模块拆分
## 2. 页面与组件设计
## 3. 数据模型
## 4. API 设计
## 5. 状态流转与校验规则
## 6. 异常与空状态处理
## 7. 实施步骤
## 8. 代码目录与文件清单（对接 generated-app-builder）
## 9. Manifest 与表结构约定
## 10. 后端 Action 清单
## 11. 前端模块拆分
## 12. 前端交互流程与状态合同
## 13. 样式规范与视觉一致性
## 14. 验收与自检清单
## 15. App Skill 契约
## 16. 可观测性与 WASM 日志
```

Do not add a "待确认问题" section. If the document would need that section, stop and ask the user before writing the draft.

### Section 1: 模块拆分

Split by runtime responsibility:

- Frontend UI modules.
- Frontend API/state modules.
- Backend action handlers.
- Data model and validation helpers.

Do not write broad layered architecture unless each module maps to files in Section 8.

Recommended table:

| 模块 | 文件 | 职责 | 依赖 |
| --- | --- | --- | --- |

Every file listed here must also appear in Section 8.

### Section 2: 页面与组件设计

For every page or region, include:

- Layout: table/list/card/form/modal/detail/filter bar/action bar.
- Visible fields and field order.
- Primary and secondary buttons.
- Event behavior for each button.
- Responsive behavior for narrow screens.
- Loading, empty, error, and success states.

Use a concrete layout sketch:

```text
顶部操作栏：搜索 / 筛选 / 新建
主体：记录表格
右侧或弹窗：新建/编辑表单
底部：分页或统计
```

Required table:

| 区域/组件 | 展示内容 | 用户操作 | 触发 action | 状态反馈 |
| --- | --- | --- | --- | --- |

Rules:

- The first screen must show the main working view and primary action.
- List each modal or detail panel as its own component.
- Name buttons exactly as they should appear in the UI.
- Every `触发 action` must exist in Section 4 and Section 10.

### Section 3: 数据模型

Define logical entities before backend actions. Include field purpose, validation, and UI usage.

Example:

| 实体 | 字段 | UI 用途 | 校验 |
| --- | --- | --- | --- |
| Book | title | 列表标题、表单必填项 | 必填，1-120 字 |

Required table:

| 实体 | 字段 | 类型语义 | UI 用途 | 校验 | 默认值 |
| --- | --- | --- | --- | --- | --- |

Rules:

- Use business names here, not physical table names.
- Include enum/status values and their display labels.
- Include whether fields appear in list, detail, form, filters, or badges.
- Keep fields compatible with manifest `dataModels` in Section 9.

### Section 4: API 设计

Every API action must include:

- Action name string.
- Request shape.
- Response shape.
- Frontend caller.
- Backend handler.
- State update after success.
- Error handling shown to the user.

Use one envelope consistently:

```json
{ "action": "create", "data": { "...": "..." } }
```

Backend responses should use:

```json
{ "success": true, "data": {}, "error": "" }
```

or, for lists:

```json
{ "success": true, "rows": [], "total": 0, "error": "" }
```

Required table:

| Action | 触发场景 | Request `data` | Response | 成功后前端状态 | 失败展示 | 权限模式 |
| --- | --- | --- | --- | --- | --- | --- |

`权限模式` must be one of:

- `button_controlled`: must be written into `manifest.actions`, requires `context.can(action)` for button display, and requires backend button permission validation.
- `read_default`: must not be written into `manifest.actions`; UI controls remain visible by default and backend falls back to menu-level access only.

Action naming rules:

- Use lower snake/camel-free simple names unless a domain prefix is needed: `list`, `detail`, `create`, `update`, `delete`, `change_status`.
- For multi-entity apps, prefix by entity: `student_list`, `class_list`, `department_list`.

Permission rules:

- Any action that changes business state must be `button_controlled`.
- Read actions such as `*_list`, `*_detail`, `*_query`, `*_search`, `*_page`, `*_stats` must be `read_default` unless the product doc explicitly says otherwise.
- `frontend/api.js`, backend dispatch action names, and `manifest.actions` must use exactly the same action keys.
- `manifest.actions` must be documented and generated as a JSON string array only, for example `["class_create", "class_update"]`; do not place label/mode objects into `manifest.actions`.
- Use exactly the same string in frontend API wrappers and backend dispatch.
- Do not use Chinese action strings.

Response rules:

- List actions return `rows` and `total`.
- Detail/create/update/status actions return `data`.
- Delete/archive actions return `{ "success": true }` or updated `data`.
- Failures return `success: false` and a user-readable `error`.

### Section 5: 状态流转与校验规则

Include:

- UI state variables.
- Business status values and allowed transitions.
- Field validation rules.
- Button disabled rules.
- Backend validation rules.

Required tables:

| 状态变量 | 类型 | 初始值 | 变化时机 | 影响 UI |
| --- | --- | --- | --- | --- |

| 业务状态 | 显示文案 | 可执行操作 | 下一状态 | 禁止条件 |
| --- | --- | --- | --- | --- |

| 字段 | 前端校验 | 后端校验 | 错误文案 |
| --- | --- | --- | --- |

### Section 6: 异常与空状态处理

Required table:

| 场景 | 触发条件 | UI 展示 | 恢复动作 | 对应 action |
| --- | --- | --- | --- | --- |

Must cover loading, no records, no search results, validation failure, backend error, save failure, and delete/status failure when relevant.

### Section 7: 实施步骤

Write implementation steps in generated-app-builder order:

1. Write or update `manifest.json`.
2. Implement backend models and logical model/relation/query constants.
3. Implement backend action handlers and validators.
4. Implement the Section 16 per-action WASM log markers with the shared logger helpers.
5. Implement frontend API wrappers.
6. Implement frontend state and normalization.
7. Implement UI render functions and event bindings.
8. Implement modal/toast helpers.
9. Implement scoped styles from the style guide.
10. Build `backend.wasm`.
11. Run handoff self-checks.

### Section 8: 代码目录与文件清单

List exact files under `generated_apps/<uuid>/`. Must match generated-app-builder `references/structure.md`:

```text
generated_apps/<uuid>/
  manifest.json
  frontend.js
  frontend/api.js, state.js, ui.js, modal.js, styles.js
  backend/main.go, platform.go, models.go, <business>_handlers.go, validators.go
  backend.wasm
```

Normal CRUD/workflow apps must not collapse into one frontend file plus one backend file.

Required table:

| 文件 | 必须生成 | 主要内容 |
| --- | --- | --- |

Use `是/否` for `必须生成`. Normal CRUD/workflow apps should mark frontend API/state/UI/modal/styles and backend handler/validator files as `是`.

### Section 9: Manifest 数据能力约定

- `manifest.id` equals app UUID folder name.
- Do not design physical table names. The platform generates tables from `dataModels`.
- Use `dataModels` for private business entities.
- Use `relations` for multi-model reads.
- Use `queries` for fixed named read capabilities that backend handlers call through `dataRunQuery`.
- Do not include raw SQL, DDL, `tables`, `db_query`, or `db_exec` for new apps.

Required table:

| 模型 | manifest `dataModels[].name` | 字段 | 校验 | 索引建议 |
| --- | --- | --- | --- | --- |

Field rules:

- Use logical field names. Platform-managed fields `id`, `uuid`, `created_at`, and `updated_at` are implicit.
- Supported field types: `string`, `text`, `int`, `float`, `bool`, `enum`, `datetime`, `id`.
- Include `required`, `maxLength`, `min`, `max`, and enum `values` when relevant.
- Add indexes only for fields used by filters or sorting.

Relations table when needed:

| Relation | From | To | Type | Usage |
| --- | --- | --- | --- | --- |

Queries table when needed:

| Query name | From | Joins | Select | Filters | Sorting | Limit | Called by actions |
| --- | --- | --- | --- | --- | --- | --- | --- |

### Section 10: 后端 Action 清单

Every action: name, request fields, response shape, data capability intent, validation. Map each to `backend/<business>_handlers.go`. Typical: `list`, `detail`, `create`, `update`, `delete`, `change_status`.

Required table:

| Action | Handler 文件 | Handler 函数 | Request | Response | Data capability 调用 | 校验 | 日志契约 |
| --- | --- | --- | --- | --- | --- | --- | --- |

Rules:

- Every action in Section 4 must appear here.
- Every handler here must be called by backend dispatch.
- Data capability intent must name logical models, relations, or query names. Do not use SQL text or physical table names.
- Validation must include required IDs and enum/status checks.
- `日志契约` must reference the corresponding complete row in Section 16; it cannot be `无` for a business action.

### Section 11: 前端模块拆分

Per module: responsibility and key exports.

- `frontend.js`: `render(container, context)`, optional `dispose(container)`
- `frontend/api.js`: `context.invokeData(context.app.id, payload)` + action names from Section 10
- `frontend/state.js`, `frontend/ui.js`, `frontend/modal.js`, `frontend/styles.js`

Every named import in `frontend.js` must have a matching export in the target module.

Required table:

| 文件 | 关键导出 | 依赖 | 被谁调用 |
| --- | --- | --- | --- |

Rules:

- `frontend/api.js` must list one wrapper per backend action.
- `frontend/state.js` must list state shape and normalization helpers.
- `frontend/ui.js` must list render functions for shell, toolbar, content, empty/error/loading states.
- `frontend/modal.js` must list modal, confirm, and toast helpers when used.
- `frontend/styles.js` must list a named `injectStyles(root)` export. Do not specify alternate names.

### Section 12: 前端交互流程与状态合同

Define the exact frontend behavior for each user operation:

| 操作 | 触发控件 | 前端状态变化 | 调用 action | 成功反馈 | 失败反馈 |
| --- | --- | --- | --- | --- | --- |

Include at least:

- Initial load.
- Create flow.
- Edit flow.
- Delete/archive flow when relevant.
- Search/filter flow.
- Status transition flow when relevant.
- Empty and error recovery flow.

State variables must be named clearly enough for implementation, for example:

- `rows`
- `filters`
- `selectedId`
- `modalMode`
- `form`
- `loading`
- `saving`
- `error`
- `toast`

Also include a frontend action parity table:

| UI 操作 | DOM 控件/事件 | API wrapper | Backend action | 成功后刷新 |
| --- | --- | --- | --- | --- |

Every backend action in Section 10 should be reachable from this table unless intentionally backend-only.

### Section 13: 样式规范与视觉一致性

Specify how the generated app applies the standard style guide:

- App-specific root class and CSS namespace prefix.
- Header layout and primary action placement.
- Toolbar controls: search, filters, reset action.
- Main content pattern: table, dense list, card list, detail region, or status board.
- Button variants: primary, secondary, danger, small.
- Form, modal, toast, badge, loading, empty, and error styles.
- Mobile behavior below 720px.

Use `frontend/styles.js` for the style module. All dynamic nodes from `modal.js` and toast helpers must have matching scoped styles.

Do not introduce a custom palette unless the user explicitly requests a domain-specific theme. Default to neutral gray surfaces, blue primary actions, and semantic green/amber/red/gray badges.

Required table:

| UI 元素 | class 命名 | 样式要求 | 状态 |
| --- | --- | --- | --- |

Must include:

- Root.
- Header.
- Toolbar.
- Primary/secondary/danger buttons.
- Inputs/selects.
- Table/list/card.
- Badges.
- Modal.
- Toast/alert.
- Loading/empty/error blocks.

Rules:

- Use one namespace prefix for all classes.
- Do not define global `button`, `input`, `table`, or `.modal` styles without the app root scope.
- Dynamic nodes from `modal.js` and toast helpers must be appended under the root or have scoped selectors that still apply.

### Section 14: 验收与自检清单

List concrete checks `generated-app-builder` must satisfy, including:

- Frontend renders real controls, not a text summary.
- Initial load invokes backend list action.
- Create/edit forms validate required fields before backend call.
- Backend implements every frontend action string.
- Manifest dataModels/relations/queries match backend data capability calls.
- Empty, loading, success, and error states are visible.
- UI remains usable on mobile width.
- Generated CSS follows the shared style guide and does not leak global selectors.
- Modal, toast, and dropdown classes all have matching styles.

Add a final handoff checklist table:

| 检查项 | 通过标准 |
| --- | --- |

Must include:

- Section 4 actions match Section 10 backend actions.
- Section 10 actions match Section 11 `frontend/api.js` wrappers.
- Section 9 model, relation, and query names match backend constants.
- Section 13 class names match components in Section 11.
- Section 15 App Skill operations cover every primary list, visible detail flow, and Agent-exposed write action.
- Section 15 field definitions match the backend request fields; read actions are absent from `manifest.actions`.
- Every Section 10 action has a complete Section 16 logging row, and every listed marker has a concrete handler location.
- Log messages contain no credentials, request/response bodies, or unredacted personal data.
- Every required product action has UI, API, backend, validation, success, and failure handling.
- No user-facing text exposes generated-app internals.
- No placeholder content remains.

### Section 15: App Skill 契约

Every generated function has one App Skill at the same business level as the function. Define the complete Agent-operable capability set before code generation; do not defer this to `generated-app-builder` or infer it from write permissions.

Required table:

| Tool key | Backend action | Effect | Agent 场景 | Flat fields | 权限与确认 | 返回形态 |
| --- | --- | --- | --- | --- | --- | --- |

Rules:

- Include one `read` list operation for every main list, work queue, dashboard, or searchable collection. Its fields must explicitly include every supported search, filter, sort, and pagination input.
- Include one `read` detail operation for every visible detail page, drawer, or modal. The stable record identifier is required.
- Include every Agent-exposed create/update/delete/execute action exactly once. Actions that are strictly internal must be marked non-exposed with the reason in this section.
- `create` may use `autoExecute: true` only when the product explicitly permits automatic creation. `update`, `delete`, and `execute` always require confirmation. Read operations do not require confirmation.
- Fields must be flat, typed as `string`, `number`, `integer`, `boolean`, or `enum`; document enum values and required fields. Nested objects and files are out of scope.
- The tool name is `<toolPrefix>__<tool key>`. Define a stable lowercase snake_case `toolPrefix` for the function.
- Read actions use `read_default` and never enter `manifest.actions`; non-read actions are `button_controlled` and must enter `manifest.actions`.
- The Function Skill runtime wraps flat tool arguments into the generated app request `{"action":"<action>","data":{...toolInput}}`; frontend and backend continue to use the same action name.

### Section 16: 可观测性与 WASM 日志

WASM execution logs are a troubleshooting contract, not user-facing page content. They are captured by the host runtime and queried from Function Management. The runtime automatically attaches the outer `request_id`, function/app/version/export metadata, timestamp, and invocation context. Guest code must never accept, construct, or print a `request_id` itself.

For every Section 10 action, define the following complete table. One action may have several rows when its critical branches differ; do not replace business markers with only a generic `handle action=...` entry.

| Action | Marker location | Level | Required message pattern | Permitted diagnostic fields | Redaction / exclusion | Expected branch |
| --- | --- | --- | --- | --- | --- | --- |

Required markers for every action:

- `INFO` on action acceptance or validation start: `action=<action> phase=validate`.
- `WARN` on validation rejection or prohibited state transition: `action=<action> phase=validate code=<safe_code>`.
- `DEBUG` or `INFO` for each critical business branch where it affects the result: `action=<action> branch=<stable_name>`.
- `INFO` immediately before a data capability, external API, or other meaningful business operation: `action=<action> operation=<stable_operation> model=<logical_model>`.
- `ERROR` on a failed business operation or error exit: `action=<action> phase=execute code=<safe_code> error=<sanitized_message>`.
- `INFO` for a successful final business result: `action=<action> result=success` with a safe result count, rows affected, or stable state when useful.

Design log messages as single-line `key=value` console messages. Use the exact lower-snake-case action key from Sections 4 and 10. Record only values that aid diagnosis: logical model/query/capability name, stable branch name, safe error code, aggregate count, rows affected, or an explicitly approved non-sensitive business identifier. Do not log passwords, access tokens, API keys, cookies, authorization headers, secrets, raw request/response JSON, full form values, or unredacted personal data. If an identifier can be sensitive, document the masking rule or omit it. Error text must be sanitized and bounded; use a stable error code where the underlying message could expose data.

The table must state where each marker belongs, such as `book_handlers.go: createBook before dataCreate` or `validators.go: validateBookPayload rejection`. Generated code uses the platform helpers `logDebug`, `logInfo`, `logWarn`, and `logError`; it must not use stdout/stderr as a response channel or try to manage correlation IDs. Keep the marker count focused: record state transitions and business boundaries, not every local variable or loop iteration.

The host runtime also records every platform data capability call (`data_list`, `data_get`, `data_create`, `data_update`, `data_delete`, `data_join_query`, and `data_run_query`) with the logical model or query name, safe result count or affected rows, duration, and safe error code. It intentionally excludes filters, IDs, record fields, and raw database errors. Section 16 should therefore define business-intent and branch markers that complement these automatic data-operation logs instead of duplicating them.

## Workflow Position

```
product-doc-builder → technical-doc (this skill) → generated-app-builder
```

Sections 8-16 are the handoff contract to code generation. If non-critical details are ambiguous, resolve them with conservative defaults in the relevant section. If ambiguity changes layout, data models, validation, action behavior, or safe observability, ask the user before writing the draft.

## Blocked Work

When the product doc cannot be read, the file cannot be written, or implementation-critical facts are unclear, explain the blocker or ask 1-3 concise follow-up questions in chat. Do not write or overwrite the draft file, and do not emit either tag.

## Public Configuration Contract

Preserve every confirmed option-source decision from the applied product document. For a field marked `公共配置`, specify the exact `config_key`, load it with `context.config.get(configKey)` or batched `context.config.getMany([...])`, render `{ value, label }` options, and persist only `value` in business records.

Do not design a generated data model, table, CRUD action, or hard-coded fallback option list for a public configuration. When the configuration is missing, empty, or unavailable, the dependent control must block submission and show a recoverable error state.

If a selectable field has no confirmed source decision, stop for clarification rather than choosing a source during technical design.

## External API Contract

For a confirmed external API action, document its exact `api_key`, request body/query mapping, call timing, idempotency behavior, and response branches. The generated frontend must never call the external service. Do not specify an arbitrary URL, Authorization header, secret, or signature implementation; the platform resolves those from the API client configuration.
