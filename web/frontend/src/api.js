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
