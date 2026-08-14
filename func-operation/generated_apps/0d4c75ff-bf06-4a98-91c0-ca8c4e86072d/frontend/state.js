export const CATEGORY_OPTIONS = ['文学', '科技', '历史', '艺术', '经济', '教育', '生活', '其他', '未分类']

export const STATUS_OPTIONS = [
  { value: 'onshelf', label: '在馆', cls: 'ga-book-badge-onshelf' },
  { value: 'lent', label: '借出', cls: 'ga-book-badge-lent' },
  { value: 'offshelf', label: '下架', cls: 'ga-book-badge-offshelf' },
]

export function statusLabel(status) {
  const opt = STATUS_OPTIONS.find(function (o) { return o.value === status })
  return opt ? opt.label : (status || '')
}

export function statusClass(status) {
  const opt = STATUS_OPTIONS.find(function (o) { return o.value === status })
  return opt ? opt.cls : 'ga-book-badge-onshelf'
}

export function createInitialState() {
  return {
    rows: [],
    total: 0,
    page: 1,
    pageSize: 10,
    keyword: '',
    categoryFilter: '',
    statusFilter: '',
    loading: true,
    saving: false,
    error: null,
    modalMode: null,
    form: defaultForm(),
    formErrors: {},
    detail: null,
    selectedId: null,
    qtyModal: null,
    confirm: null,
    toast: null,
  }
}

export function defaultForm() {
  return {
    id: null,
    title: '',
    isbn: '',
    author: '',
    category: '',
    publisher: '',
    publish_year: '',
    location: '',
    total_stock: 1,
  }
}

// normalizeRow 计算派生值 available_count，并按可借数量重算展示状态。
export function normalizeRow(row) {
  if (!row) return row
  var total = Number(row.total_stock) || 0
  var borrowed = Number(row.borrowed_count) || 0
  if (borrowed < 0) borrowed = 0
  var available = total - borrowed
  if (available < 0) available = 0
  row.available_count = available
  var status = row.status || 'onshelf'
  if (status !== 'offshelf') {
    status = available <= 0 ? 'lent' : 'onshelf'
  }
  row.status = status
  return row
}

// formToPayload 将表单值转换为提交载荷。
export function formToPayload(form) {
  var payload = {
    title: (form.title || '').trim(),
    isbn: (form.isbn || '').trim(),
    author: (form.author || '').trim(),
    category: form.category || '',
    publisher: (form.publisher || '').trim(),
    location: (form.location || '').trim(),
    total_stock: Number(form.total_stock) || 0,
  }
  var py = form.publish_year
  if (py === '' || py === null || py === undefined) {
    payload.publish_year = null
  } else {
    payload.publish_year = Number(py)
  }
  return payload
}

// validateBookForm 校验新建/编辑表单，返回字段错误对象。
export function validateBookForm(form) {
  var errors = {}
  var title = (form.title || '').trim()
  if (!title) {
    errors.title = '请输入书名'
  } else if (title.length > 120) {
    errors.title = '书名需在 1-120 字之间'
  }

  var isbn = (form.isbn || '').trim()
  if (!isbn) {
    errors.isbn = '请输入 ISBN'
  } else if (!isValidISBN(isbn)) {
    errors.isbn = 'ISBN 格式不正确'
  }

  if ((form.author || '').length > 80) errors.author = '作者需在 80 字以内'
  if ((form.publisher || '').length > 80) errors.publisher = '出版社需在 80 字以内'
  if ((form.location || '').length > 60) errors.location = '馆藏位置需在 60 字以内'

  var py = form.publish_year
  if (py !== '' && py !== null && py !== undefined) {
    var year = Number(py)
    if (isNaN(year) || year < 1000 || year > 2100) {
      errors.publish_year = '出版年份需在 1000-2100 之间'
    }
  }

  var stock = Number(form.total_stock)
  if (form.total_stock === '' || form.total_stock === null || form.total_stock === undefined || isNaN(stock) || stock < 1 || !Number.isInteger(stock)) {
    errors.total_stock = '总库存需为不小于 1 的整数'
  }

  return errors
}

// validateQuantity 校验借出/归还数量，limit 为上限。
export function validateQuantity(value, limit) {
  var qty = Number(value)
  if (value === '' || value === null || value === undefined || isNaN(qty) || qty < 1 || !Number.isInteger(qty)) {
    return '请输入不小于 1 的整数'
  }
  if (limit !== null && limit !== undefined && qty > limit) {
    return '超出可借数量'
  }
  return ''
}

export function setFormField(state, field, value) {
  state.form[field] = value
  if (state.formErrors[field]) {
    var errs = Object.assign({}, state.formErrors)
    delete errs[field]
    state.formErrors = errs
  }
}

export function setFormErrors(state, errors) {
  state.formErrors = errors || {}
}

export function setSaving(state, saving) {
  state.saving = saving
}

export function setError(state, error) {
  state.error = error
  state.loading = false
}

// isValidISBN 校验 ISBN-10 / ISBN-13（去除空格与连字符后按校验位判断）。
export function isValidISBN(value) {
  var v = String(value || '').replace(/[ \-]/g, '')
  if (v.length === 10) return isValidISBN10(v)
  if (v.length === 13) return isValidISBN13(v)
  return false
}

function isValidISBN10(v) {
  var sum = 0
  for (var i = 0; i < 10; i++) {
    var c = v[i]
    var d
    if (i === 9 && c === 'X') d = 10
    else if (c >= '0' && c <= '9') d = c.charCodeAt(0) - 48
    else return false
    sum += d * (10 - i)
  }
  return sum % 11 === 0
}

function isValidISBN13(v) {
  var sum = 0
  for (var i = 0; i < 13; i++) {
    var c = v[i]
    if (c < '0' || c > '9') return false
    var d = c.charCodeAt(0) - 48
    sum += i % 2 === 0 ? d : d * 3
  }
  return sum % 10 === 0
}
