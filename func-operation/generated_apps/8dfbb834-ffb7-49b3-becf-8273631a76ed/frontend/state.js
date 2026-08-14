export const STATUS_OPTIONS = [
  { value: 'active', label: '在职', cls: 'ga-teacher-badge-active' },
  { value: 'suspended', label: '停用', cls: 'ga-teacher-badge-suspended' },
  { value: 'resigned', label: '离职', cls: 'ga-teacher-badge-resigned' },
]

export const GENDER_OPTIONS = [
  { value: 'male', label: '男' },
  { value: 'female', label: '女' },
]

export const EDUCATION_OPTIONS = [
  { value: 'bachelor', label: '本科' },
  { value: 'master', label: '硕士' },
  { value: 'doctor', label: '博士' },
  { value: 'other', label: '其他' },
]

export function statusLabel(status) {
  const opt = STATUS_OPTIONS.find(function (o) { return o.value === status })
  return opt ? opt.label : (status || '')
}

export function statusClass(status) {
  const opt = STATUS_OPTIONS.find(function (o) { return o.value === status })
  return opt ? opt.cls : 'ga-teacher-badge-active'
}

export function genderLabel(gender) {
  const opt = GENDER_OPTIONS.find(function (o) { return o.value === gender })
  return opt ? opt.label : (gender || '')
}

export function educationLabel(education) {
  const opt = EDUCATION_OPTIONS.find(function (o) { return o.value === education })
  return opt ? opt.label : (education || '')
}

export function countryLabel(countryOptions, value) {
  if (!value) return ''
  const opt = (countryOptions || []).find(function (o) { return o.value === value })
  return opt ? opt.label : value
}

export function createInitialState() {
  return {
    rows: [],
    total: 0,
    page: 1,
    pageSize: 10,
    keyword: '',
    countryFilter: '',
    departmentFilter: '',
    statusFilter: '',
    educationFilter: '',
    titleFilter: '',
    departmentOptions: [],
    titleOptions: [],
    countryOptions: [],
    countryOptionsError: null,
    countryOptionsLoading: false,
    loading: true,
    saving: false,
    error: null,
    modalMode: null,
    form: defaultForm(),
    formErrors: {},
    detail: null,
    detailError: null,
    selectedId: null,
    confirm: null,
  }
}

export function defaultForm() {
  return {
    id: null,
    name: '',
    employee_no: '',
    country: '',
    gender: 'male',
    birth_date: '',
    education: '',
    department: '',
    title: '',
    phone: '',
    email: '',
    hire_date: '',
  }
}

// normalizeRow 补充默认性别，保持后端行数据在前后端一致。
export function normalizeRow(row) {
  if (!row) return row
  if (!row.gender) row.gender = 'male'
  return row
}

// mergeOptions 合并院系/职称筛选项，避免切换筛选时下拉选项收缩。
export function mergeOptions(current, incoming) {
  var map = {}
  ;(current || []).forEach(function (v) { if (v) map[v] = true })
  ;(incoming || []).forEach(function (v) { if (v) map[v] = true })
  return Object.keys(map).sort()
}

// formToPayload 将表单值转换为提交载荷。
export function formToPayload(form) {
  return {
    name: (form.name || '').trim(),
    employee_no: (form.employee_no || '').trim(),
    country: (form.country || '').trim(),
    gender: form.gender || 'male',
    birth_date: (form.birth_date || '').trim(),
    education: form.education || '',
    department: (form.department || '').trim(),
    title: (form.title || '').trim(),
    phone: (form.phone || '').trim(),
    email: (form.email || '').trim(),
    hire_date: (form.hire_date || '').trim(),
  }
}

// validateTeacherForm 校验新建/编辑表单，返回字段错误对象。
// countryOptions 为全局配置键 country 的选项数组（可缺省）；
// 配置加载失败或未加载时 countryOptions 为空数组，此时不校验国籍值合法性，
// 保证国籍作为可选字段不阻断其余字段保存。
export function validateTeacherForm(form, countryOptions) {
  var errors = {}

  var name = (form.name || '').trim()
  if (!name) {
    errors.name = '请输入姓名'
  } else if (name.length > 50) {
    errors.name = '姓名需在 1-50 字之间'
  }

  var employeeNo = (form.employee_no || '').trim()
  if (!employeeNo) {
    errors.employee_no = '请输入工号'
  } else if (!/^[A-Za-z0-9]{3,30}$/.test(employeeNo)) {
    errors.employee_no = '工号需为 3-30 位字母或数字'
  }

  var country = (form.country || '').trim()
  if (country.length > 50) {
    errors.country = '国籍需在 50 字以内'
  } else if (country && Array.isArray(countryOptions) && countryOptions.length > 0) {
    var matched = countryOptions.some(function (o) { return o && o.value === country })
    if (!matched) {
      errors.country = '请选择正确的国籍'
    }
  }

  var department = (form.department || '').trim()
  if (!department) {
    errors.department = '请输入所属院系'
  } else if (department.length > 50) {
    errors.department = '所属院系需在 1-50 字之间'
  }

  var phone = (form.phone || '').trim()
  if (!phone) {
    errors.phone = '请输入联系电话'
  } else if (!isValidPhone(phone)) {
    errors.phone = '请输入正确的联系电话'
  }

  var email = (form.email || '').trim()
  if (email && email.length > 100) {
    errors.email = '邮箱需在 100 字以内'
  } else if (email && !isValidEmail(email)) {
    errors.email = '请输入正确的电子邮箱'
  }

  if ((form.title || '').length > 50) {
    errors.title = '职称需在 50 字以内'
  }

  if (form.birth_date && !isValidDate(form.birth_date)) {
    errors.birth_date = '请输入正确的日期'
  }
  if (form.hire_date && !isValidDate(form.hire_date)) {
    errors.hire_date = '请输入正确的日期'
  }

  if (form.gender && ['male', 'female'].indexOf(form.gender) < 0) {
    errors.gender = '请选择正确的性别'
  }
  if (form.education && ['bachelor', 'master', 'doctor', 'other'].indexOf(form.education) < 0) {
    errors.education = '请选择正确的学历'
  }

  return errors
}

function isValidPhone(value) {
  var digits = String(value || '').replace(/[^0-9]/g, '')
  return digits.length >= 7 && digits.length <= 15
}

function isValidEmail(value) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
}

function isValidDate(value) {
  var m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (!m) return false
  var y = Number(m[1])
  var mo = Number(m[2])
  var d = Number(m[3])
  if (mo < 1 || mo > 12) return false
  var days = new Date(y, mo, 0).getDate()
  return d >= 1 && d <= days
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

// normalizeCountryResult 将 context.config.get('country') 的返回值归一化为 { options, error }。
// 兼容以下返回形态：
//   - 数组 [{ value, label }]（直接为选项数组）
//   - 对象 { country: [{ value, label }] }（按配置键返回）
//   - 数组外层包裹 { data: [...] } 或 { country: [...] }
function normalizeCountryResult(value) {
  if (value === null || value === undefined) {
    return { options: [], error: '选项加载失败' }
  }
  var arr = null
  if (Array.isArray(value)) {
    arr = value
  } else if (typeof value === 'object') {
    if (Array.isArray(value.country)) {
      arr = value.country
    } else if (Array.isArray(value.data)) {
      arr = value.data
    } else if (Array.isArray(value.value)) {
      arr = value.value
    }
  }
  if (!Array.isArray(arr)) {
    return { options: [], error: '选项加载失败' }
  }
  var options = arr
    .filter(function (o) { return o && o.value !== undefined && o.value !== null })
    .map(function (o) {
      return { value: String(o.value), label: (o.label !== undefined && o.label !== null) ? String(o.label) : String(o.value) }
    })
  return { options: options, error: '' }
}

// loadCountryOptions 读取全局配置键 country 的选项列表。
// 返回 Promise<{ options: [], error: '' | '选项加载失败' }>，由调用方写入 state 并重渲染。
export function loadCountryOptions(context) {
  return new Promise(function (resolve) {
    var getter = null
    if (context && context.config && typeof context.config.get === 'function') {
      getter = function (key) { return context.config.get(key) }
    } else if (context && typeof context.config === 'function') {
      getter = function (key) { return context.config(key) }
    }
    if (!getter) {
      resolve({ options: [], error: '选项加载失败' })
      return
    }
    var result
    try {
      result = getter('country')
    } catch (e) {
      resolve({ options: [], error: '选项加载失败' })
      return
    }
    if (result && typeof result.then === 'function') {
      result.then(
        function (value) { resolve(normalizeCountryResult(value)) },
        function () { resolve({ options: [], error: '选项加载失败' }) }
      )
    } else {
      resolve(normalizeCountryResult(result))
    }
  })
}
