import { injectStyles } from './frontend/styles.js'
import { createApi } from './frontend/api.js'
import * as st from './frontend/state.js'
import {
  renderShell, renderHeader, renderToolbar, renderTable, renderPagination, findRow,
} from './frontend/ui.js'
import {
  closeModal, closeDrawer, showToast, renderFormModal, renderDetailDrawer, renderConfirmModal,
} from './frontend/modal.js'

export function render(container, context) {
  const appId = context.app.id

  function invoke(payload) {
    return context.invokeData(appId, payload)
  }

  const can = typeof context.can === 'function'
    ? function (action) { return context.can(action) }
    : function () { return false }

  injectStyles()
  const state = st.createInitialState()
  const api = createApi(invoke)

  renderShell(container, state)
  const root = container.querySelector('#ga-teacher-root')
  if (!root) return

  function renderAll() {
    renderHeader(root, state, can)
    renderToolbar(root, state)
    renderTable(root, state, can)
    renderPagination(root, state)
  }

  function loadList(extraFilters) {
    if (extraFilters) {
      if (extraFilters.keyword !== undefined) state.keyword = extraFilters.keyword
      if (extraFilters.countryFilter !== undefined) state.countryFilter = extraFilters.countryFilter
      if (extraFilters.departmentFilter !== undefined) state.departmentFilter = extraFilters.departmentFilter
      if (extraFilters.statusFilter !== undefined) state.statusFilter = extraFilters.statusFilter
      if (extraFilters.educationFilter !== undefined) state.educationFilter = extraFilters.educationFilter
      if (extraFilters.titleFilter !== undefined) state.titleFilter = extraFilters.titleFilter
      if (extraFilters.page !== undefined) state.page = extraFilters.page
    }
    state.loading = true
    state.error = null
    renderTable(root, state, can)
    renderPagination(root, state)

    var prevOptions = JSON.stringify(state.departmentOptions || []) + '|' + JSON.stringify(state.titleOptions || [])

    api.teacherList({
      keyword: state.keyword || '',
      country: state.countryFilter || '',
      department: state.departmentFilter || '',
      status: state.statusFilter || '',
      education: state.educationFilter || '',
      title: state.titleFilter || '',
      page: state.page || 1,
      pageSize: state.pageSize || 10,
    }).then(function (result) {
      if (result && result.success) {
        state.rows = (result.rows || []).map(st.normalizeRow)
        state.total = result.total || 0
        if (Array.isArray(result.departmentOptions)) state.departmentOptions = st.mergeOptions(state.departmentOptions, result.departmentOptions)
        if (Array.isArray(result.titleOptions)) state.titleOptions = st.mergeOptions(state.titleOptions, result.titleOptions)
        state.loading = false
        state.error = null
      } else {
        state.rows = []
        state.total = 0
        state.loading = false
        state.error = (result && result.error) || '加载失败'
      }
      var nowOptions = JSON.stringify(state.departmentOptions || []) + '|' + JSON.stringify(state.titleOptions || [])
      if (nowOptions !== prevOptions) renderToolbar(root, state)
      renderTable(root, state, can)
      renderPagination(root, state)
    }).catch(function () {
      state.rows = []
      state.total = 0
      state.loading = false
      state.error = '网络异常'
      renderTable(root, state, can)
      renderPagination(root, state)
    })
  }

  function openTeacherForm(mode, data) {
    state.modalMode = mode
    state.formErrors = {}
    if (mode === 'create') {
      state.form = st.defaultForm()
    } else if (mode === 'edit' && data) {
      state.form = {
        id: data.id,
        name: data.name || '',
        employee_no: data.employee_no || '',
        country: data.country || '',
        gender: data.gender || 'male',
        birth_date: data.birth_date || '',
        education: data.education || '',
        department: data.department || '',
        title: data.title || '',
        phone: data.phone || '',
        email: data.email || '',
        hire_date: data.hire_date || '',
        status: data.status || '',
      }
    }
    renderFormModal(root, state, can)
    refreshCountryOptions(function () {
      if (state.modalMode === mode) renderFormModal(root, state, can)
    })
  }

  // refreshCountryOptions 读取全局配置键 country 选项并写入 state。
  // done 回调在读取完成（成功或失败）后执行，用于重渲染国籍下拉。
  function refreshCountryOptions(done) {
    state.countryOptionsLoading = true
    state.countryOptionsError = null
    st.loadCountryOptions(context).then(function (res) {
      state.countryOptionsLoading = false
      if (res && res.error) {
        state.countryOptionsError = res.error
        state.countryOptions = []
      } else {
        state.countryOptions = (res && res.options) || []
        state.countryOptionsError = null
      }
      if (done) done()
    })
  }

  function openConfirm(options) {
    state.confirm = {
      title: options.title,
      message: options.message,
      confirmText: options.confirmText,
      cancelText: options.cancelText,
      actionKey: options.actionKey || '',
      onConfirm: options.onConfirm,
    }
    renderConfirmModal(root, state, can)
  }

  function openDetail(id) {
    state.selectedId = id
    state.detail = null
    state.detailError = null
    api.teacherDetail(id).then(function (result) {
      if (result && result.success && result.data) {
        state.detail = st.normalizeRow(result.data)
        state.detailError = null
        renderDetailDrawer(root, state, can)
      } else {
        state.detail = null
        state.detailError = (result && result.error) || '加载教师详情失败'
        renderDetailDrawer(root, state, can)
      }
    }).catch(function () {
      state.detail = null
      state.detailError = '网络异常，加载教师详情失败'
      renderDetailDrawer(root, state, can)
    })
  }

  function refreshDetailAfterMutation() {
    api.teacherDetail(state.selectedId).then(function (result) {
      if (result && result.success && result.data) {
        state.detail = st.normalizeRow(result.data)
        state.detailError = null
        renderDetailDrawer(root, state, can)
      } else {
        closeDrawer(root)
        state.detail = null
        state.selectedId = null
      }
      loadList()
    }).catch(function () {
      closeDrawer(root)
      state.detail = null
      state.selectedId = null
      loadList()
    })
  }

  function afterMutation(successMsg, keepDetail) {
    closeModal(root)
    showToast(root, 'success', successMsg)
    if (keepDetail && state.selectedId) {
      refreshDetailAfterMutation()
    } else {
      closeDrawer(root)
      state.detail = null
      state.selectedId = null
      loadList()
    }
  }

  function handleTeacherSave() {
    var isEdit = state.modalMode === 'edit'
    if (isEdit) {
      if (!can('teacher_update')) return
    } else if (!can('teacher_create')) {
      return
    }
    var errors = st.validateTeacherForm(state.form, state.countryOptions)
    if (Object.keys(errors).length > 0) {
      st.setFormErrors(state, errors)
      renderFormModal(root, state, can)
      return
    }
    st.setSaving(state, true)
    renderFormModal(root, state, can)

    var payload = st.formToPayload(state.form)
    var promise = isEdit ? api.teacherUpdate(state.form.id, payload) : api.teacherCreate(payload)

    promise.then(function (result) {
      st.setSaving(state, false)
      if (result && result.success) {
        var wasEdit = isEdit
        state.modalMode = null
        state.formErrors = {}
        state.form = st.defaultForm()
        if (!wasEdit) state.page = 1
        afterMutation(wasEdit ? '更新成功' : '新建成功', wasEdit)
      } else {
        var errMsg = (result && result.error) || '保存失败，请重试'
        var errs = {}
        if (errMsg.indexOf('工号') >= 0) {
          errs.employee_no = '工号已存在，请检查'
        } else {
          errs.global = errMsg
        }
        st.setFormErrors(state, errs)
        renderFormModal(root, state, can)
      }
    }).catch(function () {
      st.setSaving(state, false)
      st.setFormErrors(state, { global: '网络异常，请重试' })
      renderFormModal(root, state, can)
    })
  }

  function runStatusAction(id, targetStatus, name) {
    if (!can('teacher_change_status')) return
    var label = targetStatus === 'active' ? '恢复在职' : (targetStatus === 'suspended' ? '停用' : '离职')
    var message
    if (targetStatus === 'active') {
      message = '确定将该教师恢复为在职？'
    } else if (targetStatus === 'suspended') {
      message = '确定将该教师停用？停用后将不再参与排课与教学。'
    } else {
      message = '确定将该教师办理离职？离职后档案保留，不再在职。'
    }
    openConfirm({
      title: label,
      message: message,
      confirmText: '确认',
      cancelText: '取消',
      actionKey: 'teacher_change_status',
      onConfirm: function () {
        if (!can('teacher_change_status')) return
        st.setSaving(state, true)
        renderConfirmModal(root, state, can)
        api.teacherChangeStatus(id, targetStatus).then(function (result) {
          st.setSaving(state, false)
          state.confirm = null
          if (result && result.success) {
            var keep = state.detail && state.selectedId === id
            afterMutation(label + '成功', keep)
          } else {
            showToast(root, 'error', (result && result.error) || '状态流转失败，请重试')
            closeModal(root)
          }
        }).catch(function () {
          st.setSaving(state, false)
          state.confirm = null
          closeModal(root)
          showToast(root, 'error', '状态流转失败，请重试')
        })
      },
    })
  }

  function doDelete(id, name, status) {
    if (!can('teacher_delete')) return
    if (status !== 'resigned') {
      var blockMsg = status === 'active' ? '在职教师不可删除，请先办理离职' : '停用教师不可删除，请先办理离职'
      showToast(root, 'error', blockMsg)
      return
    }
    openConfirm({
      title: '删除教师',
      message: '确定删除教师「' + (name || '') + '」？删除后不可恢复。',
      confirmText: '删除',
      cancelText: '取消',
      actionKey: 'teacher_delete',
      onConfirm: function () {
        if (!can('teacher_delete')) return
        st.setSaving(state, true)
        renderConfirmModal(root, state, can)
        api.teacherDelete(id).then(function (result) {
          st.setSaving(state, false)
          state.confirm = null
          if (result && result.success) {
            afterMutation('删除成功', false)
          } else {
            showToast(root, 'error', (result && result.error) || '删除失败，请重试')
            closeModal(root)
          }
        }).catch(function () {
          st.setSaving(state, false)
          state.confirm = null
          closeModal(root)
          showToast(root, 'error', '删除失败，请重试')
        })
      },
    })
  }

  // ---------- 事件委托 ----------

  var searchTimer = null

  root.addEventListener('input', function (e) {
    var target = e.target
    if (!target || !target.getAttribute) return
    var action = target.getAttribute('data-action')
    if (action === 'search-input') {
      var val = target.value
      if (searchTimer) clearTimeout(searchTimer)
      searchTimer = setTimeout(function () {
        loadList({ keyword: val, page: 1 })
      }, 300)
      return
    }
    var field = target.getAttribute('data-field')
    if (field && state.modalMode) {
      st.setFormField(state, field, target.value)
    }
  })

  root.addEventListener('change', function (e) {
    var target = e.target
    if (!target || !target.getAttribute) return
    var action = target.getAttribute('data-action')
    if (action === 'filter-department') {
      loadList({ departmentFilter: target.value, page: 1 })
      return
    }
    if (action === 'filter-country') {
      loadList({ countryFilter: target.value, page: 1 })
      return
    }
    if (action === 'filter-status') {
      loadList({ statusFilter: target.value, page: 1 })
      return
    }
    if (action === 'filter-education') {
      loadList({ educationFilter: target.value, page: 1 })
      return
    }
    if (action === 'filter-title') {
      loadList({ titleFilter: target.value, page: 1 })
      return
    }
    var field = target.getAttribute('data-field')
    if (field && state.modalMode && target.tagName === 'SELECT') {
      st.setFormField(state, field, target.value)
    }
  })

  root.addEventListener('click', function (e) {
    var target = e.target && e.target.closest ? e.target.closest('[data-action]') : null
    if (!target) return
    var action = target.getAttribute('data-action')
    var id = target.getAttribute('data-id')
    var status = target.getAttribute('data-status')

    function resolveRow(rid) {
      var row = findRow(state.rows, rid)
      if (!row && state.detail && String(state.detail.id) === String(rid)) row = state.detail
      return row
    }

    switch (action) {
      case 'open-create':
        if (!can('teacher_create')) return
        openTeacherForm('create', null)
        return
      case 'edit': {
        if (!can('teacher_update')) return
        var editRow = resolveRow(id)
        if (!editRow) return
        openTeacherForm('edit', editRow)
        return
      }
      case 'detail':
        openDetail(id)
        return
      case 'change-status': {
        if (!can('teacher_change_status')) return
        var statusRow = resolveRow(id)
        runStatusAction(id, status, statusRow ? statusRow.name : '')
        return
      }
      case 'delete': {
        if (!can('teacher_delete')) return
        var delRow = resolveRow(id)
        doDelete(id, delRow ? delRow.name : '', delRow ? delRow.status : '')
        return
      }
      case 'detail-retry':
        openDetail(id)
        return
      case 'country-retry':
        refreshCountryOptions(function () {
          renderToolbar(root, state)
          if (state.modalMode) renderFormModal(root, state, can)
        })
        return
      case 'reload':
        loadList({ page: 1 })
        return
      case 'clear-filters':
        state.keyword = ''
        state.countryFilter = ''
        state.departmentFilter = ''
        state.statusFilter = ''
        state.educationFilter = ''
        state.titleFilter = ''
        state.page = 1
        renderToolbar(root, state)
        loadList()
        return
      case 'page': {
        var page = Number(target.getAttribute('data-page'))
        if (page) {
          state.page = page
          loadList()
        }
        return
      }
      case 'close-modal':
        state.modalMode = null
        state.formErrors = {}
        state.confirm = null
        closeModal(root)
        return
      case 'close-drawer':
        state.detail = null
        state.selectedId = null
        state.detailError = null
        closeDrawer(root)
        return
      case 'teacher-save':
        handleTeacherSave()
        return
      case 'confirm-ok':
        if (state.confirm && state.confirm.onConfirm) state.confirm.onConfirm()
        return
      case 'confirm-cancel':
        state.confirm = null
        closeModal(root)
        return
      default:
        return
    }
  })

  renderAll()
  loadList()
  refreshCountryOptions(function () {
    renderToolbar(root, state)
  })
}

export function dispose(container) {
  container.innerHTML = ''
}
