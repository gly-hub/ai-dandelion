import { escapeHTML } from './dom.js'
import { CATEGORY_OPTIONS, statusLabel, statusClass } from './state.js'
import { formatDate } from './ui.js'

export function openModal(root, title, bodyHtml, footerHtml, width) {
  closeModal(root)
  var mask = document.createElement('div')
  mask.className = 'ga-book-modal-mask'
  mask.innerHTML = [
    '<div class="ga-book-modal" style="width:' + (width || '520px') + '">',
    '  <div class="ga-book-modal-header">',
    '    <span class="ga-book-modal-title">' + escapeHTML(title) + '</span>',
    '    <button class="ga-book-modal-close" data-action="close-modal">&times;</button>',
    '  </div>',
    '  <div class="ga-book-modal-body">' + (bodyHtml || '') + '</div>',
    '  <div class="ga-book-modal-footer">' + (footerHtml || '') + '</div>',
    '</div>',
  ].join('')
  root.appendChild(mask)
  return mask
}

export function closeModal(root) {
  var existing = root.querySelector('.ga-book-modal-mask')
  if (existing) existing.remove()
}

export function openDrawer(root, title, bodyHtml, footerHtml) {
  closeDrawer(root)
  var mask = document.createElement('div')
  mask.className = 'ga-book-drawer-mask'
  mask.innerHTML = [
    '<div class="ga-book-drawer">',
    '  <div class="ga-book-drawer-header">',
    '    <span class="ga-book-drawer-title">' + escapeHTML(title) + '</span>',
    '    <button class="ga-book-drawer-close" data-action="close-drawer">&times;</button>',
    '  </div>',
    '  <div class="ga-book-drawer-body">' + (bodyHtml || '') + '</div>',
    '  <div class="ga-book-drawer-footer">' + (footerHtml || '') + '</div>',
    '</div>',
  ].join('')
  root.appendChild(mask)
  return mask
}

export function closeDrawer(root) {
  var existing = root.querySelector('.ga-book-drawer-mask')
  if (existing) existing.remove()
}

export function showToast(root, type, message) {
  var container = root.querySelector('.ga-book-toast-container')
  if (!container) return
  var toast = document.createElement('div')
  toast.className = 'ga-book-toast ga-book-toast-' + (type === 'error' ? 'error' : 'success')
  toast.textContent = message
  toast.setAttribute('data-toast', '1')
  container.appendChild(toast)
  var dur = type === 'error' ? 4000 : 2500
  setTimeout(function () {
    if (toast.parentNode) {
      toast.style.opacity = '0'
      toast.style.transform = 'translateX(40px)'
      setTimeout(function () { if (toast.parentNode) toast.remove() }, 300)
    }
  }, dur)
}

// ---------- 新建 / 编辑图书表单弹窗 ----------

export function renderFormModal(root, state, can) {
  var isEdit = state.modalMode === 'edit'
  var title = isEdit ? '编辑图书' : '新建图书'
  var fd = state.form || {}
  var fe = state.formErrors || {}

  var globalErr = fe.global
    ? '<div class="ga-book-modal-error">' + escapeHTML(fe.global) + '</div>'
    : ''

  var stockInfo = ''
  if (isEdit) {
    stockInfo =
      '<div class="ga-book-form-field">' +
        '<label class="ga-book-form-label">借出数量</label>' +
        '<div class="ga-book-readonly-value">' + escapeHTML(String(fd.borrowed_count != null ? fd.borrowed_count : '')) + '</div>' +
      '</div>' +
      '<div class="ga-book-form-field">' +
        '<label class="ga-book-form-label">可借数量</label>' +
        '<div class="ga-book-readonly-value">' + escapeHTML(String(fd.available_count != null ? fd.available_count : '')) + '</div>' +
      '</div>'
  }

  var bodyHtml =
    globalErr +
    '<div class="ga-book-form-section">' +
      '<div class="ga-book-form-section-title">基础信息</div>' +
      textFieldHtml('title', '书名', fd.title || '', true, fe.title, '请输入书名') +
      textFieldHtml('isbn', 'ISBN', fd.isbn || '', true, fe.isbn, 'ISBN-10 / ISBN-13') +
      textFieldHtml('author', '作者', fd.author || '', false, fe.author, '') +
      selectFieldHtml('category', '分类', CATEGORY_OPTIONS, fd.category || '', fe.category) +
      textFieldHtml('publisher', '出版社', fd.publisher || '', false, fe.publisher, '') +
      textFieldHtml('publish_year', '出版年份', fd.publish_year != null && fd.publish_year !== '' ? fd.publish_year : '', false, fe.publish_year, '如 2008', 'number') +
      textFieldHtml('location', '馆藏位置', fd.location || '', false, fe.location, '如 A区-3排') +
    '</div>' +
    '<div class="ga-book-form-section">' +
      '<div class="ga-book-form-section-title">库存信息</div>' +
      textFieldHtml('total_stock', '总库存', fd.total_stock != null && fd.total_stock !== '' ? fd.total_stock : '', true, fe.total_stock, '不少于 1', 'number') +
      stockInfo +
    '</div>'

  var showSave = isEdit
    ? (can && can('update'))
    : (can && can('create'))
  var saveBtn = showSave
    ? '<button class="ga-book-btn ga-book-btn-primary" data-action="book-save" id="ga-book-save-btn"' + (state.saving ? ' disabled' : '') + '>' + (state.saving ? '保存中...' : '保存') + '</button>'
    : ''
  var footerHtml =
    '<button class="ga-book-btn ga-book-btn-secondary" data-action="close-modal">取消</button>' +
    saveBtn

  openModal(root, title, bodyHtml, footerHtml, '520px')
}

// ---------- 借出 / 归还数量弹窗 ----------

export function renderQuantityModal(root, state, can) {
  var q = state.qtyModal
  if (!q) return
  var isLend = q.type === 'lend'
  var title = isLend ? '借出图书' : '归还图书'
  var book = q.book || {}
  var limit = isLend ? (Number(book.available_count) || 0) : (Number(book.borrowed_count) || 0)
  var hint = isLend ? '当前可借数量：' + limit : '当前借出数量：' + limit
  var errHtml = q.error ? '<div class="ga-book-field-error">' + escapeHTML(q.error) + '</div>' : ''
  var qtyValue = q.quantity != null ? q.quantity : 1

  var bodyHtml =
    '<div class="ga-book-form-field">' +
      '<label class="ga-book-form-label">图书</label>' +
      '<div class="ga-book-qty-book">' + escapeHTML(book.title || '') + '</div>' +
    '</div>' +
    '<div class="ga-book-form-field">' +
      '<label class="ga-book-form-label">' + hint + '</label>' +
      '<input class="ga-book-input' + (q.error ? ' ga-book-input-error' : '') + '" type="number" min="1" step="1" data-field="qty" value="' + escapeHTML(String(qtyValue)) + '" />' +
      errHtml +
    '</div>'

  var confirmText = isLend ? '确认借出' : '确认归还'
  var showConfirm = isLend ? (can && can('lend')) : (can && can('return_book'))
  var confirmBtn = showConfirm
    ? '<button class="ga-book-btn ga-book-btn-primary" data-action="qty-confirm"' + (state.saving ? ' disabled' : '') + '>' + (state.saving ? '处理中...' : confirmText) + '</button>'
    : ''
  var footerHtml =
    '<button class="ga-book-btn ga-book-btn-secondary" data-action="qty-cancel">取消</button>' +
    confirmBtn

  openModal(root, title, bodyHtml, footerHtml, '360px')
}

// ---------- 删除 / 下架 / 上架确认弹窗 ----------

export function renderConfirmModal(root, state, can) {
  var c = state.confirm
  if (!c) return
  var showOk = true
  if (can && c.actionKey && !can(c.actionKey)) showOk = false
  var okBtn = showOk
    ? '<button class="ga-book-btn ga-book-btn-danger" data-action="confirm-ok"' + (state.saving ? ' disabled' : '') + '>' + escapeHTML(c.confirmText || '确认') + '</button>'
    : ''
  var footerHtml =
    '<button class="ga-book-btn ga-book-btn-secondary" data-action="confirm-cancel">' + escapeHTML(c.cancelText || '取消') + '</button>' +
    okBtn
  openModal(root, c.title || '确认操作', '<p class="ga-book-confirm-message">' + escapeHTML(c.message || '') + '</p>', footerHtml, '360px')
}

// ---------- 图书详情抽屉 ----------

export function renderDetailModal(root, state, can) {
  var d = state.detail
  if (!d) return
  var statusText = statusLabel(d.status)
  var statusCls = statusClass(d.status)

  var bodyHtml = [
    detailField('书名', d.title),
    detailField('ISBN', d.isbn),
    detailField('作者', d.author),
    detailField('分类', d.category || '未分类'),
    detailField('出版社', d.publisher),
    detailField('出版年份', d.publish_year != null ? String(d.publish_year) : ''),
    detailField('馆藏位置', d.location),
    detailField('总库存', d.total_stock != null ? String(d.total_stock) : ''),
    detailField('可借数量', d.available_count != null ? String(d.available_count) : ''),
    detailField('借出数量', d.borrowed_count != null ? String(d.borrowed_count) : ''),
    '<div class="ga-book-detail-field"><span class="ga-book-detail-label">状态</span><span class="ga-book-detail-value"><span class="ga-book-badge ' + statusCls + '">' + escapeHTML(statusText) + '</span></span></div>',
    detailField('最近更新时间', formatDate(d.updated_at)),
  ].join('')

  var actions = []
  var id = d.id
  if (can && can('update')) actions.push('<button class="ga-book-btn ga-book-btn-secondary" data-action="edit" data-id="' + id + '">编辑</button>')
  if (can && can('lend')) actions.push('<button class="ga-book-btn ga-book-btn-primary" data-action="lend" data-id="' + id + '">借出</button>')
  if (can && can('return_book') && (Number(d.borrowed_count) || 0) > 0) actions.push('<button class="ga-book-btn ga-book-btn-primary" data-action="return" data-id="' + id + '">归还</button>')
  if (d.status === 'offshelf') {
    if (can && can('onshelf')) actions.push('<button class="ga-book-btn ga-book-btn-secondary" data-action="onshelf" data-id="' + id + '">上架</button>')
  } else if (can && can('offshelf')) {
    actions.push('<button class="ga-book-btn ga-book-btn-secondary" data-action="offshelf" data-id="' + id + '">下架</button>')
  }
  if (can && can('delete')) actions.push('<button class="ga-book-btn ga-book-btn-danger" data-action="delete" data-id="' + id + '">删除</button>')

  openDrawer(root, '图书详情', bodyHtml, actions.join(''))
}

function detailField(label, value) {
  return '<div class="ga-book-detail-field"><span class="ga-book-detail-label">' + escapeHTML(label) + '</span><span class="ga-book-detail-value">' + escapeHTML(value || '') + '</span></div>'
}

function textFieldHtml(field, label, value, required, error, placeholder, type) {
  var reqMark = required ? '<span class="ga-book-required">*</span>' : ''
  var errHtml = error ? '<div class="ga-book-field-error">' + escapeHTML(error) + '</div>' : ''
  return '<div class="ga-book-form-field">' +
    '<label class="ga-book-form-label">' + reqMark + escapeHTML(label) + '</label>' +
    '<input class="ga-book-input' + (error ? ' ga-book-input-error' : '') + '" type="' + (type || 'text') + '" data-field="' + field + '" value="' + escapeHTML(value) + '" placeholder="' + escapeHTML(placeholder || '') + '" />' +
    errHtml +
  '</div>'
}

function selectFieldHtml(field, label, options, value, error) {
  var reqMark = ''
  var errHtml = error ? '<div class="ga-book-field-error">' + escapeHTML(error) + '</div>' : ''
  var optsHtml = '<option value="">未分类</option>' + options.map(function (opt) {
    return '<option value="' + escapeHTML(opt) + '"' + (opt === value ? ' selected' : '') + '>' + escapeHTML(opt) + '</option>'
  }).join('')
  return '<div class="ga-book-form-field">' +
    '<label class="ga-book-form-label">' + reqMark + escapeHTML(label) + '</label>' +
    '<select class="ga-book-select' + (error ? ' ga-book-input-error' : '') + '" data-field="' + field + '">' + optsHtml + '</select>' +
    errHtml +
  '</div>'
}
