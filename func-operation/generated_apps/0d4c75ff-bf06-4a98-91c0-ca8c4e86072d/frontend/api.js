export function createApi(invoke) {
  async function call(action, data) {
    const result = await invoke({ action, data })
    if (!result || typeof result !== 'object') {
      return { success: false, error: '后端无响应，请确认 backend.wasm 已重建且功能已重新匹配' }
    }
    return result
  }

  return {
    listBooks(params) {
      return call('list', params)
    },
    getBook(id) {
      return call('detail', { id })
    },
    createBook(data) {
      return call('create', data)
    },
    updateBook(id, data) {
      return call('update', Object.assign({ id: id }, data))
    },
    deleteBook(id) {
      return call('delete', { id })
    },
    lendBook(id, quantity) {
      return call('lend', { id: id, quantity: quantity })
    },
    returnBook(id, quantity) {
      return call('return_book', { id: id, quantity: quantity })
    },
    offshelfBook(id) {
      return call('offshelf', { id })
    },
    onshelfBook(id) {
      return call('onshelf', { id })
    },
  }
}
