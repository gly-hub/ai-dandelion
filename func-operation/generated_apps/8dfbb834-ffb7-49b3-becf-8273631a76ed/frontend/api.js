export function createApi(invoke) {
  async function call(action, data) {
    const result = await invoke({ action, data })
    if (!result || typeof result !== 'object') {
      return { success: false, error: '后端无响应，请确认 backend.wasm 已重建且功能已重新匹配' }
    }
    return result
  }

  return {
    teacherList(params) {
      return call('teacher_list', params)
    },
    teacherDetail(id) {
      return call('teacher_detail', { id })
    },
    teacherCreate(data) {
      return call('teacher_create', data)
    },
    teacherUpdate(id, data) {
      return call('teacher_update', Object.assign({ id: id }, data))
    },
    teacherChangeStatus(id, status) {
      return call('teacher_change_status', { id: id, status: status })
    },
    teacherDelete(id) {
      return call('teacher_delete', { id })
    },
  }
}
