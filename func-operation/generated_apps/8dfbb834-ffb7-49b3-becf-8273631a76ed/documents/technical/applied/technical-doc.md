# 教师管理 技术设计

功能 ID：`8dfbb834-ffb7-49b3-becf-8273631a76ed`
目标：为教务/人事管理人员提供教师档案维护与在职状态流转工具。支持教师列表检索与筛选、新建/编辑档案、在职/停用/离职状态流转、离职后删除归档。

## 1. 模块拆分

| 模块 | 文件 | 职责 | 依赖 |
| --- | --- | --- | --- |
| 前端入口 | `frontend.js` | 导出 `render(container, context)` / `dispose(container)`，装配页面、统一事件委托、初始化权限函数 `can` | `frontend/api.js`、`frontend/state.js`、`frontend/ui.js`、`frontend/modal.js`、`frontend/styles.js` |
| 前端 API | `frontend/api.js` | 封装 `context.invokeData(context.app.id, payload)`，按 action 提供 `teacherList`/`teacherDetail`/`teacherCreate`/`teacherUpdate`/`teacherChangeStatus`/`teacherDelete` | 无 |
| 前端状态 | `frontend/state.js` | 维护 `rows`/`filters`/`pagination`/`form`/`modalMode`/`loading`/`saving`/`error`/`detail`/`confirm`/`toast` 等状态与表单归一化/重置 | 无 |
| 前端 UI | `frontend/ui.js` | 渲染头部、工具栏、表格、分页、详情面板、表单弹窗内容、空/加载/错误块，并绑定事件委托 | `frontend/styles.js` |
| 前端弹窗 | `frontend/modal.js` | 提供表单弹窗、详情抽屉、确认框、toast 的 DOM 创建与追加到根节点 | `frontend/styles.js` |
| 前端样式 | `frontend/styles.js` | 导出命名函数 `injectStyles(root)`，注入 `ga-teacher-` 前缀作用域样式 | 无 |
| 后端入口 | `backend/main.go` | WASI 导出 `handle`，请求解码，按 action 分派到 handler | `backend/teacher_handlers.go` |
| 后端平台 | `backend/platform.go` | `data_list`/`data_get`/`data_create`/`data_update`/`data_delete` 宿主调用与结果读取 | 无 |
| 后端模型 | `backend/models.go` | 请求/响应信封、`Teacher` 行结构体、逻辑模型名、状态常量、表单载荷结构体 | 无 |
| 后端处理 | `backend/teacher_handlers.go` | 实现 `teacher_list`/`teacher_detail`/`teacher_create`/`teacher_update`/`teacher_change_status`/`teacher_delete` | `backend/validators.go` |
| 后端校验 | `backend/validators.go` | 必填/工号与电话格式/枚举/分页边界/ID 校验、状态流转规则、工号唯一性检查 | 无 |

## 2. 页面与组件设计

页面为单工作台：教师列表为主页面，新建/编辑走居中弹窗表单，详情走右侧抽屉，状态流转与删除走确认框，操作结果走 toast。

```text
root(.ga-teacher-root)
├─ header：标题「教师管理」+ 副标题 + 右侧「新建教师」主按钮
├─ toolbar：搜索框 + 院系/国籍/状态/学历/职称筛选 + 「刷新」「清除筛选」
├─ content：紧凑型表格（教师记录）+ 加载/空/错误状态
├─ pagination：分页控件
├─ detail：右侧详情抽屉（点击「详情」打开）
├─ modal：新建/编辑表单弹窗
├─ confirm：状态流转/删除确认框
└─ toast：成功/失败反馈
```

| 区域/组件 | 展示内容 | 用户操作 | 触发 action | 状态反馈 |
| --- | --- | --- | --- | --- |
| 页面头部 | 标题「教师管理」、副标题「维护教师档案与在职状态」、右侧主按钮 | 点击「新建教师」 | 无（打开创建弹窗） | 弹窗打开 |
| 工具栏 | 搜索框、院系/国籍/状态/学历/职称五个下拉筛选、「刷新」「清除筛选」 | 输入关键词 / 选择筛选值 / 点击刷新 / 点击清除筛选 | `teacher_list` | 列表刷新；无结果时显示空结果块 |
| 教师表格 | 列：姓名、工号、所属院系、职称、联系电话、电子邮箱、状态（徽章）、入职日期、最近更新时间、操作 | 行内操作：详情 / 编辑 / 停用·在职 / 离职 / 删除 | `teacher_list`（读取），`teacher_detail`、`teacher_update`、`teacher_change_status`、`teacher_delete` | 行数据变化、状态徽章切换、行移除 |
| 分页栏 | 上一页 / 页码 / 下一页、共 N 条 | 点击翻页 | `teacher_list` | 当前页数据刷新 |
| 新建/编辑表单弹窗 | 按「基本信息 / 联系方式 / 任职信息」分组展示字段，其中「国籍」为下拉单选（选项读取全局配置键 `country`），「保存」「取消」 | 填写并保存 / 取消 | `teacher_create` / `teacher_update` | 保存成功关闭弹窗并刷新列表；校验失败字段下方红字 |
| 详情抽屉 | 姓名、工号、国籍、性别、出生日期、学历、所属院系、职称、联系电话、电子邮箱、入职日期、状态、最近更新时间；顶部按状态显示操作按钮 | 点击编辑 / 停用·在职 / 离职 / 删除 | `teacher_detail`、`teacher_update`、`teacher_change_status`、`teacher_delete` | 抽屉内容刷新；读取失败显示错误与「重试」 |
| 确认框 | 二次确认文案（状态流转或删除） | 确认 / 取消 | `teacher_change_status` / `teacher_delete` | 确认后执行并刷新；取消关闭 |
| Toast | 成功/失败提示 | 无 | 无（结果反馈） | 短暂展示 |

## 3. 数据模型

逻辑实体：教师（Teacher）。平台托管字段 `id`、`uuid`、`created_at`、`updated_at`，「最近更新时间」取 `updated_at`。

| 实体 | 字段 | 类型语义 | UI 用途 | 校验 | 默认值 |
| --- | --- | --- | --- | --- | --- |
| Teacher | name | string | 列表列、表单必填项、搜索关键词 | 必填，1-50 字 | 无 |
| Teacher | employee_no | string | 列表列、表单必填项、搜索关键词 | 必填，3-30 位字母数字，全局唯一 | 无 |
| Teacher | country | string | 详情展示、表单下拉、筛选条件 | 可选；若填写，值须为全局配置键 `country` 选项中的 `value` | 无 |
| Teacher | gender | enum: male/female | 详情展示、表单选择 | 可选 | male |
| Teacher | birth_date | string（yyyy-MM-dd） | 详情展示、表单日期 | 可选，合法日期 | 无 |
| Teacher | education | enum: bachelor/master/doctor/other | 详情展示、表单选择、筛选条件 | 可选 | 无 |
| Teacher | department | string | 列表列、表单必填项、筛选条件 | 必填，1-50 字 | 无 |
| Teacher | title | string | 列表列、详情展示、表单输入、筛选条件 | 可选，1-50 字 | 无 |
| Teacher | phone | string | 列表列、表单必填项、搜索关键词 | 必填，手机/座机格式，≤20 字 | 无 |
| Teacher | email | string | 列表列、详情展示、表单输入 | 可选，合法邮箱格式，≤100 字 | 无 |
| Teacher | hire_date | string（yyyy-MM-dd） | 列表列、详情展示、表单日期 | 可选，合法日期 | 无 |
| Teacher | status | enum: active/suspended/resigned | 列表徽章、详情展示、筛选条件、流转操作 | 不可直接编辑，由 `teacher_change_status` 维护 | active |

**状态显示文案与流转目标**

| 状态值 | 显示文案 | 徽章色 |
| --- | --- | --- |
| active | 在职 | green |
| suspended | 停用 | amber |
| resigned | 离职 | gray |

## 4. API 设计

统一信封：请求 `{ "action": "<action>", "data": { ... } }`；单条响应 `{ "success": true, "data": {}, "error": "" }`；列表响应 `{ "success": true, "rows": [], "total": 0, "error": "" }`。失败统一 `{ "success": false, "data": null, "error": "<用户可读信息>" }`。

| Action | 触发场景 | Request `data` | Response | 成功后前端状态 | 失败展示 | 权限模式 |
| --- | --- | --- | --- | --- | --- | --- |
| `teacher_list` | 初始加载、搜索、筛选、刷新、翻页 | `{ keyword, country, department, status, education, title, page, pageSize }` | `{ success, rows, total }` | 写入 `rows`/`total`，渲染表格 | toast/页内错误+重试 | read_default |
| `teacher_detail` | 点击行「详情」、详情重试 | `{ id }` | `{ success, data: { Teacher } }` | 写入 `detail`，打开抽屉 | 抽屉内错误+重试 | read_default |
| `teacher_create` | 新建表单保存 | `{ name, employee_no, country, gender, birth_date, education, department, title, phone, email, hire_date }` | `{ success, data: { Teacher } }` | 关闭弹窗、清空表单、刷新列表 | 字段标红；工号重复/保存失败 toast | button_controlled |
| `teacher_update` | 编辑表单保存 | `{ id, name, employee_no, country, gender, birth_date, education, department, title, phone, email, hire_date }` | `{ success, data: { Teacher } }` | 关闭弹窗、刷新列表 | 字段标红；工号重复/保存失败 toast | button_controlled |
| `teacher_change_status` | 停用 / 恢复在职 / 离职确认 | `{ id, status }`（status ∈ active/suspended/resigned） | `{ success, data: { Teacher } }` | 关闭确认框、刷新列表、更新详情 | toast 提示流转失败原因 | button_controlled |
| `teacher_delete` | 删除确认 | `{ id }` | `{ success, data: null }` | 关闭确认框、从列表移除、关闭详情抽屉 | toast 提示（在职/停用不可删除） | button_controlled |

权限模式说明：`teacher_list`、`teacher_detail` 为 read_default，不写入 `manifest.actions`，控件默认可见。`teacher_create`、`teacher_update`、`teacher_change_status`、`teacher_delete` 为 button_controlled，写入 `manifest.actions`，前端用 `can(actionKey)` 控制显示并在事件处理时二次校验，后端由宿主按 action 校验按钮权限。

国籍选项读取约定：国籍下拉与筛选的选项不通过后端 action 获取，也不属于本功能内置数据。前端通过 `context.config.get('country')` 读取全局公共配置键 `country`（返回 `{ value, label }[]`），渲染为下拉选项，业务记录仅持久化 `value`。每次打开表单弹窗、编辑弹窗与筛选栏时重新读取，配置更新后即时生效；不缓存到后端，也不写死兜底选项。读取失败或配置键不存在时的处理见第 6 节。

## 5. 状态流转与校验规则

### 状态变量

| 状态变量 | 类型 | 初始值 | 变化时机 | 影响 UI |
| --- | --- | --- | --- | --- |
| `rows` | Array | `[]` | `teacher_list` 成功 | 渲染表格行 |
| `total` | Number | `0` | `teacher_list` 成功 | 分页总数 |
| `loading` | Boolean | `false` | 列表请求发起/返回 | 表格区加载占位，禁止重复操作 |
| `saving` | Boolean | `false` | 表单/确认框提交中 | 保存按钮 loading 并禁止重复点击 |
| `error` | String/null | `null` | 列表加载失败 | 页内错误块 + 重试 |
| `filters` | Object | `{ keyword:'', country:'', department:'', status:'', education:'', title:'' }` | 搜索/筛选输入、清除筛选 | 重新请求列表 |
| `pagination` | Object | `{ page:1, pageSize:10 }` | 翻页 | 请求列表对应页 |
| `selectedId` | Number/null | `null` | 打开详情 | 高亮行 |
| `detail` | Object/null | `null` | `teacher_detail` 成功 | 详情抽屉内容 |
| `detailError` | String/null | `null` | 详情读取失败 | 抽屉内错误 + 重试 |
| `modalMode` | 'create'/'edit'/null | `null` | 点击新建/编辑/保存完成 | 表单弹窗开合 |
| `form` | Object | 空表单对象（含 `country:''`） | 打开弹窗/输入 | 表单字段值 |
| `formErrors` | Object | `{}` | 校验 | 字段下方红字 |
| `confirm` | Object/null | `null` | 点击停用·在职/离职/删除 | 确认框开合 |
| `toast` | Object/null | `null` | 操作结果 | toast 展示 |
| `countryOptions` | Array | `[]` | `context.config.get('country')` 成功 | 表单/筛选国籍下拉选项 |
| `countryOptionsError` | String/null | `null` | 国籍配置读取失败 | 下拉显示「选项加载失败」+「重试」 |

### 业务状态流转

| 业务状态 | 显示文案 | 可执行操作 | 下一状态 | 禁止条件 |
| --- | --- | --- | --- | --- |
| active | 在职 | 停用、离职 | suspended、resigned | 直接删除被拦截，须先办理离职 |
| suspended | 停用 | 恢复在职、离职 | active、resigned | 直接删除被拦截，须先办理离职 |
| resigned | 离职 | 删除归档 | 无（状态终态） | 任何状态流转均被拦截 |

前端按钮显示规则：active 显示「停用」「离职」；suspended 显示「在职」（恢复）、「离职」；resigned 不显示任何状态流转按钮，仅显示「删除」。

### 字段校验规则

| 字段 | 前端校验 | 后端校验 | 错误文案 |
| --- | --- | --- | --- |
| name | 必填，1-50 字 | 必填，≤50 字 | 请输入姓名 |
| employee_no | 必填，3-30 位字母数字 | 必填，3-30 位字母数字，唯一 | 请输入正确的工号；工号已存在，请检查 |
| country | 可选；填写时须为配置选项 value | 可选，值须为全局配置 `country` 选项中的 value | 请选择正确的国籍 |
| department | 必填，1-50 字 | 必填，≤50 字 | 请输入所属院系 |
| phone | 必填，手机/座机格式 | 必填，合法电话格式 | 请输入正确的联系电话 |
| email | 可选，合法邮箱格式 | 可选，≤100 字，合法邮箱格式 | 请输入正确的电子邮箱 |
| gender | 可选，枚举值 | 枚举校验 | 请选择正确的性别 |
| education | 可选，枚举值 | 枚举校验 | 请选择正确的学历 |
| birth_date / hire_date | 可选，合法日期 | 可选，合法日期 | 请输入正确的日期 |

### 按钮禁用规则

- 提交中（`saving=true`）时「保存」「确认」禁用并展示 loading。
- 受控按钮在 `can(actionKey)` 为 false 时不渲染。
- resigned 状态不渲染「停用/在职/离职」按钮。

### 后端校验规则

- 所有写操作必须校验 `id` 存在（`teacher_update`/`teacher_change_status`/`teacher_delete`/`teacher_detail`）。
- `teacher_create`/`teacher_update` 校验必填、格式、枚举，并执行工号唯一性检查（排除自身 id）。
- `teacher_change_status` 校验目标枚举值、当前状态存在、目标与当前不同、且当前状态非 resigned。
- `teacher_delete` 校验当前状态为 resigned，否则返回对应提示。
- 校验失败统一返回 `success:false` 与用户可读 `error`，不修改数据。

## 6. 异常与空状态处理

| 场景 | 触发条件 | UI 展示 | 恢复动作 | 对应 action |
| --- | --- | --- | --- | --- |
| 列表加载中 | 首次进入或刷新时请求未返回 | 表格区显示加载占位 | 等待返回后渲染 | `teacher_list` |
| 无任何记录 | `teacher_list` 返回 total=0 且无筛选 | 表格区显示「暂无教师」+「新建教师」入口 | 点击「新建教师」打开创建弹窗 | `teacher_list` |
| 筛选无结果 | 有筛选条件且 total=0 | 表格区显示「没有符合条件的教师」+「清除筛选」入口 | 点击「清除筛选」重置筛选并刷新 | `teacher_list` |
| 列表加载失败 | `teacher_list` 返回失败 | 页内错误块 + 「重试」按钮 | 点击「重试」重新请求 | `teacher_list` |
| 表单校验失败 | 必填缺失/格式非法 | 对应字段下方红字提示，阻止提交 | 修正后重新提交 | `teacher_create`/`teacher_update` |
| 工号重复 | 新建/编辑时工号已存在 | 工号字段下方红字 + toast「工号已存在，请检查」 | 修改工号后重新提交 | `teacher_create`/`teacher_update` |
| 删除在职/停用教师 | 对 active/suspended 教师点删除 | toast「在职教师不可删除，请先办理离职」/「停用教师不可删除，请先办理离职」 | 先流转为离职再删除 | `teacher_delete` |
| 对离职教师再操作状态 | resigned 教师触发状态流转 | toast「已离职教师不可再变更状态」 | 无需操作 | `teacher_change_status` |
| 详情读取失败 | `teacher_detail` 返回失败 | 抽屉内错误提示 + 「重试」 | 点击「重试」重新请求 | `teacher_detail` |
| 国籍配置加载失败 | `context.config.get('country')` 失败或配置键不存在 | 表单/筛选栏国籍下拉置灰并显示「选项加载失败」+「重试」；国籍为可选字段，不阻断其余字段保存 | 点击「重试」重新读取配置后恢复下拉 | `teacher_create`/`teacher_update`/`teacher_list` |
| 保存/删除/流转失败 | 后端返回失败 | toast 展示失败原因；保留弹窗内容或列表状态 | 修正或重试 | `teacher_create`/`teacher_update`/`teacher_change_status`/`teacher_delete` |
| 无权限 | 用户缺少受控按钮权限 | 对应按钮不渲染；只读操作始终可用 | 无 | 无 |

## 7. 实施步骤

1. 编写 `manifest.json`：设置 `id`、`name`、`description`、`frontendFile`、`backendSource`、`backendModule`、`dataModels`、`configKeys:["country"]`、`actions`。
2. 编写 `backend/models.go`：信封结构、`Teacher` 行结构体、表单载荷结构体、逻辑模型名与状态常量。
3. 编写 `backend/platform.go`：`data_list`/`data_get`/`data_create`/`data_update`/`data_delete` 宿主调用。
4. 编写 `backend/validators.go`：必填、工号/电话/邮箱/日期格式、枚举、分页边界、ID、状态流转、工号唯一性校验。
5. 编写 `backend/teacher_handlers.go`：实现六个 action handler 并在 `backend/main.go` 中注册分派。
6. 编写 `frontend/api.js`：为六个 action 提供同名 API wrapper。
7. 编写 `frontend/state.js`：状态对象、默认表单、归一化与重置辅助函数、`loadCountryOptions(context)`（调用 `context.config.get('country')` 读取国籍选项）。
8. 编写 `frontend/ui.js`：头部/工具栏/表格/分页/详情/表单/空/加载/错误渲染与事件委托。
9. 编写 `frontend/modal.js`：表单弹窗、详情抽屉、确认框、toast 组件。
10. 编写 `frontend/styles.js`：`injectStyles(root)`，覆盖全部 `ga-teacher-` 类。
11. 编写 `frontend.js`：`render(container, context)`/`dispose`，注入 `can`，初始化并绑定事件。
12. 从 `backend/` 目录构建 `backend.wasm`：`GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../backend.wasm .`。
13. 核对 `backend.wasm` 比所有 `*.go` 新；执行第 14 节验收自检。

## 8. 代码目录与文件清单（对接 generated-app-builder）

```text
generated_apps/8dfbb834-ffb7-49b3-becf-8273631a76ed/
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
    teacher_handlers.go
    validators.go
  backend.wasm
```

| 文件 | 必须生成 | 主要内容 |
| --- | --- | --- |
| `manifest.json` | 是 | 应用元信息、`dataModels`、`actions` |
| `frontend.js` | 是 | 入口 `render`/`dispose`，装配页面与事件委托，`can` 注入 |
| `frontend/api.js` | 是 | 六个后端 action 的 API wrapper |
| `frontend/state.js` | 是 | 状态对象、默认表单、归一化/重置、国籍选项加载 `loadCountryOptions` |
| `frontend/ui.js` | 是 | 头部/工具栏/表格/分页/详情/表单/空/加载/错误渲染 |
| `frontend/modal.js` | 是 | 表单弹窗、详情抽屉、确认框、toast |
| `frontend/styles.js` | 是 | `injectStyles(root)` 与全部作用域样式 |
| `backend/main.go` | 是 | WASI 入口、请求解码、action 分派 |
| `backend/platform.go` | 是 | 平台数据能力宿主调用 |
| `backend/models.go` | 是 | 信封、Teacher 结构体、逻辑模型名、状态常量 |
| `backend/teacher_handlers.go` | 是 | 六个 action handler |
| `backend/validators.go` | 是 | 必填/格式/枚举/分页/ID/状态流转/工号唯一校验 |
| `backend.wasm` | 是 | 由 backend 源码构建，须新于所有 `*.go` |

## 9. Manifest 与表结构约定

- `manifest.id` = `8dfbb834-ffb7-49b3-becf-8273631a76ed`，与文件夹名一致。
- `manifest.name` = 教师管理。
- `manifest.description` = 维护教师档案与在职/停用/离职状态。
- `manifest.actions`（JSON 字符串数组）：`["teacher_create", "teacher_update", "teacher_change_status", "teacher_delete"]`。
- `manifest.configKeys`（JSON 字符串数组）：`["country"]`，声明本功能读取的全局公共配置键；国籍选项不内置、不写死、不建表。
- 不设计物理表名；平台由 `dataModels` 生成真实表。无 `relations`、无 `queries`（列表读取通过 `data_list` 动态过滤，无需预声明查询）。

| 模型 | manifest `dataModels[].name` | 字段 | 校验 | 索引建议 |
| --- | --- | --- | --- | --- |
| 教师 | `Teacher` | `name` string | required, maxLength 50 | 无 |
| 教师 | `Teacher` | `employee_no` string | required, maxLength 30 | 唯一索引（用于唯一性检查与检索） |
| 教师 | `Teacher` | `country` string | maxLength 50；可选 | 无（选项来源为全局配置，不建表） |
| 教师 | `Teacher` | `gender` enum | values `["male","female"]` | 无 |
| 教师 | `Teacher` | `birth_date` string | 可选 | 无 |
| 教师 | `Teacher` | `education` enum | values `["bachelor","master","doctor","other"]` | 无 |
| 教师 | `Teacher` | `department` string | required, maxLength 50 | 索引（用于筛选） |
| 教师 | `Teacher` | `title` string | maxLength 50 | 无 |
| 教师 | `Teacher` | `phone` string | required, maxLength 20 | 无 |
| 教师 | `Teacher` | `email` string | maxLength 100 | 无 |
| 教师 | `Teacher` | `hire_date` string | 可选 | 无 |
| 教师 | `Teacher` | `status` enum | values `["active","suspended","resigned"]`, default `"active"` | 索引（用于筛选） |

枚举显示映射：`male`=男、`female`=女；`bachelor`=本科、`master`=硕士、`doctor`=博士、`other`=其他；`active`=在职、`suspended`=停用、`resigned`=离职。

## 10. 后端 Action 清单

| Action | Handler 文件 | Handler 函数 | Request | Response | Data capability 调用 | 校验 |
| --- | --- | --- | --- | --- | --- | --- |
| `teacher_list` | `teacher_handlers.go` | `handleTeacherList` | `{ keyword, country, department, status, education, title, page, pageSize }` | `{ success, rows, total }` | `data_list(model=Teacher, filters={keyword→name/employee_no/phone contains, country, department, status, education, title}, sort=updated_at desc, page, pageSize)` | page≥1，pageSize 1-100 |
| `teacher_detail` | `teacher_handlers.go` | `handleTeacherDetail` | `{ id }` | `{ success, data: Teacher }` | `data_get(model=Teacher, id)` | id 必填且存在 |
| `teacher_create` | `teacher_handlers.go` | `handleTeacherCreate` | 表单字段（含 `country`） | `{ success, data: Teacher }` | 唯一性检查 `data_list(model=Teacher, filters={employee_no})`；`data_create(model=Teacher, record)` | 必填/格式/枚举；工号唯一 |
| `teacher_update` | `teacher_handlers.go` | `handleTeacherUpdate` | `{ id, 表单字段 }`（含 `country`） | `{ success, data: Teacher }` | 唯一性检查 `data_list(model=Teacher, filters={employee_no})`（排除自身）；`data_update(model=Teacher, id, record)` | id 必填存在；必填/格式/枚举；工号唯一 |
| `teacher_change_status` | `teacher_handlers.go` | `handleTeacherChangeStatus` | `{ id, status }` | `{ success, data: Teacher }` | `data_get(model=Teacher, id)`；`data_update(model=Teacher, id, {status})` | id 存在；status ∈ 枚举；当前状态非 resigned；目标≠当前 |
| `teacher_delete` | `teacher_handlers.go` | `handleTeacherDelete` | `{ id }` | `{ success, data: null }` | `data_get(model=Teacher, id)`；`data_delete(model=Teacher, id)` | id 存在；当前状态为 resigned，否则返回「在职/停用教师不可删除，请先办理离职」 |

分派说明：`backend/main.go` 对请求 `action` 字符串与上述六值完全一致地分派。所有写 action 均被宿主按 `manifest.actions` 做按钮权限校验；只读 action 仅依赖功能菜单权限。

## 11. 前端模块拆分

| 文件 | 关键导出 | 依赖 | 被谁调用 |
| --- | --- | --- | --- |
| `frontend.js` | `render(container, context)`、`dispose(container)`、内部 `can(action)` | `./frontend/api.js`、`state.js`、`ui.js`、`modal.js`、`styles.js` | 宿主运行时 |
| `frontend/api.js` | `teacherList(params)`、`teacherDetail(id)`、`teacherCreate(record)`、`teacherUpdate(id, record)`、`teacherChangeStatus(id, status)`、`teacherDelete(id)` | `context.invokeData` | `frontend.js`、`state.js` |
| `frontend/state.js` | `createInitialState()`、`defaultForm()`、`normalizeRows(list)`、`resetFilters(state)`、`mergeForm(state, patch)`、`loadCountryOptions(context)` | `context.config.get('country')` | `frontend.js`、`ui.js` |
| `frontend/ui.js` | `renderShell(root, ctx)`、`renderToolbar(root, ctx)`、`renderTable(root, ctx)`、`renderPagination(root, ctx)`、`renderEmptyOrError(root, ctx)`、`renderDetailDrawer(root, ctx)`、`renderFormBody(ctx)`、`bindEvents(root, ctx)` | `frontend/styles.js`、`frontend/modal.js` | `frontend.js` |
| `frontend/modal.js` | `openFormModal(root, ctx, mode)`、`openDetailDrawer(root, ctx)`、`openConfirm(root, ctx, opts)`、`showToast(root, type, message)`、`closeTopModal(root)` | `frontend/styles.js` | `frontend.js`、`ui.js` |
| `frontend/styles.js` | `injectStyles(root)` | 无 | `frontend.js` |

API wrapper action 字符串与 Section 4 / Section 10 完全一致；受控 action 的 UI 渲染与事件处理均使用 `can(actionKey)`。

国籍选项读取：`frontend/api.js` 不包含国籍读取 wrapper（国籍非后端 action）；由 `frontend/state.js` 的 `loadCountryOptions(context)` 调用 `context.config.get('country')` 获取 `{ value, label }[]` 并写入 `state.countryOptions`，表单/筛选下拉据此渲染，业务记录仅存 `value`。

## 12. 前端交互流程与状态合同

| 操作 | 触发控件 | 前端状态变化 | 调用 action | 成功反馈 | 失败反馈 |
| --- | --- | --- | --- | --- | --- |
| 初始加载 | `render` | `loading=true` → 请求 → `rows/total` 更新，`loading=false` | `teacher_list` | 渲染表格 | `error` 块+重试 |
| 国籍选项加载 | 初始渲染 / 打开新建/编辑弹窗 / 打开筛选栏 | `countryOptionsLoading=true` → `context.config.get('country')` → `countryOptions` 更新 | 无（非后端 action） | 表单/筛选国籍下拉渲染选项 | `countryOptionsError` 置位，下拉显示「选项加载失败」+「重试」，不阻断其余字段保存 |
| 搜索 | 搜索框输入 | `filters.keyword` 更新，`pagination.page=1`，重新加载 | `teacher_list` | 列表刷新 | 空结果块 / 错误块 |
| 筛选 | 院系/国籍/状态/学历/职称下拉 | `filters.*` 更新，`pagination.page=1`，重新加载 | `teacher_list` | 列表刷新 | 空结果块 / 错误块 |
| 清除筛选 | 「清除筛选」按钮 | `resetFilters`，`pagination.page=1`，重新加载 | `teacher_list` | 列表恢复全部 | 错误块 |
| 刷新 | 「刷新」按钮 | `loading=true`，重新加载 | `teacher_list` | 列表刷新 | 错误块 |
| 翻页 | 分页控件 | `pagination.page` 更新，重新加载 | `teacher_list` | 对应页渲染 | 错误块 |
| 新建 | 「新建教师」按钮（`can('teacher_create')`） | `modalMode='create'`，`form=defaultForm()`，`formErrors={}` | 无 | 打开表单弹窗 | 无 |
| 保存新建 | 弹窗「保存」 | `saving=true` → 校验 → 提交 → `modalMode=null`，刷新列表 | `teacher_create` | 关闭弹窗，列表出现新记录，toast 成功 | 字段标红；toast 失败，保留表单 |
| 编辑 | 行/详情「编辑」（`can('teacher_update')`） | `modalMode='edit'`，`form=记录值`，`selectedId=id` | 无 | 打开表单弹窗 | 无 |
| 保存编辑 | 弹窗「保存」 | `saving=true` → 校验 → 提交 → `modalMode=null`，刷新列表 | `teacher_update` | 关闭弹窗，列表刷新，toast 成功 | 字段标红；toast 失败，保留表单 |
| 停用 | 行/详情「停用」（`can('teacher_change_status')`） | `confirm={ text, status:'suspended', id }` | 无 | 打开确认框 | 无 |
| 恢复在职 | 行/详情「在职」（`can('teacher_change_status')`） | `confirm={ text, status:'active', id }` | 无 | 打开确认框 | 无 |
| 离职 | 行/详情「离职」（`can('teacher_change_status')`） | `confirm={ text, status:'resigned', id }` | 无 | 打开确认框 | 无 |
| 确认流转 | 确认框「确认」 | `saving=true` → 提交 → `confirm=null`，刷新列表/详情 | `teacher_change_status` | toast 成功，状态徽章更新 | toast 失败原因 |
| 删除 | 行/详情「删除」（`can('teacher_delete')`） | 前端预检状态：非 resigned 直接 toast 拦截；resigned 打开 `confirm` | `teacher_delete` | 确认后从列表移除，关闭详情，toast 成功 | toast「在职/停用教师不可删除」 |
| 详情 | 行「详情」 | `selectedId=id`，`detailError=null`，请求详情 | `teacher_detail` | 打开详情抽屉 | `detailError` + 重试 |
| 详情重试 | 抽屉「重试」 | 重新请求详情 | `teacher_detail` | 抽屉刷新 | `detailError` + 重试 |

### 前端 Action 对齐表

| UI 操作 | DOM 控件/事件 | API wrapper | Backend action | 成功后刷新 |
| --- | --- | --- | --- | --- |
| 加载/搜索/筛选/翻页/刷新 | toolbar/分页 change/click | `teacherList` | `teacher_list` | 表格 |
| 国籍选项加载 | render / 打开弹窗 / 筛选栏 | `state.loadCountryOptions(context)`（`context.config.get('country')`） | 无（非 invokeData action） | 表单/筛选下拉 |
| 打开详情/重试 | 行按钮/抽屉按钮 | `teacherDetail` | `teacher_detail` | 抽屉 |
| 保存新建 | 弹窗保存 click | `teacherCreate` | `teacher_create` | 表格 |
| 保存编辑 | 弹窗保存 click | `teacherUpdate` | `teacher_update` | 表格 |
| 停用/恢复在职/离职 | 行/详情按钮→确认框 | `teacherChangeStatus` | `teacher_change_status` | 表格+详情 |
| 删除 | 行/详情按钮→确认框 | `teacherDelete` | `teacher_delete` | 表格+详情 |

## 13. 样式规范与视觉一致性

采用共享 style-guide 的 `ga-` 设计令牌与中性灰底、蓝色主操作、语义徽章风格。应用根类 `ga-teacher-root`，全部选择器以 `ga-teacher-` 为命名空间前缀，禁止全局裸样式。动态节点（弹窗/抽屉/确认框/toast）一律追加到 `.ga-teacher-root` 之下。

| UI 元素 | class 命名 | 样式要求 | 状态 |
| --- | --- | --- | --- |
| 根容器 | `.ga-teacher-root` | `width:100%`、`min-height:100%`、`box-sizing:border-box` | 常驻 |
| 头部 | `.ga-teacher-header`、`.ga-teacher-header-title`、`.ga-teacher-header-desc` | 标题 20-22px，副标题 13-14px，主操作右对齐；窄屏纵向堆叠 | 常驻 |
| 工具栏 | `.ga-teacher-toolbar`、`.ga-teacher-search`、`.ga-teacher-select` | flex 可换行，搜索框 min-width 200px，筛选用 select | 常驻 |
| 主按钮 | `.ga-teacher-btn-primary` | 蓝底白字，8px 14px padding，6px 圆角 | 禁用降透明度 |
| 次按钮 | `.ga-teacher-btn-secondary` | 白底灰边 | 禁用降透明度 |
| 危险按钮 | `.ga-teacher-btn-danger` | 红字，soft 红 hover | 禁用降透明度 |
| 小按钮 | `.ga-teacher-btn-small` | 4px 10px padding，13px 字号 | 行内操作 |
| 输入/下拉 | `.ga-teacher-input`、`.ga-teacher-select` | 8px 12px padding，灰边，6px 圆角，蓝色 focus 光环 | 校验失败红边 |
| 下拉选项错误 | `.ga-teacher-options-error` | 国籍下拉旁红字「选项加载失败」，与「重试」按钮并列；配置未加载时下拉置灰 | 配置读取失败 |
| 表单 | `.ga-teacher-form`、`.ga-teacher-form-group`、`.ga-teacher-form-group-title`、`.ga-teacher-field`、`.ga-teacher-label`、`.ga-teacher-error` | 按「基本信息/联系方式/任职信息」分组，字段错误红字 | 校验错误 |
| 表格 | `.ga-teacher-table-wrap`、`.ga-teacher-table`、`.ga-teacher-row-actions` | 1px 边、8px 圆角、表头 `#f9fafb`、行 hover、窄屏横向滚动 | 加载/空/错误态 |
| 徽章 | `.ga-teacher-badge`、`.ga-teacher-badge-active`、`.ga-teacher-badge-suspended`、`.ga-teacher-badge-resigned` | 12px、500 字重、10px 圆角；green/amber/gray | 按状态 |
| 弹窗 | `.ga-teacher-modal-mask`、`.ga-teacher-modal`、`.ga-teacher-modal-header`、`.ga-teacher-modal-body`、`.ga-teacher-modal-footer` | 遮罩 `position:absolute; inset:0` 于根节点内；居中面板 420-560px，`max-width:calc(100vw - 32px)`，8px 圆角 | 开/合 |
| 抽屉 | `.ga-teacher-drawer`、`.ga-teacher-detail`、`.ga-teacher-detail-item` | 右侧面板，展示完整档案字段，操作区靠上 | 读取失败错误态 |
| 确认框 | `.ga-teacher-confirm`（复用 modal 类） | 居中弹窗，文案+确认/取消 | 开/合 |
| Toast | `.ga-teacher-toast`、`.ga-teacher-toast-success`、`.ga-teacher-toast-error` | 根内右上角，8px 圆角；成功自动消失，错误保留至用户操作 | 出现/消失 |
| 加载/空/错误块 | `.ga-teacher-loading`、`.ga-teacher-empty`、`.ga-teacher-error-block` | 居中提示；空状态带下一步按钮，错误带「重试」 | 按状态 |
| 分页 | `.ga-teacher-pagination` | 底部分页，展示共 N 条 | 常驻 |

响应式：低于 720px 时头部纵向堆叠、工具栏控件全宽、表格保持横向滚动、行操作按钮可换行、弹窗宽 `calc(100vw - 32px)`、避免固定高度裁切内容。

## 14. 验收与自检清单

- 首次进入调用 `teacher_list`，表格渲染真实控件（搜索框、下拉筛选、按钮），非文本摘要。
- 新建/编辑表单在调用后端前完成必填/格式校验，错误显示在字段下方。
- 工号重复时前后端均拦截并提示「工号已存在，请检查」。
- 停用/恢复在职/离职经确认框后状态徽章正确切换；resigned 不再显示流转按钮。
- 删除 active/suspended 教师被前端预检与后端双重拦截；resigned 可删除。
- 受控按钮（新建/编辑/停用·在职/离职/删除）仅 `can(actionKey)` 为 true 时渲染，事件处理二次校验；搜索/筛选/详情/刷新/分页默认可见。
- 国籍下拉与筛选选项来自全局配置键 `country`（`context.config.get`），不内置、不写死；`manifest.configKeys` 声明该键。
- 国籍配置加载失败时下拉显示「选项加载失败」+「重试」，可恢复，且不阻断其余字段保存。
- 加载、空、无结果、错误、校验、无权限状态均可视且带恢复路径。
- `backend.wasm` 新于所有 `*.go`；后端分派 action 与前端 wrapper 字符串一致。
- `manifest.dataModels` 与后端 data capability 调用的逻辑模型一致；无物理表名/裸 SQL。
- 弹窗、抽屉、确认框、toast 均追加到 `.ga-teacher-root` 下且具有匹配作用域样式。
- 720px 以下布局可用，无全局样式泄漏，无占位符与生成应用内部文案外露。

| 检查项 | 通过标准 |
| --- | --- |
| Section 4 actions 与 Section 10 后端 actions 一致 | 六项 action 字符串逐一对应 |
| Section 10 actions 与 Section 11 `frontend/api.js` wrapper 一致 | `teacher_list/detail/create/update/change_status/delete` 六项 wrapper 齐全 |
| Section 9 模型名与后端常量一致 | `Teacher` 逻辑模型名唯一且一致；无 `relations`/`queries` 不引用 |
| Section 9 `configKeys` 与 `context.config.get('country')` 一致 | manifest 声明 `["country"]`，前端按此键读取，业务记录仅存 value |
| Section 13 class 与 Section 11 组件对应 | 每个渲染节点 class 均在 `injectStyles` 中存在 |
| 每个产品操作具备 UI/API/后端/校验/成功/失败处理 | 新建、编辑、停用、恢复在职、离职、删除、搜索、筛选、详情、刷新、分页均闭环 |
| 用户可见文案无生成应用内部信息 | 按钮、提示、空状态均为业务文案 |
| 无占位符内容 | 无 `<业务>`、`待生成`、`TODO` 等残留 |
