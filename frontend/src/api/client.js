const API_BASE = import.meta.env.VITE_API_BASE || ''

export async function apiFetch(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) }
  const response = await fetch(`${API_BASE}${path}`, { ...options, headers })
  const text = await response.text()
  const data = text ? JSON.parse(text) : null
  if (!response.ok) {
    throw new Error(data?.error || `请求失败：${response.status}`)
  }
  return data
}

export function authTokenKey(role) {
  return `${role}_token`
}

export function authHeaders(role) {
  const token = localStorage.getItem(authTokenKey(role))
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export function unifiedLogin(role, payload = {}) {
  return apiFetch('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ role, ...payload })
  })
}

export function customerHeaders() {
  return authHeaders('customer')
}

export function merchantHeaders() {
  return authHeaders('merchant')
}

export function adminHeaders() {
  return authHeaders('admin')
}

export function money(cents = 0) {
  return `¥${(Number(cents) / 100).toFixed(2)}`
}

export function dateTime(value) {
  if (!value) return '未设置'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

export function statusText(status) {
  return {
    pending_accept: '待接单',
    accepted: '已接单',
    preparing: '备货中',
    ready: '可取货',
    completed: '已完成',
    rejected: '已拒单',
    canceled: '已取消',
    expired: '已过期',
    active: '营业中',
    ended: '已收摊'
  }[status] || status || '未知'
}
