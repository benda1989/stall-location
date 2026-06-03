const API_BASE = import.meta.env.VITE_API_BASE || ''

export async function apiFetch(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) }
  const response = await fetch(`${API_BASE}${path}`, { ...options, headers })
  const text = await response.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = { error: text }
    }
  }
  if (!response.ok) {
    const error = new Error(data?.error || `请求失败：${response.status}`)
    error.status = response.status
    error.data = data
    throw error
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

export function readTokenClaims(role) {
  const token = localStorage.getItem(authTokenKey(role))
  if (!token) return null
  const parts = token.split('.')
  if (parts.length < 2) return null
  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const json = decodeURIComponent(Array.from(atob(base64), (char) => `%${char.charCodeAt(0).toString(16).padStart(2, '0')}`).join(''))
    return JSON.parse(json)
  } catch {
    return null
  }
}

export function hasValidTokenRole(role) {
  const claims = readTokenClaims(role)
  if (!claims || claims.role !== role) return false
  return !claims.exp || Number(claims.exp) * 1000 > Date.now()
}

export function clearAuthToken(role) {
  localStorage.removeItem(authTokenKey(role))
}

export function devCustomerOpenID() {
  const key = 'mplzDevOpenID'
  const saved = localStorage.getItem(key)
  if (saved) return saved
  const random = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`
  const openID = `dev-customer-${random}`
  localStorage.setItem(key, openID)
  return openID
}

export function unifiedLogin(role, payload = {}) {
  return apiFetch('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ role, ...payload })
  })
}

export function customerHeaders() {
  const headers = {}
  if (hasValidTokenRole('customer')) {
    Object.assign(headers, authHeaders('customer'))
  } else {
    clearAuthToken('customer')
  }
  if (import.meta.env.DEV) {
    headers['X-Dev-Customer'] = '1'
    headers['X-Dev-OpenID'] = devCustomerOpenID()
  }
  return headers
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
