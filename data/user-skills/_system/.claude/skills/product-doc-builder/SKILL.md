---
name: product-doc-builder
description: Build interaction-ready product requirement documents for ai-dandelion func-operation features. Use when the func-operation editor triggers product doc generation, when Codex needs to turn a feature idea into a user-facing feature workflow before technical design, or when output must be written to generated_apps documents and signaled with a completion tag.
---

# Product Doc Builder

## Purpose

Create a product-facing requirement document before technical design or code generation. Define the user experience clearly enough that the technical doc and generated app can implement a real interactive business page. Do not write code in this stage.

## Authority Scope

This skill is only authorized to create or update the product draft document specified by `draft 文件`.

Allowed:

- Read the feature name, feature description, and user-provided product context.
- Read product-facing reference docs such as `doc/func-operation/interaction-design.md`.
- Write exactly one file: `generated_apps/<app-id>/documents/product/draft/product-doc.md`, or the exact `draft 文件` path from the prompt.

Forbidden:

- Do not modify technical documents under `documents/technical/`.
- Do not modify generated app code, frontend files, backend files, `manifest.json`, `backend.wasm`, or platform source code.
- Do not run build commands, regenerate WASM, edit config, edit proto files, or call backend APIs.
- Do not perform code repair even if the user asks in the product conversation. Instead, state that code changes belong to the generated-app/code generation session.
- Do not change applied product docs directly; only the draft file may be written.

If a user request in the product conversation asks for technical design or code changes, do not perform it. Summarize the requested change as product intent when appropriate, or ask the user to switch to the correct technical/code generation step.

## Func-Operation Prompt Contract

The frontend sends a short prompt like:

```text
使用 `product-doc-builder` 技能。
功能名称：...
功能描述：...
draft 文件：generated_apps/<app-id>/documents/product/draft/product-doc.md
完成标签：<func-operation-document-ready function-id="<function-id>" doc-type="product" />
待继续标签：<func-operation-continue function-id="<function-id>" conversation="product" />
```

When you see this shape:

1. Read `功能名称` and `功能描述` as the only business input unless the user adds more in chat.
2. Before writing the document, decide whether key product facts are missing. If they are missing and a reasonable assumption would change scope, ask concise follow-up questions instead of writing the draft.
3. Write the full document to the exact `draft 文件` path only after the scope is clear enough (overwrite if exists).
4. Reply with 1-3 sentence summary only; do not paste the full document in chat.
5. Put `完成标签` alone on the last line when successful.
6. Put `待继续标签` alone on the last line only when the work remains unfinished and a later AI turn must continue it because of a turn, time, or execution boundary.
7. When waiting for user clarification or reporting a blocker, do not emit either tag.
8. Never write either tag inside `draft 文件` or any document body. Tags are chat-only signals.

## Document Storage

Documents live alongside generated app code:

```text
generated_apps/<app-id>/documents/product/draft/product-doc.md   ← write here
generated_apps/<app-id>/documents/product/applied/product-doc.md ← applied by user later
```

Same pattern for technical docs under `documents/technical/`.

## Hard Rules

- No code, SQL, API implementations, directory plans, or manifest design.
- Stay within stated scope; ask concise follow-up questions before document generation if key facts are missing.
- Do not write unresolved questions, TODOs, assumptions needing confirmation, or "待确认问题" into the final product document. Clarify first, then produce a complete document.
- Overwrite draft with the latest complete document; never append partial content.
- Keep page names, business objects, and core flows stable — `technical-doc-builder` and `generated-app-builder` will reuse them verbatim.
- Design for a usable feature, not a narrative summary page. The final generated app must let users create, view, update, filter, transition, or otherwise operate business data.
- Use user-facing language. Do not expose `generated app`, `manifest`, `binding`, `draft`, `applied`, `session`, `wasm`, or internal registry concepts in the product flow.
- If the repository has `doc/func-operation/interaction-design.md`, follow its layout and workflow conventions: usage mode, design mode, five-step flow, preview states, and recovery actions.
- Use the platform's standard generated-app visual style. Do not request custom brand themes, marketing-style layouts, or one-off visual systems unless the user explicitly asks for them.

## Quality Bar

A good product doc must answer:

- Who uses this feature and what job they complete.
- What the first screen looks like and what primary action is obvious.
- Which business records or objects the user manipulates.
- Which actions are available, including create/edit/delete/status changes when relevant.
- Which actions require button-level permission control, and which actions are read-only defaults that should remain visible without separate button permission.
- What users see in loading, empty, success, error, and no-permission states.
- How the feature is useful after generation, not just what the AI planned.
- Which standard UI pattern fits the feature: table workspace, card list, detail panel, modal form, filter bar, or status board.

## Required Structure

1. 业务目标
2. 目标用户与使用场景
3. 核心流程
4. 核心页面
5. 关键字段或业务对象
6. 关键操作与按钮
7. 页面状态与异常恢复
8. 标准样式与布局约束
9. 范围边界与不做内容
10. 验收标准

Use short headings and concrete bullet points. A PM, designer, and engineer should be able to align on scope and interaction from this alone.

Do not add a "待确认问题" section. If the document would need that section, stop and ask the user before writing the draft.

### Section 3: 核心流程

Write the user journey as concrete ordered steps. Include the happy path and at least two recovery paths.

Prefer:

```text
打开功能 -> 查看列表 -> 点击新建 -> 填写表单 -> 保存 -> 列表出现新记录 -> 可继续编辑状态
```

Avoid vague flows such as “用户管理数据” or “系统展示分析结果”.

### Section 4: 核心页面

For each page or major region, define:

- Layout: list/table/cards/form/modal/detail/preview.
- Primary action.
- Secondary actions.
- Required visible fields.
- Empty state.
- Error state.

Normal operational tools should feel like usable workspaces: clear lists, filters, forms, detail panels, and action feedback. Do not propose a page that only displays generated text, API notes, or design summaries.

### Section 5: 关键字段或业务对象

Define business objects with user-facing fields and field purpose. Do not design database column types here, but be concrete enough for technical design.

Example:

| 对象 | 字段 | 说明 |
| --- | --- | --- |
| 图书 | 书名、作者、分类、库存、状态 | 用于图书入库、检索和借阅状态判断 |

### Section 6: 关键操作与按钮

List actions the generated app must support. Each action should include trigger, required inputs, success feedback, and failure feedback.

For every button-like operation, also classify permission behavior:

- `受控按钮`:
  Actions that change business state, such as create, edit, delete, submit, approve, publish, borrow/return, assign, enable/disable.
- `非受控操作`:
  Read-only behavior such as list, detail, search, filter, pagination, refresh, and stats viewing.

The product document must make this distinction explicit so later steps can decide:

- which actions must become `manifest.actions`
- which controls should stay visible by default without extra button permission

Typical actions:

- 新建
- 编辑
- 删除或归档
- 搜索
- 筛选
- 状态流转
- 查看详情
- 批量处理, only when clearly needed

For each listed action, include one extra field in prose or table form:

- `权限要求`: `受控` or `非受控`

Rules:

- If a control exists in the UI and changes business state, mark it `受控`.
- If an action is not marked `受控`, later steps must not place it into `manifest.actions`.

### Section 7: 页面状态与异常恢复

Cover at least:

- Loading state.
- Empty state.
- Validation error.
- Backend or save failure.
- No result after search/filter.
- Recoverable next action.

### Section 8: 标准样式与布局约束

State that the feature should use the standard func-operation generated-app style:

- Work-focused internal tool layout.
- Header, toolbar, content area, modal/detail area, and toast/alert feedback.
- Neutral gray surfaces and blue primary action.
- Compact table/list/card layouts.
- Loading, empty, error, and validation states.

Do not define exact CSS values here; `technical-doc-builder` and `generated-app-builder` will apply the shared style guide.

### Section 10: 验收标准

Write acceptance criteria as observable user outcomes. Include at least five criteria.

Example:

- 用户可以新建一条记录，并在列表中立即看到它。
- 用户提交缺少必填字段的表单时，页面显示具体字段错误。
- 用户搜索不存在的关键词时，页面显示空结果和清除筛选入口。

## Workflow Position

```
product-doc (this skill) → technical-doc-builder → generated-app-builder
```

Product doc defines *what users can do*; later steps define *how*. Do not pre-empt technical sections (tables, APIs, file trees), but do define concrete pages, operations, and states.

## Blocked Work

When the file cannot be written or product scope is unclear, explain the blocker or ask 1-3 concise follow-up questions in chat. Do not write or overwrite the draft file, and do not emit either tag.

## Public Configuration Decisions

For every selectable form field or filter, identify exactly one source in the product document: `公共配置`, `静态配置`, or `业务数据`.

- Use `公共配置` only for a shared, independently maintained option set. Record the exact catalogue `config_key`; never infer a key from its name.
- Use `静态配置` only for options unique to this function and explicitly stable.
- Use `业务数据` when options need their own records, CRUD, search, permissions, history, or relations.
- For `公共配置`, specify the option shape `[{ value, label }]`, and state that business records persist `value` while the interface displays `label`.
- If the supplied public configuration catalogue does not make the key unambiguous, ask for a selection or creation decision instead of silently hard-coding options.

## External API Decisions

For every action that communicates with an external system, record an existing `api_key`, its client, the business trigger, request-field mapping, and the success/failure outcome. Do not infer an API from a domain name or invent a URL. If no API is configured, require the interface administrator to create one before generation.
