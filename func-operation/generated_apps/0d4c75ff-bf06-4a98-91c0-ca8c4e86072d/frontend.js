import { injectStyles } from './frontend/styles.js'
import { createApi } from './frontend/api.js'
import * as st from './frontend/state.js'
import {
  renderShell, renderHeader, renderToolbar, renderTable, renderPagination, findRow,
} from './frontend/ui.js'
import {
  closeModal, closeDrawer, showToast, renderFormModal, renderQuantityModal,
  renderConfirmModal, renderDetailModal,
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
  const root = container.querySelector('#ga-book-root')
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
      if (extraFilters.categoryFilter !== undefined) state.categoryFilter = extraFilters.categoryFilter
      if (extraFilters.statusFilter !== undefined) state.statusFilter = extraFilters.statusFilter
      if (extraFilters.page !== undefined) state.page = extraFilters.page
    }
    state.loading = true
    state.error = null
    renderTable(root, state, can)
    renderPagination(root, state)

    api.listBooks({
      keyword: state.keyword || '',
      category: state.categoryFilter || '',
      status: state.statusFilter || '',
      page: state.page || 1,
      pageSize: state.pageSize || 10,
    }).then(function (result) {
      if (result && result.success) {
        state.rows = (result.rows || []).map(st.normalizeRow)
        state.total = result.total || 0
        state.loading = false
        state.error = null
      } else {
        state.rows = []
        state.total = 0
        state.loading = false
        state.error = (result && result.error) || '加载失败'
      }
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

  function openBookForm(mode, data) {
    state.modalMode = mode
    state.formErrors = {}
    if (mode === 'create') {
      state.form = st.defaultForm()
    } else if (mode === 'edit' && data) {
      state.form = {
        id: data.id,
        title: data.title || '',
        isbn: data.isbn || '',
        author: data.author || '',
        category: data.category || '',
        publisher: data.publisher || '',
        publish_year: (data.publish_year === null || data.publish_year === undefined) ? '' : data.publish_year,
        location: data.location || '',
        total_stock: data.total_stock != null ? data.total_stock : 1,
        borrowed_count: data.borrowed_count,
        available_count: data.available_count,
      }
    }
    renderFormModal(root, state, can)
  }

  function openQuantityModal(type, row) {
    if (type === 'lend') {
      if (!can('lend')) return
      if (row.status === 'offshelf') {
        showToast(root, 'error', '该图书已下架，不能借出')
        return
      }
    } else if (!can('return_book')) {
      return
    }
    state.qtyModal = { type: type, id: row.id, book: row, quantity: 1, error: '' }
    renderQuantityModal(root, state, can)
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
    api.getBook(id).then(function (result) {
      if (result && result.success && result.data) {
        state.detail = st.normalizeRow(result.data)
        state.selectedId = id
        renderDetailModal(root, state, can)
      } else {
        showToast(root, 'error', (result && result.error) || '加载图书详情失败')
      }
    }).catch(function () {
      showToast(root, 'error', '加载图书详情失败')
    })
  }

  function refreshDetailAfterMutation() {
    api.getBook(state.selectedId).then(function (result) {
      if (result && result.success && result.data) {
        state.detail = st.normalizeRow(result.data)
        renderDetailModal(root, state, can)
      } else {
        closeDrawer(root)
        state.detail = null
        state.selectedId = null
      }
      loadList()
    }).catch(function () {
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

  function handleBookSave() {
    var isEdit = state.modalMode === 'edit'
    if (isEdit) {
      if (!can('update')) return
    } else if (!can('create')) {
      return
    }
    var errors = st.validateBookForm(state.form)
    if (Object.keys(errors).length > 0) {
      st.setFormErrors(state, errors)
      renderFormModal(root, state, can)
      return
    }
    st.setSaving(state, true)
    renderFormModal(root, state, can)

    var payload = st.formToPayload(state.form)
    var promise = isEdit ? api.updateBook(state.form.id, payload) : api.createBook(payload)

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
        st.setFormErrors(state, { global: (result && result.error) || '保存失败，请重试' })
        renderFormModal(root, state, can)
      }
    }).catch(function () {
      st.setSaving(state, false)
      st.setFormErrors(state, { global: '网络异常，请重试' })
      renderFormModal(root, state, can)
    })
  }

  function validateQty(value, limit, isLend) {
    var v = String(value)
    var num = Number(v)
    if (v === '' || v === null || v === undefined || isNaN(num) || num < 1 || !Number.isInteger(num)) {
      return '请输入不小于 1 的整数'
    }
    if (num > limit) {
      return isLend ? '超出可借数量' : '归还数量大于借出数量'
    }
    return ''
  }

  function handleQuantityConfirm() {
    var q = state.qtyModal
    if (!q) return
    if (q.type === 'lend') {
      if (!can('lend')) return
    } else if (!can('return_book')) {
      return
    }
    var isLend = q.type === 'lend'
    var limit = isLend ? (Number(q.book.available_count) || 0) : (Number(q.book.borrowed_count) || 0)
    var err = validateQty(q.quantity, limit, isLend)
    if (err) {
      state.qtyModal.error = err
      renderQuantityModal(root, state, can)
      return
    }
    var qty = Number(q.quantity)
    st.setSaving(state, true)
    renderQuantityModal(root, state, can)

    var promise = isLend ? api.lendBook(q.id, qty) : api.returnBook(q.id, qty)
    promise.then(function (result) {
      st.setSaving(state, false)
      if (result && result.success) {
        var keep = state.detail && state.selectedId === q.id
        state.qtyModal = null
        afterMutation(isLend ? '借出成功' : '归还成功', keep)
      } else {
        state.qtyModal.error = (result && result.error) || '操作失败'
        renderQuantityModal(root, state, can)
      }
    }).catch(function () {
      st.setSaving(state, false)
      state.qtyModal.error = '网络异常，请重试'
      renderQuantityModal(root, state, can)
    })
  }

  function runStatusAction(kind, id) {
    var actionKey = kind === 'offshelf' ? 'offshelf' : 'onshelf'
    if (!can(actionKey)) return
    var confirmText = kind === 'offshelf' ? '确认下架' : '确认上架'
    var message = kind === 'offshelf'
      ? '确定将该图书下架？下架后不可借出。'
      : '确定将该图书重新上架？'
    var apiCall = kind === 'offshelf' ? api.offshelfBook(id) : api.onshelfBook(id)
    openConfirm({
      title: kind === 'offshelf' ? '下架图书' : '上架图书',
      message: message,
      confirmText: confirmText,
      cancelText: '取消',
      actionKey: actionKey,
      onConfirm: function () {
        if (!can(actionKey)) return
        st.setSaving(state, true)
        renderConfirmModal(root, state, can)
        apiCall.then(function (result) {
          st.setSaving(state, false)
          state.confirm = null
          if (result && result.success) {
            var keep = state.detail && state.selectedId === id
            afterMutation(kind === 'offshelf' ? '下架成功' : '上架成功', keep)
          } else {
            showToast(root, 'error', (result && result.error) || '操作失败，请重试')
            closeModal(root)
          }
        }).catch(function () {
          st.setSaving(state, false)
          state.confirm = null
          closeModal(root)
          showToast(root, 'error', '操作失败，请重试')
        })
      },
    })
  }

  function doDelete(id, name) {
    if (!can('delete')) return
    openConfirm({
      title: '删除图书',
      message: '确定删除图书《' + (name || '') + '》？存在借出记录时将无法删除。',
      confirmText: '删除',
      cancelText: '取消',
      actionKey: 'delete',
      onConfirm: function () {
        if (!can('delete')) return
        st.setSaving(state, true)
        renderConfirmModal(root, state, can)
        api.deleteBook(id).then(function (result) {
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
    if (field === 'qty' && state.qtyModal) {
      state.qtyModal.quantity = target.value
      if (state.qtyModal.error) state.qtyModal.error = ''
      return
    }
    if (field && state.modalMode) {
      st.setFormField(state, field, target.value)
    }
  })

  root.addEventListener('change', function (e) {
    var target = e.target
    if (!target || !target.getAttribute) return
    var action = target.getAttribute('data-action')
    if (action === 'filter-category') {
      loadList({ categoryFilter: target.value, page: 1 })
      return
    }
    if (action === 'filter-status') {
      loadList({ statusFilter: target.value, page: 1 })
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

    function resolveRow(rid) {
      var row = findRow(state.rows, rid)
      if (!row && state.detail && String(state.detail.id) === String(rid)) row = state.detail
      return row
    }

    switch (action) {
      case 'open-create':
        if (!can('create')) return
        openBookForm('create', null)
        return
      case 'edit': {
        if (!can('update')) return
        var row = resolveRow(id)
        if (!row) return
        openBookForm('edit', row)
        return
      }
      case 'detail':
        openDetail(id)
        return
      case 'delete': {
        if (!can('delete')) return
        var delRow = resolveRow(id)
        doDelete(id, delRow ? delRow.title : '')
        return
      }
      case 'lend': {
        if (!can('lend')) return
        var lendRow = resolveRow(id)
        if (!lendRow) return
        openQuantityModal('lend', lendRow)
        return
      }
      case 'return': {
        if (!can('return_book')) return
        var retRow = resolveRow(id)
        if (!retRow) return
        openQuantityModal('return', retRow)
        return
      }
      case 'offshelf':
        runStatusAction('offshelf', id)
        return
      case 'onshelf':
        runStatusAction('onshelf', id)
        return
      case 'reload':
        loadList({ page: 1 })
        return
      case 'clear-filters':
        state.keyword = ''
        state.categoryFilter = ''
        state.statusFilter = ''
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
        state.qtyModal = null
        state.confirm = null
        closeModal(root)
        return
      case 'close-drawer':
        state.detail = null
        state.selectedId = null
        closeDrawer(root)
        return
      case 'book-save':
        handleBookSave()
        return
      case 'qty-confirm':
        handleQuantityConfirm()
        return
      case 'qty-cancel':
        state.qtyModal = null
        closeModal(root)
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
}

export function dispose(container) {
  container.innerHTML = ''
}
