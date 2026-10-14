async function request(path, options = {}) {
  const response = await fetch(path, {
    headers: {
      Accept: 'application/json',
      ...options.headers,
    },
    ...options,
  })

  let payload = null
  try {
    payload = await response.json()
  } catch {
    payload = null
  }

  if (!response.ok) {
    throw new Error(payload?.error || `请求失败：${response.status}`)
  }

  return payload
}

export function getHealth() {
  return request('/api/health')
}

export function getMeta() {
  return request('/api/meta')
}

export function getTasks(options = {}) {
  const params = new URLSearchParams()
  if (options.includeCompleted) params.set('include_completed', 'true')
  if (options.search) params.set('search', options.search)
  if (options.status) params.set('status', options.status)
  if (options.priority) params.set('priority', options.priority)
  if (options.tag) params.set('tag', options.tag)
  if (options.sort) params.set('sort', options.sort)
  const query = params.toString()
  return request(`/api/tasks${query ? `?${query}` : ''}`)
}

export function getCalendarEvents(options = {}) {
  const params = new URLSearchParams()
  if (options.start) params.set('start', options.start)
  if (options.end) params.set('end', options.end)
  if (options.sources?.length) params.set('sources', options.sources.join(','))
  if (options.includeCompleted) params.set('include_completed', 'true')
  const query = params.toString()
  return request(`/api/calendar${query ? `?${query}` : ''}`)
}

export function getTask(id) {
  return request(`/api/tasks/${id}`)
}

export function createTask(task) {
  return request('/api/tasks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(task),
  })
}

export function previewTaskImport(taskId, file) {
  const formData = new FormData()
  formData.append('file', file)
  return request(`/api/tasks/${taskId}/import/preview`, {
    method: 'POST',
    body: formData,
  })
}

export function importTasks(taskId, file) {
  const formData = new FormData()
  formData.append('file', file)
  return request(`/api/tasks/${taskId}/import`, {
    method: 'POST',
    body: formData,
  })
}

export function updateTask(id, task) {
  return request(`/api/tasks/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(task),
  })
}

export function completeTask(id) {
  return request(`/api/tasks/${id}/complete`, { method: 'POST' })
}

export function scheduleTask(id) {
  return request(`/api/tasks/${id}/schedule`, { method: 'POST' })
}

export function unscheduleTask(id) {
  return request(`/api/tasks/${id}/schedule`, { method: 'DELETE' })
}

export function moveTask(id, parentId) {
  return request(`/api/tasks/${id}/move`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ parent_id: parentId ?? null }),
  })
}

export function deleteTask(id) {
  return request(`/api/tasks/${id}`, { method: 'DELETE' })
}

export function uploadAttachment(taskId, file) {
  const formData = new FormData()
  formData.append('file', file)
  return request(`/api/tasks/${taskId}/attachments`, {
    method: 'POST',
    body: formData,
  })
}

export function deleteAttachment(taskId, attachmentId) {
  return request(`/api/tasks/${taskId}/attachments/${attachmentId}`, { method: 'DELETE' })
}

export function getTags() {
  return request('/api/tags')
}

export function createTag(name) {
  return request('/api/tags', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
}

export function getRecipe() {
  return request('/api/recipes')
}

export function reorderRecipe(order) {
  return request('/api/recipes/order', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(order),
  })
}

export function createRecipeFood(food) {
  return request('/api/recipes/foods', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(food),
  })
}

export function updateRecipeFood(id, food) {
  return request(`/api/recipes/foods/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(food),
  })
}

export function deleteRecipeFood(id) {
  return request(`/api/recipes/foods/${id}`, { method: 'DELETE' })
}

export function getDoubaoAlbum() {
  return request('/api/doubao/album')
}

export function uploadDoubaoAlbum(file, note = '') {
  const formData = new FormData()
  formData.append('file', file)
  if (note.trim()) formData.append('note', note.trim())
  return request('/api/doubao/album', {
    method: 'POST',
    body: formData,
  })
}

export function updateDoubaoAlbumItem(id, note) {
  return request(`/api/doubao/album/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note }),
  })
}

export function deleteDoubaoAlbumItem(id) {
  return request(`/api/doubao/album/${id}`, { method: 'DELETE' })
}

export function getSchedules() {
  return request('/api/schedules')
}

export function getSchedule(id) {
  return request(`/api/schedules/${id}`)
}

export function createSchedule(schedule) {
  return request('/api/schedules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(schedule),
  })
}

export function updateSchedule(id, schedule) {
  return request(`/api/schedules/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(schedule),
  })
}

export function completeSchedule(id) {
  return request(`/api/schedules/${id}/complete`, { method: 'POST' })
}

export function deleteSchedule(id) {
  return request(`/api/schedules/${id}`, { method: 'DELETE' })
}

export function getPlans() {
  return request('/api/plans')
}

export function getPlan(id) {
  return request(`/api/plans/${id}`)
}

export function createPlan(plan) {
  return request('/api/plans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(plan),
  })
}

export function updatePlan(id, plan) {
  return request(`/api/plans/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(plan),
  })
}

export function completePlan(id) {
  return request(`/api/plans/${id}/complete`, { method: 'POST' })
}

export function deletePlan(id) {
  return request(`/api/plans/${id}`, { method: 'DELETE' })
}

export function getEntertainment(options = {}) {
  const params = new URLSearchParams()
  if (options.type) params.set('type', options.type)
  if (options.status) params.set('status', options.status)
  if (options.search) params.set('search', options.search)
  if (options.rating) params.set('rating', options.rating)
  if (options.favorite) params.set('favorite', 'true')
  if (options.sort) params.set('sort', options.sort)
  const query = params.toString()
  return request(`/api/entertainment${query ? `?${query}` : ''}`)
}

export function getEntertainmentItem(id) {
  return request(`/api/entertainment/${id}`)
}

export function createEntertainmentItem(item) {
  return request('/api/entertainment', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(item),
  })
}

export function updateEntertainmentItem(id, item) {
  return request(`/api/entertainment/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(item),
  })
}

export function deleteEntertainmentItem(id) {
  return request(`/api/entertainment/${id}`, { method: 'DELETE' })
}

export function previewEntertainmentImport(file) {
  const body = new FormData()
  body.append('file', file)
  return request('/api/entertainment/import/preview', {
    method: 'POST',
    body,
  })
}

export function importEntertainmentItems(file) {
  const body = new FormData()
  body.append('file', file)
  return request('/api/entertainment/import', {
    method: 'POST',
    body,
  })
}

export function getMemos(options = {}) {
  const params = new URLSearchParams()
  if (typeof options.archived === 'boolean') params.set('archived', options.archived ? 'true' : 'false')
  if (options.search) params.set('search', options.search)
  const query = params.toString()
  return request(`/api/memos${query ? `?${query}` : ''}`)
}

export function getMemo(id) {
  return request(`/api/memos/${id}`)
}

export function createMemo(memo) {
  return request('/api/memos', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(memo),
  })
}

export function updateMemo(id, memo) {
  return request(`/api/memos/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(memo),
  })
}

export function archiveMemo(id) {
  return request(`/api/memos/${id}/archive`, { method: 'POST' })
}

export function restoreMemo(id) {
  return request(`/api/memos/${id}/restore`, { method: 'POST' })
}

export function deleteMemo(id) {
  return request(`/api/memos/${id}`, { method: 'DELETE' })
}
