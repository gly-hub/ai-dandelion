import { escapeHTML } from './dom.js'
import { STATUS_OPTIONS, EDUCATION_OPTIONS, statusLabel, statusClass, educationLabel } from './state.js'

export function renderShell(container, state) {
  container.innerHTML = [
    '<div class="ga-teacher-root" id="ga-teacher-root">',
    '  <div class="ga-teacher-view">',
    '    <div class="ga-teacher-header" id="ga-teacher-header"></div>',
    '    <div class="ga-teacher-toolbar" id="ga-teacher-toolbar"></div>',
    '    <div class="ga-teacher-content" id="ga-teacher-content"></div>',
    '    <div class="ga-teacher-pagination" id="ga-teacher-pagination"></div>',
    '  </div>',
    '  <div class="ga-teacher-toast-container"></div>',
    '</div>',
  ].join('')
}

export function renderHeader(root, state, can) {
  var el = root.querySelector('#ga-teacher-header')
  if (!el) return
  var newBtn = (can && can('teacher_create'))
    ? '<button class="ga-teacher-btn ga-teacher-btn-primary" data-action="open-create">+ 新建教师</button>'
    : ''
  el.innerHTML =
    '<div class="ga-teacher-header-text">' +
      '<h1 class="ga-teacher-title">教师管理</h1>' +
      '<p class="ga-teacher-desc">维护教师档案与在职 / 停用 / 离职状态</p>' +
    '</div>' +
    '<div class="ga-teacher-header-actions">' + newBtn + '</div>'
}

function optionList(values, current, placeholder) {
  var html = '<option value="">' + escapeHTML(placeholder) + '</option>'
  ;(values || []).forEach(function (v) {
    html += '<option value="' + escapeHTML(v) + '"' + (v === current ? ' selected' : '') + '>' + escapeHTML(v) + '</option>'
  })
  return html
}

export function renderToolbar(root, state) {
  var el = root.querySelector('#ga-teacher-toolbar')
  if (!el) return
  var statusOptions = ['<option value="">全部状态</option>'].concat(
    STATUS_OPTIONS.map(function (s) {
      return '<option value="' + s.value + '"' + (state.statusFilter === s.value ? ' selected' : '') + '>' + s.label + '</option>'
    })
  ).join('')
  var educationOptions = ['<option value="">全部学历</option>'].concat(
    EDUCATION_OPTIONS.map(function (s) {
      return '<option value="' + s.value + '"' + (state.educationFilter === s.value ? ' selected' : '') + '>' + s.label + '</option>'
    })
  ).join('')
  el.innerHTML = [
    '<input class="ga-teacher-search" type="text" placeholder="搜索姓名 / 工号 / 联系电话" value="' + escapeHTML(state.keyword) + '" data-action="search-input" />',
    '<select class="ga-teacher-select" data-action="filter-department">' + optionList(state.departmentOptions, state.departmentFilter, '全部院系') + '</select>',
    countryFilterHtml(state),
    '<select class="ga-teacher-select" data-action="filter-status">' + statusOptions + '</select>',
    '<select class="ga-teacher-select" data-action="filter-education">' + educationOptions + '</select>',
    '<select class="ga-teacher-select" data-action="filter-title">' + optionList(state.titleOptions, state.titleFilter, '全部职称') + '</select>',
    '<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="reload">刷新</button>',
    '<button class="ga-teacher-btn ga-teacher-btn-clear" data-action="clear-filters">清除筛选</button>',
  ].join('')
}

// countryFilterHtml 渲染筛选栏国籍下拉；选项来自全局配置键 country。
function countryFilterHtml(state) {
  if (state.countryOptionsError) {
    return '<div class="ga-teacher-country-control">' +
      '<select class="ga-teacher-select" data-action="filter-country" disabled><option value="">全部国籍</option></select>' +
      '<span class="ga-teacher-options-error">选项加载失败</span>' +
      '<button class="ga-teacher-btn ga-teacher-btn-small" data-action="country-retry" type="button">重试</button>' +
    '</div>'
  }
  var opts = ['<option value="">全部国籍</option>'].concat(
    (state.countryOptions || []).map(function (o) {
      var v = o && o.value
      var label = (o && o.label) || v
      return '<option value="' + escapeHTML(v) + '"' + (state.countryFilter === v ? ' selected' : '') + '>' + escapeHTML(label) + '</option>'
    })
  ).join('')
  return '<select class="ga-teacher-select" data-action="filter-country"' + (state.countryOptionsLoading ? ' disabled' : '') + '>' + opts + '</select>'
}

export function renderTable(root, state, can) {
  var el = root.querySelector('#ga-teacher-content')
  if (!el) return

  if (state.loading) {
    renderLoadingState(el)
    return
  }
  if (state.error) {
    renderErrorState(el)
    return
  }
  if (!state.rows || state.rows.length === 0) {
    var hasFilter = state.keyword || state.countryFilter || state.departmentFilter || state.statusFilter || state.educationFilter || state.titleFilter
    if (hasFilter) {
      renderSearchEmptyState(el)
    } else {
      renderEmptyState(el, can)
    }
    return
  }

  var rowsHtml = state.rows.map(function (row) {
    return '<tr>' +
      '<td>' + escapeHTML(row.name || '') + '</td>' +
      '<td>' + escapeHTML(row.employee_no || '') + '</td>' +
      '<td>' + escapeHTML(row.department || '') + '</td>' +
      '<td>' + escapeHTML(row.title || '') + '</td>' +
      '<td>' + escapeHTML(row.phone || '') + '</td>' +
      '<td>' + escapeHTML(row.email || '') + '</td>' +
      '<td><span class="ga-teacher-badge ' + statusClass(row.status) + '">' + escapeHTML(statusLabel(row.status)) + '</span></td>' +
      '<td>' + escapeHTML(row.hire_date || '') + '</td>' +
      '<td>' + escapeHTML(formatDate(row.updated_at)) + '</td>' +
      '<td class="ga-teacher-actions">' + renderRowActions(row, can) + '</td>' +
    '</tr>'
  }).join('')

  el.innerHTML = [
    '<div class="ga-teacher-table-wrapper">',
    '  <table class="ga-teacher-table">',
    '    <thead><tr>',
    '      <th>姓名</th><th>工号</th><th>所属院系</th><th>职称</th><th>联系电话</th><th>电子邮箱</th><th>状态</th><th>入职日期</th><th>最近更新时间</th><th class="ga-teacher-actions-th">操作</th>',
    '    </tr></thead>',
    '    <tbody>' + rowsHtml + '</tbody>',
    '  </table>',
    '</div>',
  ].join('')
}

// renderRowActions 按权限与状态渲染行操作按钮。
function renderRowActions(row, can) {
  var parts = []
  var id = row.id
  var status = row.status
  parts.push('<button class="ga-teacher-btn ga-teacher-btn-small" data-action="detail" data-id="' + id + '">详情</button>')
  if (can && can('teacher_update')) {
    parts.push('<button class="ga-teacher-btn ga-teacher-btn-small" data-action="edit" data-id="' + id + '">编辑</button>')
  }
  if (can && can('teacher_change_status')) {
    if (status === 'active') {
      parts.push('<button class="ga-teacher-btn ga-teacher-btn-small" data-action="change-status" data-id="' + id + '" data-status="suspended">停用</button>')
      parts.push('<button class="ga-teacher-btn ga-teacher-btn-small" data-action="change-status" data-id="' + id + '" data-status="resigned">离职</button>')
    } else if (status === 'suspended') {
      parts.push('<button class="ga-teacher-btn ga-teacher-btn-small" data-action="change-status" data-id="' + id + '" data-status="active">在职</button>')
      parts.push('<button class="ga-teacher-btn ga-teacher-btn-small" data-action="change-status" data-id="' + id + '" data-status="resigned">离职</button>')
    }
  }
  if (can && can('teacher_delete')) {
    parts.push('<button class="ga-teacher-btn ga-teacher-btn-small-danger" data-action="delete" data-id="' + id + '">删除</button>')
  }
  return parts.join('')
}

export function renderPagination(root, state) {
  var el = root.querySelector('#ga-teacher-pagination')
  if (!el) return
  if (state.total <= state.pageSize) {
    el.innerHTML = ''
    return
  }
  var totalPages = Math.ceil(state.total / state.pageSize) || 1
  var pages = []
  var start = Math.max(1, state.page - 2)
  var end = Math.min(totalPages, state.page + 2)
  if (start > 1) {
    pages.push('<button class="ga-teacher-page-btn" data-action="page" data-page="1">1</button>')
    if (start > 2) pages.push('<span class="ga-teacher-page-ellipsis">...</span>')
  }
  for (var i = start; i <= end; i++) {
    pages.push('<button class="ga-teacher-page-btn' + (i === state.page ? ' ga-teacher-page-active' : '') + '" data-action="page" data-page="' + i + '">' + i + '</button>')
  }
  if (end < totalPages) {
    if (end < totalPages - 1) pages.push('<span class="ga-teacher-page-ellipsis">...</span>')
    pages.push('<button class="ga-teacher-page-btn" data-action="page" data-page="' + totalPages + '">' + totalPages + '</button>')
  }
  el.innerHTML =
    '<span class="ga-teacher-page-info">共 ' + state.total + ' 条</span>' +
    '<div class="ga-teacher-page-btns">' + pages.join('') + '</div>'
}

function renderEmptyState(el, can) {
  var newBtn = (can && can('teacher_create'))
    ? '<button class="ga-teacher-btn ga-teacher-btn-primary" data-action="open-create" style="margin-top:12px;">新建教师</button>'
    : ''
  el.innerHTML = [
    '<div class="ga-teacher-empty">',
    '  <p>暂无教师，点击"新建教师"录入第一位教师档案</p>',
    newBtn,
    '</div>',
  ].join('')
}

function renderSearchEmptyState(el) {
  el.innerHTML = [
    '<div class="ga-teacher-empty">',
    '  <p>没有符合条件的教师</p>',
    '  <button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="clear-filters" style="margin-top:8px;">清除筛选</button>',
    '</div>',
  ].join('')
}

function renderErrorState(el) {
  el.innerHTML = [
    '<div class="ga-teacher-error">',
    '  <p>加载失败，请重试</p>',
    '  <button class="ga-teacher-btn ga-teacher-btn-primary" data-action="reload" style="margin-top:8px;">重试</button>',
    '</div>',
  ].join('')
}

function renderLoadingState(el) {
  var skeleton = ''
  for (var i = 0; i < 5; i++) {
    skeleton += '<div class="ga-teacher-skeleton-row"><div class="ga-teacher-skeleton" style="width:' + (55 + i * 7 % 30) + '%"></div></div>'
  }
  el.innerHTML = '<div class="ga-teacher-loading">' + skeleton + '</div>'
}

export function findRow(rows, id) {
  if (!rows || id === null || id === undefined) return null
  for (var i = 0; i < rows.length; i++) {
    if (String(rows[i].id) === String(id)) return rows[i]
  }
  return null
}

// formatDate 兼容 unix 微秒时间戳与日期字符串。
export function formatDate(value) {
  if (value === null || value === undefined || value === '') return '-'
  var ms
  if (typeof value === 'number' || /^\d+$/.test(String(value))) {
    var n = Number(value)
    if (n > 1e14) ms = Math.floor(n / 1000)
    else if (n > 1e11) ms = n
    else ms = n * 1000
  } else {
    var parsed = new Date(value)
    if (isNaN(parsed.getTime())) return String(value)
    return formatDateObj(parsed)
  }
  var dt = new Date(ms)
  if (isNaN(dt.getTime())) return String(value)
  return formatDateObj(dt)
}

function formatDateObj(dt) {
  var yy = dt.getFullYear()
  var mm = String(dt.getMonth() + 1).padStart(2, '0')
  var dd = String(dt.getDate()).padStart(2, '0')
  var hh = String(dt.getHours()).padStart(2, '0')
  var mi = String(dt.getMinutes()).padStart(2, '0')
  return yy + '-' + mm + '-' + dd + ' ' + hh + ':' + mi
}
