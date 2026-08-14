# 图书管理 技术设计

> 功能 ID：`0d4c75ff-bf06-4a98-91c0-ca8c4e86072d`
> 产品文档：`documents/product/applied/product-doc.md`
> 本技术设计是 `generated-app-builder` 的实现契约。所有 action 名、模型名、字段名、查询名、类名均以下文为准，避免在实现阶段另行发明。

---

## 1. 模块拆分

按运行时职责拆分，每个模块对应第 8 节的具体文件。

| 模块 | 文件 | 职责 | 依赖 |
| --- | --- | --- | --- |
| 前端入口 | `frontend.js` | 导出 `render(container, context)` / `dispose(container)`，负责页面根节点挂载、事件委托绑定、初始化加载、权限 `can` 包装 | `frontend/api.js`、`frontend/state.js`、`frontend/ui.js`、`frontend/modal.js`、`frontend/styles.js` |
| 前端 API | `frontend/api.js` | 封装 `context.invokeData(context.app.id, { action, data })`，为每个后端 action 提供同名 wrapper | `frontend/state.js`（无运行时依赖，仅常量） |
| 前端状态 | `frontend/state.js` | 定义 state 结构、默认值、常量（分类枚举、状态枚举与文案）、行数据归一化（计算 `available_count`、状态文案映射） | 无 |
| 前端 UI | `frontend/ui.js` | 渲染 header、toolbar、表格、行操作、详情面板、空/加载/错误占位，绑定事件委托 | `frontend/state.js`、`frontend/api.js` |
| 前端弹窗 | `frontend/modal.js` | 新建/编辑表单弹窗、借出/归还数量弹窗、删除/下架/上架确认弹窗、toast 反馈 | `frontend/state.js`、`frontend/api.js` |
| 前端样式 | `frontend/styles.js` | 导出 `injectStyles(root)`，注入 `ga-book-` 前缀作用域样式 | 无 |
| 后端入口 | `backend/main.go` | WASI 入口、`handle` 请求解码、action 字符串分发到 handler | `backend/platform.go`、`backend/models.go` |
| 后端平台能力 | `backend/platform.go` | 封装 `data_list`/`data_get`/`data_create`/`data_update`/`data_delete`/`data_run_query`/`result_*` 宿主调用 | 无 |
| 后端模型 | `backend/models.go` | 请求/响应 envelope、Book 行结构体、表单 payload、逻辑模型名/查询名常量、状态枚举 | 无 |
| 后端图书处理 | `backend/book_handlers.go` | 实现 `list`/`detail`/`create`/`update`/`delete`/`lend`/`return_book`/`offshelf`/`onshelf` 处理 | `backend/platform.go`、`backend/models.go`、`backend/validators.go` |
| 后端校验 | `backend/validators.go` | 必填、ISBN 格式、分类枚举、数量正整数、总库存下限、ISBN 唯一、可借/借出数量关系校验 | 无 |

---

## 2. 页面与组件设计

### 2.1 图书列表（工作台主页面）

布局草图：

```text
header
  标题：图书管理
  描述：维护图书档案与库存，跟踪借出/归还/下架
  右上角主操作：[+ 新建图书]  ← 仅 can('create') 时显示

toolbar
  搜索框（书名/ISBN/作者）  分类下拉（全部分类+枚举）  状态下拉（全部状态+在馆/借出/下架）
  [刷新]  [清除筛选]

content
  记录表格（紧凑型）
    | 书名 | ISBN | 作者 | 分类 | 馆藏位置 | 总库存 | 可借数量 | 状态 | 最近更新时间 | 操作 |
  分页：上一页 / 下一页 / 共 N 条
```

- 首次进入自动调用 `list` 加载第一页。
- 表格为空记录时，内容区显示"暂无图书" + "新建图书"入口。
- 筛选无结果时，内容区显示"没有符合条件的图书" + "清除筛选"按钮。
- 列表加载失败时，内容区显示错误提示 + "重试"按钮。
- 表格在 <720px 宽度下保持横向滚动，行操作按钮允许换行。

| 区域/组件 | 展示内容 | 用户操作 | 触发 action | 状态反馈 |
| --- | --- | --- | --- | --- |
| header 标题区 | 标题"图书管理"、一句描述 | 无 | - | - |
| header 主操作 | "新建图书"按钮 | 点击打开新建表单 | `create`（前端仅 `openBookForm`，提交才调 `create`） | 打开表单弹窗 |
| toolbar 搜索框 | placeholder"搜索书名/ISBN/作者" | 输入关键词回车或防抖触发 | `list` | 重新加载列表 |
| toolbar 分类筛选 | 下拉：全部分类 + 分类枚举 | 选择后触发 | `list` | 重新加载列表 |
| toolbar 状态筛选 | 下拉：全部状态 + 在馆/借出/下架 | 选择后触发 | `list` | 重新加载列表 |
| toolbar 刷新 | "刷新"按钮 | 点击按当前筛选重载第一页 | `list` | 列表刷新；失败显示错误占位 |
| toolbar 清除筛选 | "清除筛选"按钮 | 点击清空关键词/分类/状态并重载 | `list` | 筛选控件复位，列表重载 |
| 表格区 | 记录表格，字段见下 | 点击行操作按钮 | 见 2.1 行操作 | 见 2.1 行操作 |
| 分页区 | 上一页/下一页、"共 N 条" | 翻页 | `list` | 列表刷新到对应页 |
| 空状态块 | "暂无图书" | 点击"新建图书" | 打开新建表单 | 打开表单弹窗 |
| 空结果块 | "没有符合条件的图书" | 点击"清除筛选" | 复位筛选 + `list` | 列表重载 |
| 错误块 | "加载失败，请重试" | 点击"重试" | `list` | 重新加载列表 |
| 加载占位 | 骨架/loading | 无 | - | - |

**必见表字段与列顺序**：书名、ISBN、作者、分类、馆藏位置、总库存、可借数量、状态（badge）、最近更新时间、操作列。

**行操作按钮**（按权限与状态渲染）：

| 行操作 | 显示条件 | 点击行为 |
| --- | --- | --- |
| 详情 | 始终显示 | 打开右侧/弹窗详情面板，调 `detail` |
| 编辑 | `can('update')` | 打开编辑表单，调 `update` |
| 借出 | `can('lend')` | 若状态为下架，toast"该图书已下架，不能借出"；否则打开借出数量弹窗，提交调 `lend` |
| 归还 | `can('return_book')` | 打开归还数量弹窗，提交调 `return_book` |
| 下架/上架 | 非下架时 `can('offshelf')` 显示"下架"；下架时 `can('onshelf')` 显示"上架" | 打开二次确认弹窗，确认后调 `offshelf` / `onshelf` |
| 删除 | `can('delete')` | 打开二次确认弹窗，确认后调 `delete` |

### 2.2 新建 / 编辑图书表单（弹窗）

- 布局：居中弹窗（宽 480px），表单字段按"基础信息 / 库存信息"分组。
- 主操作："保存"；次要操作："取消"。
- 新建字段：书名、ISBN、作者、分类、出版社、出版年份、馆藏位置、总库存。
- 编辑字段：同新建 + 只读展示"借出数量""可借数量"。
- 校验失败在对应字段下方红字提示，阻止提交；保存期间按钮 loading 且禁止重复点击。

| 区域/组件 | 展示内容 | 用户操作 | 触发 action | 状态反馈 |
| --- | --- | --- | --- | --- |
| 弹窗头部 | 标题"新建图书"/"编辑图书"、关闭按钮 | 关闭 | - | 关闭弹窗 |
| 基础信息分组 | 书名*、ISBN*、作者、分类、出版社、出版年份、馆藏位置 | 输入 | - | 字段级校验红字 |
| 库存信息分组 | 总库存*；编辑模式额外只读展示借出数量、可借数量 | 输入 | - | 字段级校验红字 |
| 弹窗底部 | "保存"（主）、"取消"（次） | 点击 | 新建 `create` / 编辑 `update` | 保存中 loading；成功后关闭弹窗并刷新列表；失败 toast 显示原因，保留表单 |

### 2.3 图书详情（详情面板）

- 布局：点击"详情"打开弹窗/右侧面板。
- 展示字段（只读）：书名、ISBN、作者、分类、出版社、出版年份、馆藏位置、总库存、可借数量、借出数量、状态、最近更新时间。
- 面板底部展示与行操作一致的操作按钮（权限/状态同上）。
- 详情读取失败时显示错误提示 + "重试"。

| 区域/组件 | 展示内容 | 用户操作 | 触发 action | 状态反馈 |
| --- | --- | --- | --- | --- |
| 详情面板 | 全部字段（只读） | 无 | - | - |
| 详情操作区 | 编辑/借出/归还/下架·上架/删除 | 同 2.1 行操作 | 同 2.1 | 同 2.1 |
| 详情错误块 | "读取失败，请重试" | 点击"重试" | `detail` | 重新加载详情 |

### 2.4 借出 / 归还数量弹窗

- 布局：小弹窗（宽 360px），展示图书书名、当前可借数量（借出时）或借出数量（归还时），一个数量输入框。
- 默认值 1；主操作"确认"，次操作"取消"。
- 数量非法（非整数、<1）或超出上限时字段下方红字提示：借出超限提示"超出可借数量"；归还超限提示"归还数量大于借出数量"。

| 区域/组件 | 展示内容 | 用户操作 | 触发 action | 状态反馈 |
| --- | --- | --- | --- | --- |
| 数量弹窗 | 书名、当前可借/已借数量、数量输入 | 输入数量 | - | 字段级校验红字 |
| 弹窗底部 | "确认"（主）、"取消"（次） | 点击 | `lend` / `return_book` | 成功后关闭弹窗并刷新列表；失败 toast 保留弹窗与输入 |

### 2.5 删除 / 下架 / 上架确认弹窗

- 布局：确认弹窗（宽 360px），文案提示影响。
- 删除：确认文案"确定删除该图书？存在借出记录时将无法删除。"；确认调 `delete`。
- 下架：确认文案"确定将该图书下架？下架后不可借出。"；确认调 `offshelf`。
- 上架：确认文案"确定将该图书重新上架？"；确认调 `onshelf`。
- 主操作"确认"，次操作"取消"；提交失败 toast 提示且弹窗保留。

| 区域/组件 | 展示内容 | 用户操作 | 触发 action | 状态反馈 |
| --- | --- | --- | --- | --- |
| 确认弹窗 | 确认文案 | 确认/取消 | `delete` / `offshelf` / `onshelf` | 成功后关闭并刷新；失败 toast 保留 |

### 2.6 Toast 反馈

- 顶部右侧 toast：成功（绿色/自动消失）、失败（红色/需关闭或较长停留）。
- 用于新增、编辑、删除、借出、归还、下架、上架的结果反馈。

---

## 3. 数据模型

单一业务实体：**Book（图书）**。逻辑字段名如下，物理表名由平台根据 `dataModels` 生成，不设计表名。

| 实体 | 字段 | 类型语义 | UI 用途 | 校验 | 默认值 |
| --- | --- | --- | --- | --- | --- |
| Book | title | string | 列表标题、表单必填、搜索关键词 | 必填，1-120 字 | - |
| Book | isbn | string | 列表展示、表单必填、搜索关键词、唯一标识 | 必填，ISBN-10/ISBN-13 格式，全库唯一（排除自身） | - |
| Book | author | string | 列表/详情展示、表单、搜索关键词 | 选填，≤80 字 | '' |
| Book | category | enum | 列表/详情展示、表单下拉、分类筛选 | 选填，取值见枚举；为空显示"未分类" | "未分类" |
| Book | publisher | string | 详情展示、表单 | 选填，≤80 字 | '' |
| Book | publish_year | int | 详情展示、表单 | 选填，1000-2100 | null |
| Book | location | string | 列表/详情展示（馆藏位置）、表单 | 选填，≤60 字 | '' |
| Book | total_stock | int | 列表/详情展示、表单必填（总库存） | 必填，整数 ≥1 | 1 |
| Book | borrowed_count | int | 详情展示（借出数量）、归还/删除校验依据 | 不可直接编辑，整数 ≥0，恒 ≤ total_stock | 0 |
| Book | status | enum | 列表/详情 badge、状态筛选、借出/归还/下架/上架判定 | 取值 `onshelf`/`lent`/`offshelf`，由后端维护 | "onshelf" |
| Book | updated_at | datetime | 列表/详情展示（最近更新时间） | 平台隐式字段，不用在 dataModels 声明 | 平台管理 |

**派生值（不落库，后端在响应中计算返回）**：

- `available_count`（可借数量）= `total_stock - borrowed_count`。
- 展示状态 `status`：
  - `offshelf`（下架）：手动下架后的状态，不可借出。
  - 否则若 `available_count == 0` → `lent`（借出，全部借出）。
  - 否则 → `onshelf`（在馆）。

**状态枚举与显示文案**：

| 内部值 | 显示文案 | badge 色 |
| --- | --- | --- |
| `onshelf` | 在馆 | 绿色 |
| `lent` | 借出 | 蓝色 |
| `offshelf` | 下架 | 灰色 |

**分类枚举与显示文案**（表单下拉与筛选下拉同源，筛选增加"全部分类"空选项）：

`["文学", "科技", "历史", "艺术", "经济", "教育", "生活", "其他", "未分类"]`

---

## 4. API 设计

统一请求 envelope：`{ "action": "<action>", "data": { ... } }`。
统一响应：单对象 `{ "success": true, "data": {}, "error": "" }`；列表 `{ "success": true, "rows": [], "total": 0, "error": "" }`；失败 `{ "success": false, "data": {}, "error": "<用户可读信息>" }`。

**行对象（列表/详情返回）**：

```json
{
  "id": 1,
  "title": "三体",
  "isbn": "978-7-5366-9293-0",
  "author": "刘慈欣",
  "category": "科技",
  "publisher": "重庆出版社",
  "publish_year": 2008,
  "location": "A区-3排",
  "total_stock": 5,
  "borrowed_count": 2,
  "available_count": 3,
  "status": "onshelf",
  "updated_at": "2026-08-03 14:30:00"
}
```

| Action | 触发场景 | Request `data` | Response | 成功后前端状态 | 失败展示 | 权限模式 |
| --- | --- | --- | --- | --- | --- | --- |
| `list` | 首次进入、搜索、筛选、刷新、翻页 | `{ keyword?, category?, status?, page?, pageSize? }` | `{ success, rows, total, error }` | 更新 `rows`/`total`/`page`/`filters`；清空 `error` | 页面级错误占位 + 重试 | `read_default` |
| `detail` | 点击行"详情" | `{ id }` | `{ success, data, error }` | 更新 `detail`，打开详情面板 | 详情区错误提示 + 重试 | `read_default` |
| `create` | 新建表单保存 | `{ title, isbn, author, category, publisher, publish_year, location, total_stock }` | `{ success, data, error }` | 关闭弹窗、复位表单、回到第一页并刷新列表、toast 成功 | 字段红字 / toast 失败原因，保留表单 | `button_controlled` |
| `update` | 编辑表单保存 | `{ id, title, isbn, author, category, publisher, publish_year, location, total_stock }` | `{ success, data, error }` | 关闭弹窗、刷新列表与详情、toast 成功 | 字段红字 / toast 失败原因，保留表单 | `button_controlled` |
| `delete` | 删除确认弹窗确认 | `{ id }` | `{ success: true, error }` | 关闭弹窗、刷新列表、toast 成功 | toast"存在借出记录，请先归还"或失败原因 | `button_controlled` |
| `lend` | 借出数量弹窗确认 | `{ id, quantity }` | `{ success, data, error }` | 关闭弹窗、刷新列表与详情、toast 成功 | 字段红字"超出可借数量"/toast"该图书已下架，不能借出" | `button_controlled` |
| `return_book` | 归还数量弹窗确认 | `{ id, quantity }` | `{ success, data, error }` | 关闭弹窗、刷新列表与详情、toast 成功 | 字段红字"归还数量大于借出数量" | `button_controlled` |
| `offshelf` | 下架确认弹窗确认 | `{ id }` | `{ success, data, error }` | 关闭弹窗、刷新列表与详情、toast 成功 | toast 失败原因 | `button_controlled` |
| `onshelf` | 上架确认弹窗确认 | `{ id }` | `{ success, data, error }` | 关闭弹窗、刷新列表与详情、toast 成功 | toast 失败原因 | `button_controlled` |

**前端 API wrapper（`frontend/api.js`）与后端 dispatch 使用完全相同的 action 字符串**：`list`、`detail`、`create`、`update`、`delete`、`lend`、`return_book`、`offshelf`、`onshelf`。

**`manifest.actions`（仅按钮受控动作，JSON 字符串数组）**：

```json
["create", "update", "delete", "lend", "return_book", "offshelf", "onshelf"]
```

---

## 5. 状态流转与校验规则

### 5.1 前端状态变量

| 状态变量 | 类型 | 初始值 | 变化时机 | 影响 UI |
| --- | --- | --- | --- | --- |
| `rows` | Array | `[]` | `list` 成功后 | 表格行渲染 |
| `total` | number | 0 | `list` 成功后 | 分页"共 N 条" |
| `page` | number | 1 | 翻页、刷新、筛选、新建/编辑后回到 1 | 分页高亮、列表数据 |
| `pageSize` | number | 10 | 用户选择 | 分页、列表数据 |
| `filters` | Object `{ keyword, category, status }` | `{ keyword:'', category:'', status:'' }` | 搜索/筛选输入 | 工具栏值、列表数据 |
| `loading` | boolean | true | 初始加载/刷新时 true，加载结束 false | 加载占位、按钮 loading |
| `saving` | boolean | false | 表单/数量/确认弹窗提交时 true | 提交按钮 loading 禁点 |
| `error` | string | '' | 加载失败置文案，成功清空 | 错误占位 |
| `modalMode` | `'create'` / `'edit'` / `null` | null | 打开/关闭表单弹窗 | 表单弹窗 |
| `form` | Object | 默认空表单 | 打开表单、输入、保存成功 | 表单字段值 |
| `selectedId` | number/null | null | 选择行、打开详情/弹窗 | 行高亮、详情目标 |
| `detail` | Object/null | null | `detail` 成功后 | 详情面板 |
| `qtyModal` | Object/null | null | 打开借出/归还数量弹窗 | 数量弹窗 |
| `confirm` | Object/null | null | 打开删除/下架/上架确认 | 确认弹窗 |
| `toast` | Object/null | null | 操作成功/失败后置入，自动或手动清除 | toast 提示 |

### 5.2 业务状态流转

| 业务状态 | 显示文案 | 可执行操作 | 下一状态 | 禁止条件 |
| --- | --- | --- | --- | --- |
| `onshelf`（可借>0） | 在馆 | 借出、编辑、下架、删除、归还（若已借>0） | 借出（借出后可借=0 时） | 删除需 `borrowed_count==0` |
| `lent`（可借=0） | 借出 | 归还、编辑、下架、删除 | 在馆（归还后可借>0） | 借出（可借=0 超额）；删除需 `borrowed_count==0` |
| `offshelf` | 下架 | 上架、编辑、删除、归还（若已借>0） | 上架后回到在馆/借出（按可借数） | 借出；删除需 `borrowed_count==0` |

后端在每个变更动作后维护 `status`：

- `create`：`borrowed_count=0`，`status='onshelf'`。
- `lend(n)`：校验通过后 `borrowed_count+=n`；若 `available_count==0` 则 `status='lent'`，否则 `status='onshelf'`（下架状态下不允许借出）。
- `return_book(n)`：校验通过后 `borrowed_count-=n`；若 `status!='offshelf'`，则 `available_count>0` 时 `status='onshelf'`，否则 `status='lent'`。
- `offshelf`：`status='offshelf'`。
- `onshelf`：`available_count==0` 时 `status='lent'`，否则 `status='onshelf'`。
- `update`：`total_stock` 变更后重算；若 `available_count<0` 拦截；若 `status!='offshelf'`，按 `available_count` 重算 `status`。

### 5.3 字段校验规则

| 字段 | 前端校验 | 后端校验 | 错误文案 |
| --- | --- | --- | --- |
| 书名 | 必填，trim 后 1-120 字 | 必填，1-120 字 | "请输入书名" / "书名需在 1-120 字之间" |
| ISBN | 必填，去空格连字符后为 10 或 13 位且格式合法 | 必填，ISBN-10/ISBN-13 格式；全库唯一（排除自身） | "请输入 ISBN" / "ISBN 格式不正确" / "ISBN 已存在" |
| 作者 | 选填，≤80 字 | 选填，≤80 字 | "作者需在 80 字以内" |
| 分类 | 选填，须为枚举值 | 选填，须为枚举值，为空按"未分类" | "请选择有效的分类" |
| 出版社 | 选填，≤80 字 | 选填，≤80 字 | "出版社需在 80 字以内" |
| 出版年份 | 选填，整数 1000-2100 | 选填，整数 1000-2100 | "出版年份需在 1000-2100 之间" |
| 馆藏位置 | 选填，≤60 字 | 选填，≤60 字 | "馆藏位置需在 60 字以内" |
| 总库存 | 必填，整数 ≥1 | 必填，整数 ≥1；`>= borrowed_count` | "请输入总库存" / "总库存需为不小于 1 的整数" / "总库存不能小于已借出数量" |
| 借出数量 | 不可直接编辑（只读展示） | 整数 ≥0，恒 ≤ total_stock | - |
| 借出 quantity | 整数 ≥1，≤ available_count | 整数 ≥1；`quantity<=available_count`；`status!='offshelf'` | "请输入不小于 1 的整数" / "超出可借数量" / "该图书已下架，不能借出" |
| 归还 quantity | 整数 ≥1，≤ borrowed_count | 整数 ≥1；`quantity<=borrowed_count` | "请输入不小于 1 的整数" / "归还数量大于借出数量" |
| id（详情/删除/状态动作） | 必选 | 必须存在 | "记录不存在" |

---

## 6. 异常与空状态处理

| 场景 | 触发条件 | UI 展示 | 恢复动作 | 对应 action |
| --- | --- | --- | --- | --- |
| 首次加载中 | 进入页面加载列表 | 表格区 loading 占位 | 自动完成 | `list` |
| 无任何记录 | `list` 返回 `total==0` 且无筛选 | "暂无图书" + "新建图书"按钮 | 点击"新建图书" | `create`（弹窗） |
| 筛选无结果 | `list` 返回 `total==0` 且有筛选 | "没有符合条件的图书" + "清除筛选"按钮 | 点击"清除筛选"复位并重载 | `list` |
| 列表加载失败 | `list` 返回 `success:false` | 内容区错误提示 + "重试"按钮 | 点击"重试" | `list` |
| 详情读取失败 | `detail` 返回 `success:false` | 详情面板错误提示 + "重试" | 点击"重试" | `detail` |
| 表单校验失败 | 新建/编辑必填缺失或非法 | 字段下方红字提示 | 修正后重新提交 | `create` / `update` |
| 借出超额/下架拦截 | `lend` 前置校验失败或后端拦截 | 数量弹窗字段红字 / toast | 调整数量或取消 | `lend` |
| 归还超额 | `return_book` 前置校验失败 | 数量弹窗字段红字 | 调整数量或取消 | `return_book` |
| 删除被借出拦截 | `delete` 时 `borrowed_count>0` | toast"存在借出记录，请先归还" | 先归还再删除 | `delete` |
| 保存失败 | `create`/`update`/状态动作返回 `success:false` | toast 失败原因 | 重试或修正；保留当前输入/页面状态 | 对应动作 |
| 无权限 | 用户缺某受控按钮权限 | 对应按钮不渲染 | 只读操作仍可用 | - |

所有失败路径均保留当前页面/表单状态，可修正、清除筛选或重试，不出现死页面。

---

## 7. 实施步骤

按 `generated-app-builder` 顺序实施：

1. 编写 `manifest.json`：`id` 用 UUID `0d4c75ff-bf06-4a98-91c0-ca8c4e86072d`；声明 `dataModels.Book`、`queries`（`book_list`、`book_by_isbn`）、`actions` 数组。
2. 实现 `backend/models.go`：请求/响应 envelope、Book 行结构体、表单 payload、模型名/查询名/状态枚举常量。
3. 实现 `backend/platform.go`：封装宿主数据能力调用。
4. 实现 `backend/validators.go`：必填、ISBN 格式、枚举、数量、总库存、唯一、数量关系校验。
5. 实现 `backend/book_handlers.go`：`list`/`detail`/`create`/`update`/`delete`/`lend`/`return_book`/`offshelf`/`onshelf` 九大 handler，并在 `main.go` 中建立 dispatch。
6. 实现 `frontend/api.js`：9 个 wrapper，action 字符串与后端一致。
7. 实现 `frontend/state.js`：state 结构、默认值、常量、行归一化。
8. 实现 `frontend/ui.js`：header/toolbar/table/详情/空/加载/错误渲染与事件委托。
9. 实现 `frontend/modal.js`：表单/数量/确认弹窗与 toast 助手。
10. 实现 `frontend/styles.js`：`injectStyles(root)`，按第 13 节作用域类名。
11. 构建 `backend.wasm`：`cd backend && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../backend.wasm .`，并确认 `backend.wasm` 比所有 `*.go` 新。
12. 运行第 14 节自检。

---

## 8. 代码目录与文件清单（对接 generated-app-builder）

```text
generated_apps/0d4c75ff-bf06-4a98-91c0-ca8c4e86072d/
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
    book_handlers.go
    validators.go
  backend.wasm
```

| 文件 | 必须生成 | 主要内容 |
| --- | --- | --- |
| `manifest.json` | 是 | id/name/description/export/frontendFile/backendSource/backendModule/dataModels/queries/actions |
| `frontend.js` | 是 | `render(container, context)`、`dispose(container)`、权限 `can` 包装、事件委托、初始化加载 |
| `frontend/api.js` | 是 | 9 个 action wrapper（`listBooks/getBook/createBook/updateBook/deleteBook/lendBook/returnBook/offshelfBook/onshelfBook`） |
| `frontend/state.js` | 是 | state 默认值、分类/状态常量、`normalizeRow`、`formToPayload` |
| `frontend/ui.js` | 是 | shell/toolbar/table/行操作/详情/空/加载/错误渲染与事件绑定 |
| `frontend/modal.js` | 是 | 表单弹窗、数量弹窗、确认弹窗、toast |
| `frontend/styles.js` | 是 | `injectStyles(root)`，`ga-book-` 前缀作用域样式 |
| `backend/main.go` | 是 | WASI 入口、`handle`、请求解码、action 分发 |
| `backend/platform.go` | 是 | `data_list/get/create/update/delete/run_query` 宿主封装 |
| `backend/models.go` | 是 | envelope、行结构体、模型/查询/状态常量 |
| `backend/book_handlers.go` | 是 | 9 个业务 handler |
| `backend/validators.go` | 是 | 校验助手 |
| `backend.wasm` | 是 | 后端编译产物，构建后生成 |

---

## 9. Manifest 与表结构约定

- `manifest.id` = `0d4c75ff-bf06-4a98-91c0-ca8c4e86072d`（与目录同名）。
- 不设计物理表名；平台从 `dataModels` 生成表。隐式平台字段 `id`/`uuid`/`created_at`/`updated_at` 不声明。
- `manifest.actions` 为 JSON 字符串数组：`["create","update","delete","lend","return_book","offshelf","onshelf"]`。
- 单一实体，无需 `relations`。

### 9.1 dataModels

| 模型 | manifest `dataModels[].name` | 字段 | 校验 | 索引建议 |
| --- | --- | --- | --- | --- |
| 图书 | `Book` | `title` string | required, maxLength 120 | 是（搜索关键词，配合查询 contains） |
| 图书 | `Book` | `isbn` string | required, maxLength 32 | 是（唯一性查询） |
| 图书 | `Book` | `author` string | maxLength 80 | 是（搜索关键词） |
| 图书 | `Book` | `category` enum（`["文学","科技","历史","艺术","经济","教育","生活","其他","未分类"]`） | 可选 | 是（分类筛选） |
| 图书 | `Book` | `publisher` string | maxLength 80 | - |
| 图书 | `Book` | `publish_year` int | min 1000, max 2100，可选 | - |
| 图书 | `Book` | `location` string | maxLength 60 | - |
| 图书 | `Book` | `total_stock` int | required, min 1 | - |
| 图书 | `Book` | `borrowed_count` int | min 0，默认 0 | - |
| 图书 | `Book` | `status` enum（`["onshelf","lent","offshelf"]`） | required，默认 `onshelf` | 是（状态筛选） |

### 9.2 queries

| Query name | From | Joins | Select | Filters | Sorting | Limit | Called by actions |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `book_list` | `Book` | 无 | 全部字段 | `keyword`（title/isbn/author contains，任一命中）、`category`（eq）、`status`（eq） | `updated_at` desc | `pageSize`（默认 10，上限 100） | `list` |
| `book_by_isbn` | `Book` | 无 | `id`, `isbn` | `isbn`（eq） | 无 | 1 | `create`、`update`（唯一性校验） |

`data_run_query` 返回 `{ "rows": [...], "total": N }`；`detail` 走 `data_get('Book', id)`。

---

## 10. 后端 Action 清单

| Action | Handler 文件 | Handler 函数 | Request | Response | Data capability 调用 | 校验 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `list` | `book_handlers.go` | `handleList` | `{ keyword?, category?, status?, page?, pageSize? }` | `{ success, rows, total, error }`，行含 `available_count` | `dataRunQuery('book_list', {...})` | `page>=1`；`pageSize` 1-100；`category`/`status` 若传须为枚举值；`keyword` trim |
| `detail` | `book_handlers.go` | `handleDetail` | `{ id }` | `{ success, data, error }`，含 `available_count` | `dataGet('Book', id)` | `id` 必填且存在 |
| `create` | `book_handlers.go` | `handleCreate` | `{ title, isbn, author, category, publisher, publish_year, location, total_stock }` | `{ success, data, error }` | `dataRunQuery('book_by_isbn', { isbn })` 查重；`dataCreate('Book', {...borrowed_count:0, status:'onshelf'})` | 书名/ISBN/总库存必填；ISBN 格式与唯一；分类枚举；出版年份范围；总库存≥1 |
| `update` | `book_handlers.go` | `handleUpdate` | `{ id, title, isbn, author, category, publisher, publish_year, location, total_stock }` | `{ success, data, error }` | `dataRunQuery('book_by_isbn', { isbn })` 查重（排除自身）；`dataGet('Book', id)`；`dataUpdate('Book', id, {...})` | 同 create；`total_stock>=borrowed_count`；`id` 存在 |
| `delete` | `book_handlers.go` | `handleDelete` | `{ id }` | `{ success: true, error }` | `dataGet('Book', id)`；`dataDelete('Book', id)` | `id` 存在；`borrowed_count==0`，否则报"存在借出记录，请先归还" |
| `lend` | `book_handlers.go` | `handleLend` | `{ id, quantity }` | `{ success, data, error }` | `dataGet('Book', id)`；`dataUpdate('Book', id, { borrowed_count, status })` | `id` 存在；`quantity` 整数≥1；`status!='offshelf'`；`quantity<=available_count` |
| `return_book` | `book_handlers.go` | `handleReturnBook` | `{ id, quantity }` | `{ success, data, error }` | `dataGet('Book', id)`；`dataUpdate('Book', id, { borrowed_count, status })` | `id` 存在；`quantity` 整数≥1；`quantity<=borrowed_count` |
| `offshelf` | `book_handlers.go` | `handleOffshelf` | `{ id }` | `{ success, data, error }` | `dataGet('Book', id)`；`dataUpdate('Book', id, { status:'offshelf' })` | `id` 存在；`status!='offshelf'`，否则报"已是下架状态" |
| `onshelf` | `book_handlers.go` | `handleOnshelf` | `{ id }` | `{ success, data, error }` | `dataGet('Book', id)`；`dataUpdate('Book', id, { status })`（按可借数重算） | `id` 存在；`status=='offshelf'`，否则报"非下架状态，无需上架" |

所有 handler 由 `main.go` 的 action 字符串分发调用；dispatch 中 action 字符串与 `frontend/api.js`、`manifest.actions` 完全一致。校验失败返回 `success:false` + 用户可读 `error`，不返回权限错误（权限由宿主在 WASM 前校验）。

---

## 11. 前端模块拆分

| 文件 | 关键导出 | 依赖 | 被谁调用 |
| --- | --- | --- | --- |
| `frontend.js` | `render(container, context)`、`dispose(container)`、内部 `can(action)`、`loadList()`、事件委托 `onAction` | `frontend/api.js`、`frontend/state.js`、`frontend/ui.js`、`frontend/modal.js`、`frontend/styles.js` | 宿主运行时 |
| `frontend/api.js` | `listBooks`、`getBook`、`createBook`、`updateBook`、`deleteBook`、`lendBook`、`returnBook`、`offshelfBook`、`onshelfBook` | 无（内部调用 `context.invokeData`） | `frontend.js`、`frontend/ui.js`、`frontend/modal.js` |
| `frontend/state.js` | `createInitialState()`、`CATEGORY_OPTIONS`、`STATUS_OPTIONS`、`statusLabel()`、`normalizeRow()`、`formToPayload()`、`defaultForm()` | 无 | `frontend.js`、`frontend/ui.js`、`frontend/modal.js` |
| `frontend/ui.js` | `renderShell()`、`renderToolbar()`、`renderTable()`、`renderRowActions()`、`renderDetail()`、`renderEmpty()`、`renderError()`、`renderLoading()` | `frontend/state.js`、`frontend/api.js` | `frontend.js` |
| `frontend/modal.js` | `openBookForm()`、`openQuantityModal()`、`openConfirm()`、`showToast()`、`closeModal()` | `frontend/state.js`、`frontend/api.js` | `frontend.js`、`frontend/ui.js` |
| `frontend/styles.js` | `injectStyles(root)` | 无 | `frontend.js` |

- `frontend.js` 中所有命名导入必须在对应模块中存在同名导出。
- `frontend/styles.js` 仅导出 `injectStyles`。
- `frontend/api.js` 的 wrapper 直接返回 `{ success, data, error }` / `{ success, rows, total, error }`，不包裹额外 `response` 字段。

---

## 12. 前端交互流程与状态合同

### 12.1 交互流程

| 操作 | 触发控件 | 前端状态变化 | 调用 action | 成功反馈 | 失败反馈 |
| --- | --- | --- | --- | --- | --- |
| 初始加载 | 页面 `render` | `loading=true` → 调 list → `rows/total/page` 更新 → `loading=false` | `list` | 渲染表格 | 渲染错误占位 |
| 新建 | header"新建图书" | `modalMode='create'`、`form=defaultForm()` | - | 打开表单弹窗 | - |
| 保存新建 | 表单"保存" | `saving=true` → 调 create → `modalMode=null`、`form=reset`、`page=1`、重载 list → `saving=false` | `create` | 关闭弹窗、toast 成功、列表出现新记录 | 字段红字或 toast，保留表单 |
| 编辑 | 行"编辑" | `modalMode='edit'`、`form` 回填、`selectedId` | - | 打开表单弹窗 | - |
| 保存编辑 | 表单"保存" | `saving=true` → 调 update → 关闭弹窗、重载 list/detail → `saving=false` | `update` | toast 成功、列表刷新 | 字段红字或 toast，保留表单 |
| 查看详情 | 行"详情" | `selectedId=id`、打开详情面板 loading → 调 detail → `detail` 更新 | `detail` | 展示详情面板 | 详情区错误 + 重试 |
| 借出 | 行"详情" 或详情"借出" | 若下架 → toast 拦截；否则 `qtyModal={type:'lend', id, quantity:1}` | - | 打开数量弹窗 | toast 拦截 |
| 确认借出 | 数量弹窗"确认" | `saving=true` → 调 lend → 关闭弹窗、重载 list/detail → `saving=false` | `lend` | toast 成功、可借数量减少、状态更新 | 字段红字"超出可借数量"或 toast |
| 归还 | 行"详情" 或详情"归还" | `qtyModal={type:'return', id, quantity:1}` | - | 打开数量弹窗 | - |
| 确认归还 | 数量弹窗"确认" | `saving=true` → 调 return_book → 关闭弹窗、重载 list/detail → `saving=false` | `return_book` | toast 成功、可借数量恢复 | 字段红字"归还数量大于借出数量" |
| 下架 | 行"详情" 或详情"下架" | `confirm={type:'offshelf', id}` | - | 打开确认弹窗 | - |
| 确认下架 | 确认弹窗"确认" | `saving=true` → 调 offshelf → 关闭、重载 → `saving=false` | `offshelf` | toast 成功、状态变下架 | toast 失败原因 |
| 上架 | 行"详情" 或详情"上架" | `confirm={type:'onshelf', id}` | - | 打开确认弹窗 | - |
| 确认上架 | 确认弹窗"确认" | `saving=true` → 调 onshelf → 关闭、重载 → `saving=false` | `onshelf` | toast 成功、状态恢复 | toast 失败原因 |
| 删除 | 行"详情" 或详情"删除" | `confirm={type:'delete', id}` | - | 打开确认弹窗 | - |
| 确认删除 | 确认弹窗"确认" | `saving=true` → 调 delete → 关闭、重载 → `saving=false` | `delete` | toast 成功、记录移除 | toast"存在借出记录，请先归还"或失败原因 |
| 搜索 | 工具栏搜索框 | `filters.keyword` 更新、`page=1`、`loading=true` → 调 list | `list` | 渲染结果 | 空结果占位/错误占位 |
| 分类筛选 | 工具栏分类下拉 | `filters.category` 更新、`page=1` → 调 list | `list` | 渲染结果 | 空结果占位/错误占位 |
| 状态筛选 | 工具栏状态下拉 | `filters.status` 更新、`page=1` → 调 list | `list` | 渲染结果 | 空结果占位/错误占位 |
| 清除筛选 | 工具栏"清除筛选" | `filters` 复位、`page=1` → 调 list | `list` | 渲染全部 | 错误占位 |
| 刷新 | 工具栏"刷新" | `page=1`、`loading=true` → 调 list | `list` | 渲染结果 | 错误占位 |
| 翻页 | 分页上一页/下一页 | `page` 变更 → 调 list | `list` | 渲染对应页 | 错误占位 |
| 空状态恢复 | 空态"新建图书" | `modalMode='create'` | - | 打开表单 | - |
| 错误恢复 | 错误占位"重试" | `loading=true` → 调 list | `list` | 渲染结果 | 保留错误占位 |

### 12.2 动作对齐表

| UI 操作 | DOM 控件/事件 | API wrapper | Backend action | 成功后刷新 |
| --- | --- | --- | --- | --- |
| 初始加载/搜索/筛选/刷新/翻页 | toolbar 输入/下拉/按钮、分页、data-action=`list` | `listBooks` | `list` | 重绘表格/分页 |
| 打开详情 | 行按钮 data-action=`detail` | `getBook` | `detail` | 重绘详情面板 |
| 保存新建 | 表单 data-action=`book-save`（mode=create） | `createBook` | `create` | 重载列表第一页 |
| 保存编辑 | 表单 data-action=`book-save`（mode=edit） | `updateBook` | `update` | 重载列表/详情 |
| 确认删除 | 确认弹窗 data-action=`confirm-delete` | `deleteBook` | `delete` | 重载列表 |
| 确认借出 | 数量弹窗 data-action=`confirm-lend` | `lendBook` | `lend` | 重载列表/详情 |
| 确认归还 | 数量弹窗 data-action=`confirm-return` | `returnBook` | `return_book` | 重载列表/详情 |
| 确认下架 | 确认弹窗 data-action=`confirm-offshelf` | `offshelfBook` | `offshelf` | 重载列表/详情 |
| 确认上架 | 确认弹窗 data-action=`confirm-onshelf` | `onshelfBook` | `onshelf` | 重载列表/详情 |

每个受控 UI 操作的事件处理器在调用 API 前再次执行 `can(actionKey)` 检查（`create`→`can('create')`、`update`→`can('update')`、`delete`→`can('delete')`、借出/归还/下架/上架同理），防止绕过渲染层直接触发。

---

## 13. 样式规范与视觉一致性

采用标准 func-operation 生成应用样式（`style-guide.md`）。应用根类 `ga-book-root`，所有选择器带 `ga-book-` 前缀，不定义全局 `button/input/table/.modal` 样式。设计令牌使用 style-guide 的 CSS 变量（`--ga-*`）。

| UI 元素 | class 命名 | 样式要求 | 状态 |
| --- | --- | --- | --- |
| 根容器 | `ga-book-root` | `width:100%`、`min-height:100%`、`box-sizing:border-box`、`--ga-*` 变量定义 | - |
| 头部 | `ga-book-header` | 标题 20-22px、描述 13-14px muted、主操作右对齐；<720px 纵向堆叠 | - |
| 工具栏 | `ga-book-toolbar` | flex wrap；搜索框 min-width 200px；下拉 select 统一输入样式 | - |
| 主按钮 | `ga-book-btn ga-book-btn-primary` | 蓝底白字，8px/14px padding，6px radius，禁用降透明度 | default/hover/disabled |
| 次按钮 | `ga-book-btn ga-book-btn-secondary` | 白底灰边 | default/hover/disabled |
| 危险按钮 | `ga-book-btn ga-book-btn-danger` | 红字、软红 hover 背景 | default/hover |
| 小按钮 | `ga-book-btn ga-book-btn-sm` | 4px/10px padding、13px | - |
| 输入框/下拉 | `ga-book-input` | 8px/12px padding、1px 灰边、6px radius、聚焦蓝边 + 浅环；非法红边 | default/focus/invalid |
| 表格容器 | `ga-book-table-wrap` | 1px 边、8px radius、横向 overflow | - |
| 表格 | `ga-book-table` | 表头 `#f9fafb`、muted 13px 600；正文 14px；行 hover `#f9fafb`；操作列右对齐 | header/row/hover |
| 状态标签 | `ga-book-badge` | 2px/8px padding、12px 500、10px radius；绿=在馆、蓝=借出、灰=下架 | - |
| 分页 | `ga-book-pagination` | 次级小按钮 + "共 N 条" muted | - |
| 弹窗遮罩 | `ga-book-modal-mask` | `position:absolute; inset:0`（挂载于 `ga-book-root` 内，非 fixed） | - |
| 弹窗面板 | `ga-book-modal` | 居中、宽 420-560px、`max-width:calc(100vw - 32px)`、白底 8px radius、popover 阴影、头/体/脚三段 | - |
| 表单分组 | `ga-book-form-group` | 字段标签 + 输入 + 字段错误红字 `ga-book-field-error` | - |
| Toast | `ga-book-toast` | 顶部右侧、白/状态色底、8px radius、13-14px；成功自动消失，失败保留 | success/error |
| 加载占位 | `ga-book-loading` | 内容区居中 loading 文本/骨架 | - |
| 空/空结果 | `ga-book-empty` | 居中 muted 文案 + 引导按钮 | - |
| 错误块 | `ga-book-error` | 居中 danger 文案 + "重试"按钮 | - |

动态节点（弹窗、toast、确认框）统一 `appendChild` 到 `ga-book-root` 内，确保作用域样式生效；`modal.js` 与 toast 使用的所有 class 均在 `injectStyles` 中覆盖。

---

## 14. 验收与自检清单

- 前端渲染真实控件（表格、表单、弹窗、按钮），不是文字摘要。
- 初始加载调用 `list` 并渲染表格；空态、加载、错误态可见。
- 新建/编辑表单在调后端前完成必填与格式校验并显示字段红字。
- 后端实现全部前端 action 字符串：`list/detail/create/update/delete/lend/return_book/offshelf/onshelf`。
- `manifest.dataModels.Book`、`queries.book_list`、`queries.book_by_isbn` 与后端数据能力调用一致。
- 借出/归还数量弹窗在调后端前校验数量上限；下架图书借出被前端拦截且后端兜底。
- 删除被借出图书被后端拦截（`borrowed_count>0`），归还后可删除。
- 搜索/筛选无结果显示"清除筛选"；清除后恢复全量。
- 受控按钮（新建/编辑/删除/借出/归还/下架/上架）在 `can(actionKey)==false` 时不渲染，事件处理器二次校验权限。
- `manifest.actions` 仅含 7 个受控动作；`list`/`detail` 不在其中。
- <720px 下 header 堆叠、工具栏全宽、表格横向滚动、弹窗 `calc(100vw - 32px)`。
- CSS 全部作用域于 `ga-book-root`，无全局选择器泄漏；modal/toast 类均有对应样式。

### 交接核对表

| 检查项 | 通过标准 |
| --- | --- |
| Section 4 actions 匹配 Section 10 后端 actions | 9 个 action 名逐一对应 |
| Section 10 actions 匹配 Section 11 `frontend/api.js` wrappers | 每个后端 action 有且仅有一个同名 wrapper |
| Section 9 模型/查询名匹配后端常量 | `Book`、`book_list`、`book_by_isbn` 一致 |
| Section 13 类名匹配 Section 11 组件 | 组件使用 `ga-book-*` 类均有样式 |
| 每个产品操作具备 UI/API/后端/校验/成功/失败处理 | 新增/编辑/删除/借出/归还/下架/上架/搜索/筛选/详情/刷新/分页全覆盖 |
| 无面向用户的生成应用内部实现细节暴露 | 文案全部为业务中文 |
| 无占位内容 | 无 `<business>`/`待生成`/`TODO` 等残留 |
