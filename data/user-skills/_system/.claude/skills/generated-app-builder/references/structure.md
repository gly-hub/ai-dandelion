# Generated App Directory Structure

Use this structure for each `func-operation` generated app. The app directory name is the app UUID only. Read `style-guide.md` before writing `frontend/styles.js`.

```text
generated_apps/<uuid>/
  manifest.json
  frontend.js
  frontend/
    api.js
    state.js
    ui.js
    modal.js
    styles.js
  backend/
    main.go
    platform.go
    models.go
    <business>_handlers.go
    validators.go
  backend.wasm
```

Small apps may omit unused optional files, but do not collapse a normal CRUD or workflow app into one frontend file plus one backend file.

## Manifest

`manifest.json` is the runtime contract.

- `id`: lowercase UUID, exactly matching the folder name.
- `name`: user-facing function name.
- `description`: user-facing capability summary.
- `export`: usually `handle`.
- `frontendFile`: `frontend.js`.
- `backendSource`: `backend`.
- `backendModule`: `backend.wasm`.
- `dataModels`: logical data models used by platform data capability APIs.
- `relations`: declared model relationships used by `data_join_query`.
- `queries`: predeclared named queries used by `data_run_query`.
- `tablePrefix`: legacy metadata from the platform. Do not use it for new app data access.

Do not create or reference physical table names in new apps. The platform generates real tables from `dataModels`.

Do not create placeholder data models before the real schema is known. Do not derive model names from the generated app UUID.

## Backend Files

Backend source is a Go package built from `generated_apps/<uuid>/backend/`.

- `main.go`: WASI entrypoint, request buffer, exported `alloc`, exported `handle`, request decoding, top-level dispatch call.
- `platform.go`: host imports and helpers for `data_list`, `data_get`, `data_create`, `data_update`, `data_delete`, `data_join_query`, `data_run_query`, `result_len`, `result_read`, and `result_store`.
- `models.go`: request/response envelopes, row structs, form payload structs, logical model names, relation names, query names, and statuses. Do not define physical table constants.
- `<business>_handlers.go`: action handlers such as list/create/update/delete/status transition.
- `validators.go`: optional validation helpers for required fields, enum values, pagination bounds, and IDs.

Each Go source file must include:

```go
//go:build wasip1

package main
```

Build from the backend directory:

```bash
cd generated_apps/<uuid>/backend
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../backend.wasm .
```

After building, check that `../backend.wasm` is newer than every `*.go` file in `backend/`. If the WASM is older than the source, the runtime will execute stale backend code even though the source files look correct.

Use `-buildmode=c-shared` for generated app backends. On `wasip1`, this creates a WASI reactor/library whose `//go:wasmexport` functions can be called by the host. The default build mode creates an executable; calling exported functions from that shape can fail with `module closed with exit_code(0)` or `runtime.notInitialized()`.

Use platform data capability APIs only. Do not write SQL strings or concatenate user input into SQL.

`data_list`, `data_join_query`, and `data_run_query` return row data:

```json
{
  "rows": [{ "id": 1, "title": "Example" }],
  "total": 1
}
```

`data_create`, `data_update`, and `data_delete` return write results such as `record`, `id`, and `rowsAffected`.

## Frontend Files

`frontend.js` is the ESM entrypoint exported to the host. It may import relative modules from `./frontend/*.js`.

- `frontend.js`: exports `render(container, context)` and optional `dispose(container)`, wires the page and event listeners.
- `frontend/api.js`: wraps `context.invokeData(context.app.id, payload)` and action names.
- `frontend/state.js`: local state, filters, pagination, form defaults, normalization.
- `frontend/ui.js`: HTML rendering helpers and event binding helpers.
- `frontend/modal.js`: dialog, drawer, toast, confirm helpers.
- `frontend/styles.js`: named `injectStyles(root)` export.

`frontend/styles.js` must follow `style-guide.md`: scoped root class, standard tokens, compact operational controls, semantic badges, modal/toast styles, and responsive rules.

The runtime serves `frontend.js` and `frontend/*.js` as browser modules. Keep imported paths relative, for example:

```js
import { createApi } from './frontend/api.js'
import { renderTable } from './frontend/ui.js'
```

Do not use bare package imports, bundler-only syntax, JSX, TypeScript, CSS imports, or asset imports that the browser cannot load directly.

Style modules must export `injectStyles(root)` and be imported exactly as `import { injectStyles } from './frontend/styles.js'`. Avoid alternate export names, default exports, or globals such as `window.__APP_STYLES__`.

After writing or changing frontend modules, verify import/export consistency:

- Every named import in `frontend.js` must exist in the imported module.
- Every named import inside `frontend/*.js` must exist in the referenced module.
- `frontend/styles.js` must provide a named `injectStyles` export if any module imports it.
- Replace the scaffold `frontend.js`; do not leave placeholder imports or placeholder UI after generating real modules.
- If a module is renamed or split, update every import path that references it.
- When using `context.invokeData(...)`, handle the returned object directly as `{ success, data, error }`; do not expect a nested `response` field.

## Dynamic DOM And Popups

If CSS is scoped under an app root, dynamic nodes must be appended under that same root.

Correct:

```js
const root = container.querySelector('.my-app-root')
root.appendChild(modal)
```

Risky:

```js
container.appendChild(modal)
```

This matters for modals, overlays, toasts, dropdown menus, and confirmation dialogs. If the selector is `.my-app-root .modal`, appending the modal outside `.my-app-root` will make it render without the intended popup styles.
