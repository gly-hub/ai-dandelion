# Generated App Permission Contract

This reference defines how generated apps must implement menu access, button permissions, frontend visibility, and backend authorization. Read it before writing `manifest.json`, frontend action controls, frontend API wrappers, or backend dispatch code.

## Permission Model

Generated apps use two permission layers:

- Function menu permission: controls whether the user can open a published generated function.
- Button action permission: controls state-changing operations inside that function, such as create, update, delete, archive, submit, approve, assign, save, or status transitions.

Read-only operations are not button permissions. List, detail, query, search, filter, pagination, refresh, stats, and buttons that only open a read-only list/detail/management view must remain available by default unless the technical document explicitly declares that exact entry action as button-controlled.

## Manifest Actions

`manifest.actions` is the source of truth for button-controlled action keys.

Rules:

- Include every state-changing frontend/backend action that is triggered by a button or menu item.
- Exclude read-only actions such as list, detail, query, search, filter, page, refresh, and stats.
- Exclude read-only entry buttons that only open a nested list, modal, drawer, tab, or detail panel.
- Use a JSON string array only, for example `['class_create', 'class_update']` in conceptual examples and valid JSON in `manifest.json`.
- Do not write action objects with labels or modes in `manifest.actions`.

Example:

```json
{
  "actions": [
    "class_create",
    "class_update",
    "class_archive",
    "teacher_add",
    "teacher_update",
    "teacher_remove"
  ]
}
```

In this example, `teacher_list` is read-only and must not be in `manifest.actions`. A `manage_teachers` button that only opens the teacher list/modal also stays outside `manifest.actions` unless opening it itself changes state.

## Backend Authorization Flow

Generated app backend code still implements all actions, including read-only actions. Authorization is enforced by the host runtime before the WASM handler runs.

Runtime behavior:

1. Frontend calls `context.invokeData(context.app.id, { action, data })`.
2. Host resolves the generated function for the app id.
3. Host reads the app manifest actions.
4. If `action` is in `manifest.actions`, the host checks the matching button permission.
5. If `action` is not in `manifest.actions`, the host checks only function menu access.
6. If authorization fails, host returns `FORBIDDEN` and the WASM handler is not treated as the authorization boundary.

Backend implementation rules:

- Keep backend dispatch action names identical to frontend API wrapper action names.
- Do not add backend-only write actions that are absent from `manifest.actions`.
- Do not rely on frontend hiding as the only protection; host authorization is the security boundary for declared actions.
- Return normal business errors for validation failures, not permission decisions.

## Frontend Context

The host injects permission helpers through `GeneratedAppRenderContext`:

```js
context.permissions = {
  controlledActions: ['class_create', 'teacher_add'],
  actions: ['class_create']
}
context.can('class_create')
context.isControlled('class_create')
```

`context.can(actionKey)` means:

- If `actionKey` is not controlled by `manifest.actions`, return true.
- If `actionKey` is controlled and granted to the current user, return true.
- If `actionKey` is controlled but not granted, return false.

Generated frontend code should wrap this as a local function and pass it through child renderers:

```js
const can = typeof context.can === 'function'
  ? function (action) { return context.can(action) }
  : function () { return true }
```

## Frontend Rendering Rules

Controlled buttons must be hidden when `can(actionKey)` is false.

Apply this in every UI layer:

- Main toolbar buttons.
- Row action buttons.
- Detail panel buttons.
- Modal footer buttons.
- Drawer buttons.
- Tab pane actions.
- Nested table actions.
- Child module buttons.
- Dropdown or context menu items.

Read-only entry controls remain visible by default when they are not listed in `manifest.actions`.

Example: a class list row has a `教师管理` button that opens a teacher modal. The button itself only opens a read-only teacher list, so it should stay visible. Inside the modal:

- `添加教师` renders only when `can('teacher_add')`.
- `编辑` renders only when `can('teacher_update')`.
- `移除` renders only when `can('teacher_remove')`.

## Frontend Event Handler Rules

Rendering checks are not enough. Event handlers for controlled write actions must re-check permission before calling API wrappers.

Recommended pattern:

```js
if (action === 'teacher-add') {
  if (!can('teacher_add')) return
  openTeacherForm()
  return
}

if (action === 'teacher-save') {
  if (formMode === 'add' && !can('teacher_add')) return
  if (formMode === 'edit' && !can('teacher_update')) return
  saveTeacher()
  return
}

if (action === 'teacher-remove') {
  if (!can('teacher_remove')) return
  removeTeacher(id)
  return
}
```

This prevents manually triggered DOM events from invoking hidden write operations.

## Nested UI Propagation

When a nested renderer owns controlled buttons, pass `can` into it explicitly.

Correct:

```js
openTeacherManagerModal(root, {
  api,
  can,
})
```

Inside the modal:

```js
const can = opts.can || function () { return true }
const addButton = can('teacher_add')
  ? '<button data-action="teacher-add">添加教师</button>'
  : ''
```

Incorrect:

```js
const canManageTeacher = true
// Permission checks removed - all teacher operations enabled.
```

Also incorrect: hiding the outer read-only entry because the user lacks inner write permissions.

```js
// Wrong when this button only opens a read-only nested list.
if (!can('teacher_add')) return ''
return '<button data-action="manage-teachers">教师管理</button>'
```

## Action Parity Checklist

Before completion, verify:

- Every state-changing frontend API wrapper action is listed in `manifest.actions`.
- No read-only API wrapper action is listed in `manifest.actions`.
- Every action in `manifest.actions` is handled by backend dispatch.
- Every backend write action has at least one frontend controlled path.
- Every controlled path renders only when `can(actionKey)` is true.
- Every controlled event handler re-checks `can(actionKey)` before invoking backend.
- Nested modals, drawers, details, tabs, tables, and child modules receive `can` when they render controlled buttons.
- Read-only entry buttons not listed in `manifest.actions` stay visible by default.
- There is no code comment or implementation that removes permission checks for controlled actions.
