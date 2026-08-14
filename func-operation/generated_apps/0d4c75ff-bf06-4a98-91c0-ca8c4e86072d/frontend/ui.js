import { escapeHTML } from './dom.js'
import { CATEGORY_OPTIONS, STATUS_OPTIONS, statusLabel, statusClass } from './state.js'

export function renderShell(container, state) {
  container.innerHTML = [
    '<div class="ga-book-root" id="ga-book-root">',
    '  <div class="ga-book-view">',
    '    <div class="ga-book-header" id="ga-book-header"></div>',
    '    <div class="ga-book-toolbar" id="ga-book-toolbar"></div>',
    '    <div class="ga-book-content" id="ga-book-content"></div>',
    '    <div class="ga-book-pagination" id="ga-book-pagination"></div>',
    '  </div>',
    '  <div class="ga-book-toast-container"></div>',
    '</div>',
  ].join('')
}

export function renderHeader(root, state, can) {
  var el = root.querySelector('#ga-book-header')
  if (!el) return
  var newBtn = (can && can('create'))
    ? '<button class="ga-book-btn ga-book-btn-primary" data-action="open-create">+ 新建图书</button>'
    : ''
  el.innerHTML =
    '<div class="ga-book-header-text">' +
      '<h1 class="ga-book-title">图书管理</h1>' +
      '<p class="ga-book-desc">维护图书档案与库存，跟踪借出 / 归还 / 下架状态流转</p>' +
    '</div>' +
    '<div class="ga-book-header-actions">' + newBtn + '</div>'
}

export function renderToolbar(root, state) {
  var el = root.querySelector('#ga-book-toolbar')
  if (!el) return
  var categoryOptions = ['<option value="">全部分类</option>'].concat(
    CATEGORY_OPTIONS.map(function (c) {
      return '<option value="' + escapeHTML(c) + '"' + (state.categoryFilter === c ? ' selected' : '') + '>' + escapeHTML(c) + '</option>'
    })
  ).join('')
  var statusOptions = ['<option value="">全部状态</option>'].concat(
    STATUS_OPTIONS.map(function (s) {
      return '<option value="' + s.value + '"' + (state.statusFilter === s.value ? ' selected' : '') + '>' + s.label + '</option>'
    })
  ).join('')
  el.innerHTML = [
    '<input class="ga-book-search" type="text" placeholder="搜索书名 / ISBN / 作者" value="' + escapeHTML(state.keyword) + '" data-action="search-input" />',
    '<select class="ga-book-select" data-action="filter-category">' + categoryOptions + '</select>',
    '<select class="ga-book-select" data-action="filter-status">' + statusOptions + '</select>',
    '<button class="ga-book-btn ga-book-btn-secondary" data-action="reload">刷新</button>',
    '<button class="ga-book-btn ga-book-btn-clear" data-action="clear-filters">清除筛选</button>',
  ].join('')
}

export function renderTable(root, state, can) {
  var el = root.querySelector('#ga-book-content')
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
    var hasFilter = state.keyword || state.categoryFilter || state.statusFilter
    if (hasFilter) {
      renderSearchEmptyState(el)
    } else {
      renderEmptyState(el, can)
    }
    return
  }

  var rowsHtml = state.rows.map(function (row) {
    return '<tr>' +
      '<td class="ga-book-cell-title">' + escapeHTML(row.title || '') + '</td>' +
      '<td>' + escapeHTML(row.isbn || '') + '</td>' +
      '<td>' + escapeHTML(row.author || '') + '</td>' +
      '<td>' + escapeHTML(row.category || '未分类') + '</td>' +
      '<td>' + escapeHTML(row.location || '') + '</td>' +
      '<td>' + escapeHTML(String(row.total_stock != null ? row.total_stock : '')) + '</td>' +
      '<td>' + escapeHTML(String(row.available_count != null ? row.available_count : '')) + '</td>' +
      '<td><span class="ga-book-badge ' + statusClass(row.status) + '">' + escapeHTML(statusLabel(row.status)) + '</span></td>' +
      '<td>' + escapeHTML(formatDate(row.updated_at)) + '</td>' +
      '<td class="ga-book-actions">' + renderRowActions(row, can) + '</td>' +
    '</tr>'
  }).join('')

  el.innerHTML = [
    '<div class="ga-book-table-wrapper">',
    '  <table class="ga-book-table">',
    '    <thead><tr>',
    '      <th>书名</th><th>ISBN</th><th>作者</th><th>分类</th><th>馆藏位置</th><th>总库存</th><th>可借数量</th><th>状态</th><th>最近更新时间</th><th class="ga-book-actions-th">操作</th>',
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
  parts.push('<button class="ga-book-btn ga-book-btn-small" data-action="detail" data-id="' + id + '">详情</button>')
  if (can && can('update')) {
    parts.push('<button class="ga-book-btn ga-book-btn-small" data-action="edit" data-id="' + id + '">编辑</button>')
  }
  if (can && can('lend')) {
    parts.push('<button class="ga-book-btn ga-book-btn-small" data-action="lend" data-id="' + id + '">借出</button>')
  }
  if (can && can('return_book') && (Number(row.borrowed_count) || 0) > 0) {
    parts.push('<button class="ga-book-btn ga-book-btn-small" data-action="return" data-id="' + id + '">归还</button>')
  }
  if (row.status === 'offshelf') {
    if (can && can('onshelf')) {
      parts.push('<button class="ga-book-btn ga-book-btn-small" data-action="onshelf" data-id="' + id + '">上架</button>')
    }
  } else if (can && can('offshelf')) {
    parts.push('<button class="ga-book-btn ga-book-btn-small" data-action="offshelf" data-id="' + id + '">下架</button>')
  }
  if (can && can('delete')) {
    parts.push('<button class="ga-book-btn ga-book-btn-small-danger" data-action="delete" data-id="' + id + '">删除</button>')
  }
  return parts.join('')
}

export function renderPagination(root, state) {
  var el = root.querySelector('#ga-book-pagination')
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
    pages.push('<button class="ga-book-page-btn" data-action="page" data-page="1">1</button>')
    if (start > 2) pages.push('<span class="ga-book-page-ellipsis">...</span>')
  }
  for (var i = start; i <= end; i++) {
    pages.push('<button class="ga-book-page-btn' + (i === state.page ? ' ga-book-page-active' : '') + '" data-action="page" data-page="' + i + '">' + i + '</button>')
  }
  if (end < totalPages) {
    if (end < totalPages - 1) pages.push('<span class="ga-book-page-ellipsis">...</span>')
    pages.push('<button class="ga-book-page-btn" data-action="page" data-page="' + totalPages + '">' + totalPages + '</button>')
  }
  el.innerHTML =
    '<span class="ga-book-page-info">共 ' + state.total + ' 条</span>' +
    '<div class="ga-book-page-btns">' + pages.join('') + '</div>'
}

function renderEmptyState(el, can) {
  var newBtn = (can && can('create'))
    ? '<button class="ga-book-btn ga-book-btn-primary" data-action="open-create" style="margin-top:12px;">新建图书</button>'
    : ''
  el.innerHTML = [
    '<div class="ga-book-empty">',
    '  <p>暂无图书，点击"新建图书"录入第一本馆藏</p>',
    newBtn,
    '</div>',
  ].join('')
}

function renderSearchEmptyState(el) {
  el.innerHTML = [
    '<div class="ga-book-empty">',
    '  <p>没有符合条件的图书</p>',
    '  <button class="ga-book-btn ga-book-btn-secondary" data-action="clear-filters" style="margin-top:8px;">清除筛选</button>',
    '</div>',
  ].join('')
}

function renderErrorState(el) {
  el.innerHTML = [
    '<div class="ga-book-error">',
    '  <p>加载失败，请重试</p>',
    '  <button class="ga-book-btn ga-book-btn-primary" data-action="reload" style="margin-top:8px;">重试</button>',
    '</div>',
  ].join('')
}

function renderLoadingState(el) {
  var skeleton = ''
  for (var i = 0; i < 5; i++) {
    skeleton += '<div class="ga-book-skeleton-row"><div class="ga-book-skeleton" style="width:' + (55 + i * 7 % 30) + '%"></div></div>'
  }
  el.innerHTML = '<div class="ga-book-loading">' + skeleton + '</div>'
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
