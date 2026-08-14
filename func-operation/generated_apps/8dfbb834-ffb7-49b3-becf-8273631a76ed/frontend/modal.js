import { escapeHTML } from './dom.js'
import { GENDER_OPTIONS, EDUCATION_OPTIONS, statusLabel, statusClass, genderLabel, educationLabel, countryLabel } from './state.js'
import { formatDate } from './ui.js'

// ---------- 基础弹窗 / 抽屉 / toast ----------

export function openModal(root, title, bodyHtml, footerHtml, width) {
  closeModal(root)
  var mask = document.createElement('div')
  mask.className = 'ga-teacher-modal-mask'
  mask.innerHTML = [
    '<div class="ga-teacher-modal" style="width:' + (width || '520px') + '">',
    '  <div class="ga-teacher-modal-header">',
    '    <span class="ga-teacher-modal-title">' + escapeHTML(title) + '</span>',
    '    <button class="ga-teacher-modal-close" data-action="close-modal">&times;</button>',
    '  </div>',
    '  <div class="ga-teacher-modal-body">' + (bodyHtml || '') + '</div>',
    '  <div class="ga-teacher-modal-footer">' + (footerHtml || '') + '</div>',
    '</div>',
  ].join('')
  root.appendChild(mask)
  return mask
}

export function closeModal(root) {
  var existing = root.querySelector('.ga-teacher-modal-mask')
  if (existing) existing.remove()
}

export function openDrawer(root, title, bodyHtml, footerHtml) {
  closeDrawer(root)
  var mask = document.createElement('div')
  mask.className = 'ga-teacher-drawer-mask'
  mask.innerHTML = [
    '<div class="ga-teacher-drawer">',
    '  <div class="ga-teacher-drawer-header">',
    '    <span class="ga-teacher-drawer-title">' + escapeHTML(title) + '</span>',
    '    <button class="ga-teacher-drawer-close" data-action="close-drawer">&times;</button>',
    '  </div>',
    '  <div class="ga-teacher-drawer-body">' + (bodyHtml || '') + '</div>',
    '  <div class="ga-teacher-drawer-footer">' + (footerHtml || '') + '</div>',
    '</div>',
  ].join('')
  root.appendChild(mask)
  return mask
}

export function closeDrawer(root) {
  var existing = root.querySelector('.ga-teacher-drawer-mask')
  if (existing) existing.remove()
}

export function showToast(root, type, message) {
  var container = root.querySelector('.ga-teacher-toast-container')
  if (!container) return
  var toast = document.createElement('div')
  toast.className = 'ga-teacher-toast ga-teacher-toast-' + (type === 'error' ? 'error' : 'success')
  toast.textContent = message
  toast.setAttribute('data-toast', '1')
  container.appendChild(toast)
  var dur = type === 'error' ? 4000 : 2500
  setTimeout(function () {
    if (toast.parentNode) {
      toast.style.opacity = '0'
      toast.style.transform = 'translateY(-8px)'
      setTimeout(function () { if (toast.parentNode) toast.remove() }, 300)
    }
  }, dur)
}

// ---------- 新建 / 编辑教师表单弹窗 ----------

export function renderFormModal(root, state, can) {
  var isEdit = state.modalMode === 'edit'
  var title = isEdit ? '编辑教师' : '新建教师'
  var fd = state.form || {}
  var fe = state.formErrors || {}

  var globalErr = fe.global
    ? '<div class="ga-teacher-modal-error">' + escapeHTML(fe.global) + '</div>'
    : ''

  var statusBlock = ''
  if (isEdit && fd.status) {
    statusBlock =
      '<div class="ga-teacher-form-field">' +
        '<label class="ga-teacher-form-label">状态</label>' +
        '<div><span class="ga-teacher-badge ' + statusClass(fd.status) + '">' + escapeHTML(statusLabel(fd.status)) + '</span></div>' +
      '</div>'
  }

  var bodyHtml =
    globalErr +
    '<div class="ga-teacher-form-section">' +
      '<div class="ga-teacher-form-section-title">基本信息</div>' +
      textFieldHtml('name', '姓名', fd.name || '', true, fe.name, '请输入姓名') +
      textFieldHtml('employee_no', '工号', fd.employee_no || '', true, fe.employee_no, '3-30 位字母或数字') +
      countryFieldHtml(state, fd, fe) +
      selectFieldHtml('gender', '性别', GENDER_OPTIONS, fd.gender || 'male', fe.gender) +
      dateFieldHtml('birth_date', '出生日期', fd.birth_date || '', fe.birth_date) +
      selectFieldHtml('education', '学历', EDUCATION_OPTIONS, fd.education || '', fe.education) +
    '</div>' +
    '<div class="ga-teacher-form-section">' +
      '<div class="ga-teacher-form-section-title">联系方式</div>' +
      textFieldHtml('phone', '联系电话', fd.phone || '', true, fe.phone, '手机 / 座机') +
      textFieldHtml('email', '电子邮箱', fd.email || '', false, fe.email, 'example@school.edu.cn') +
    '</div>' +
    '<div class="ga-teacher-form-section">' +
      '<div class="ga-teacher-form-section-title">任职信息</div>' +
      textFieldHtml('department', '所属院系', fd.department || '', true, fe.department, '请输入所属院系') +
      textFieldHtml('title', '职称', fd.title || '', false, fe.title, '如 讲师 / 副教授 / 教授') +
      dateFieldHtml('hire_date', '入职日期', fd.hire_date || '', fe.hire_date) +
      statusBlock +
    '</div>'

  var showSave = isEdit
    ? (can && can('teacher_update'))
    : (can && can('teacher_create'))
  var saveBtn = showSave
    ? '<button class="ga-teacher-btn ga-teacher-btn-primary" data-action="teacher-save"' + (state.saving ? ' disabled' : '') + '>' + (state.saving ? '保存中...' : '保存') + '</button>'
    : ''
  var footerHtml =
    '<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="close-modal">取消</button>' +
    saveBtn

  openModal(root, title, bodyHtml, footerHtml, '520px')
}

// ---------- 教师详情抽屉 ----------

export function renderDetailDrawer(root, state, can) {
  if (state.detailError) {
    openDrawer(root, '教师详情',
      '<div class="ga-teacher-error-block">' +
        '<p>' + escapeHTML(state.detailError) + '</p>' +
        '<button class="ga-teacher-btn ga-teacher-btn-primary" data-action="detail-retry" data-id="' + escapeHTML(String(state.selectedId)) + '">重试</button>' +
      '</div>',
      '<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="close-drawer">关闭</button>')
    return
  }

  var d = state.detail
  if (!d) return
  var id = d.id

  var bodyHtml = [
    detailField('姓名', d.name),
    detailField('工号', d.employee_no),
    detailField('国籍', countryLabel(state.countryOptions, d.country)),
    detailField('性别', genderLabel(d.gender)),
    detailField('出生日期', d.birth_date),
    detailField('学历', educationLabel(d.education)),
    detailField('所属院系', d.department),
    detailField('职称', d.title),
    detailField('联系电话', d.phone),
    detailField('电子邮箱', d.email),
    detailField('入职日期', d.hire_date),
    '<div class="ga-teacher-detail-field"><span class="ga-teacher-detail-label">状态</span><span class="ga-teacher-detail-value"><span class="ga-teacher-badge ' + statusClass(d.status) + '">' + escapeHTML(statusLabel(d.status)) + '</span></span></div>',
    detailField('最近更新时间', formatDate(d.updated_at)),
  ].join('')

  var actions = []
  var status = d.status
  if (can && can('teacher_update')) {
    actions.push('<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="edit" data-id="' + id + '">编辑</button>')
  }
  if (can && can('teacher_change_status')) {
    if (status === 'active') {
      actions.push('<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="change-status" data-id="' + id + '" data-status="suspended">停用</button>')
      actions.push('<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="change-status" data-id="' + id + '" data-status="resigned">离职</button>')
    } else if (status === 'suspended') {
      actions.push('<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="change-status" data-id="' + id + '" data-status="active">在职</button>')
      actions.push('<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="change-status" data-id="' + id + '" data-status="resigned">离职</button>')
    }
  }
  if (can && can('teacher_delete')) {
    actions.push('<button class="ga-teacher-btn ga-teacher-btn-danger" data-action="delete" data-id="' + id + '">删除</button>')
  }

  openDrawer(root, '教师详情', bodyHtml, actions.join(''))
}

// ---------- 状态流转 / 删除确认框 ----------

export function renderConfirmModal(root, state, can) {
  var c = state.confirm
  if (!c) return
  var showOk = true
  if (can && c.actionKey && !can(c.actionKey)) showOk = false
  var okBtn = showOk
    ? '<button class="ga-teacher-btn ga-teacher-btn-primary" data-action="confirm-ok"' + (state.saving ? ' disabled' : '') + '>' + escapeHTML(c.confirmText || '确认') + '</button>'
    : ''
  var footerHtml =
    '<button class="ga-teacher-btn ga-teacher-btn-secondary" data-action="confirm-cancel">' + escapeHTML(c.cancelText || '取消') + '</button>' +
    okBtn
  openModal(root, c.title || '确认操作', '<p class="ga-teacher-confirm-message">' + escapeHTML(c.message || '') + '</p>', footerHtml, '360px')
}

// ---------- 表单 / 详情辅助 ----------

function detailField(label, value) {
  return '<div class="ga-teacher-detail-field"><span class="ga-teacher-detail-label">' + escapeHTML(label) + '</span><span class="ga-teacher-detail-value">' + escapeHTML(value || '') + '</span></div>'
}

function textFieldHtml(field, label, value, required, error, placeholder) {
  var reqMark = required ? '<span class="ga-teacher-required">*</span>' : ''
  var errHtml = error ? '<div class="ga-teacher-field-error">' + escapeHTML(error) + '</div>' : ''
  return '<div class="ga-teacher-form-field">' +
    '<label class="ga-teacher-form-label">' + reqMark + escapeHTML(label) + '</label>' +
    '<input class="ga-teacher-input' + (error ? ' ga-teacher-input-error' : '') + '" type="text" data-field="' + field + '" value="' + escapeHTML(value) + '" placeholder="' + escapeHTML(placeholder || '') + '" />' +
    errHtml +
  '</div>'
}

function dateFieldHtml(field, label, value, error) {
  var errHtml = error ? '<div class="ga-teacher-field-error">' + escapeHTML(error) + '</div>' : ''
  return '<div class="ga-teacher-form-field">' +
    '<label class="ga-teacher-form-label">' + escapeHTML(label) + '</label>' +
    '<input class="ga-teacher-input' + (error ? ' ga-teacher-input-error' : '') + '" type="date" data-field="' + field + '" value="' + escapeHTML(value) + '" />' +
    errHtml +
  '</div>'
}

function selectFieldHtml(field, label, options, value, error) {
  var errHtml = error ? '<div class="ga-teacher-field-error">' + escapeHTML(error) + '</div>' : ''
  var optsHtml = '<option value="">请选择</option>' + options.map(function (opt) {
    var v = opt.value
    var t = opt.label
    return '<option value="' + escapeHTML(v) + '"' + (v === value ? ' selected' : '') + '>' + escapeHTML(t) + '</option>'
  }).join('')
  return '<div class="ga-teacher-form-field">' +
    '<label class="ga-teacher-form-label">' + escapeHTML(label) + '</label>' +
    '<select class="ga-teacher-select' + (error ? ' ga-teacher-input-error' : '') + '" data-field="' + field + '">' + optsHtml + '</select>' +
    errHtml +
  '</div>'
}

// countryFieldHtml 渲染表单国籍下拉。选项来自全局配置键 country，
// 配置加载失败时置灰并展示「选项加载失败」+「重试」，不阻断其余字段保存。
function countryFieldHtml(state, fd, fe) {
  var errHtml = (fe && fe.country)
    ? '<div class="ga-teacher-field-error">' + escapeHTML(fe.country) + '</div>'
    : ''
  var value = (fd && fd.country) || ''
  var control
  if (state.countryOptionsError) {
    control = '<div class="ga-teacher-country-control">' +
      '<select class="ga-teacher-select" data-field="country" disabled><option value="">请选择</option></select>' +
      '<span class="ga-teacher-options-error">选项加载失败</span>' +
      '<button class="ga-teacher-btn ga-teacher-btn-small" data-action="country-retry" type="button">重试</button>' +
    '</div>'
  } else {
    var hasValue = false
    var opts = '<option value="">请选择</option>'
    ;(state.countryOptions || []).forEach(function (o) {
      var v = o && o.value
      if (v === value) hasValue = true
      var t = (o && o.label) || v
      opts += '<option value="' + escapeHTML(v) + '"' + (v === value ? ' selected' : '') + '>' + escapeHTML(t) + '</option>'
    })
    // 历史记录的国籍值可能不在当前配置选项中，仍回显原值供保存。
    if (value && !hasValue) {
      opts = '<option value="' + escapeHTML(value) + '" selected>' + escapeHTML(value) + '</option>' + opts
    }
    control = '<select class="ga-teacher-select' + (fe && fe.country ? ' ga-teacher-input-error' : '') + '" data-field="country"' + (state.countryOptionsLoading ? ' disabled' : '') + '>' + opts + '</select>'
  }
  return '<div class="ga-teacher-form-field">' +
    '<label class="ga-teacher-form-label">' + escapeHTML('国籍') + '</label>' +
    control +
    errHtml +
  '</div>'
}
