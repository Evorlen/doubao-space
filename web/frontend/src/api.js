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
