export const API_BASE = String(import.meta.env.VITE_API_BASE || '').replace(/\/$/, '')
const JSON_CONTENT_TYPE = 'application/json'

export function apiURL(path) {
  const value = String(path || '')
  if (/^https?:\/\//i.test(value)) return value
  return `${API_BASE}${value}`
}

export function request(path, options = {}) {
  const method = options.method || 'GET'
  const url = apiURL(path)
  const headers = { ...(options.headers || {}) }
  const data = options.data === undefined ? options.body : options.data
  if (data !== undefined && headers['Content-Type'] === undefined && headers['content-type'] === undefined) {
    headers['Content-Type'] = JSON_CONTENT_TYPE
  }

  if (!hasUniRuntime()) {
    return fetchRequest(url, { ...options, method, data, headers })
  }

  return new Promise((resolve, reject) => {
    uni.request({
      url,
      method,
      data,
      header: headers,
      timeout: options.timeout || 15000,
      success: (res) => {
        const status = Number(res.statusCode || 0)
        const body = normalizeResponseData(res.data)
        if (status < 200 || status >= 300) {
          reject(buildApiError(status, body))
          return
        }
        resolve(body)
      },
      fail: (err) => reject(new Error(formatRequestFail(err, url)))
    })
  })
}

function hasUniRuntime() {
  return typeof uni !== 'undefined' && typeof uni.request === 'function'
}

async function fetchRequest(url, options = {}) {
  const headers = { ...(options.headers || {}) }
  const data = options.data === undefined ? options.body : options.data
  const init = {
    method: options.method || 'GET',
    headers
  }
  if (data !== undefined && init.method !== 'GET') {
    init.body = typeof data === 'string' || data instanceof FormData ? data : JSON.stringify(data)
  }
  const response = await fetch(url, init)
  const text = await response.text()
  const body = normalizeResponseData(text)
  if (!response.ok) throw buildApiError(response.status, body)
  return body
}

export function apiFetch(path, options = {}) {
  return request(path, options)
}

export function uploadFile(filePath, role = 'customer', options = {}) {
  return new Promise((resolve, reject) => {
    uni.uploadFile({
      url: apiURL('/api/upload'),
      filePath,
      name: 'file',
      header: authHeaders(role),
      formData: uploadFormData(options),
      timeout: 30000,
      success: (res) => {
        const status = Number(res.statusCode || 0)
        const body = normalizeResponseData(res.data)
        if (status < 200 || status >= 300) {
          reject(buildApiError(status, body))
          return
        }
        resolve(normalizeUploadURL(body?.url || body?.path || body?.message || body?.data?.url || body?.data?.path || body?.file?.url || body))
      },
      fail: (err) => reject(new Error(formatRequestFail(err, apiURL('/api/upload'))))
    })
  })
}

export function chooseImageFilePath() {
  return new Promise((resolve, reject) => {
    uni.chooseImage({
      count: 1,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
      success: (res) => {
        const path = res?.tempFilePaths?.[0]
        if (path) resolve(path)
        else reject(new Error('未选择图片'))
      },
      fail: (err) => reject(new Error(err?.errMsg || '选择图片失败'))
    })
  })
}

export async function chooseAndUploadImage(role = 'customer', options = {}) {
  const filePath = await chooseImageFilePath()
  return uploadFile(filePath, role, options)
}

function uploadFormData(options = {}) {
  const preset = String(options.preset || options.scene || options.usage || '').trim()
  return preset ? { preset } : {}
}

function normalizeUploadURL(value) {
  const url = String(value || '').trim()
  if (!url) return ''
  if (/^https?:\/\//i.test(url) || url.startsWith('data:image')) return url
  if (url.startsWith('/')) return apiURL(url)
  return url
}

function formatRequestFail(err, url) {
  const raw = err?.errMsg || '网络请求失败'
  if (/timeout/i.test(raw)) return `请求超时：${url}`
  return `${raw}：${url}`
}

function normalizeResponseData(data) {
  if (data === '' || data === undefined) return null
  if (typeof data !== 'string') return data
  try {
    return JSON.parse(data)
  } catch {
    return { message: data }
  }
}

function buildApiError(status, data) {
  const error = new Error(data?.message || data?.error || data?.msg || `请求失败：${status}`)
  error.status = status
  error.code = data?.code || ''
  error.ui = data?.ui || null
  error.data = data
  return error
}

export function queryString(params = {}) {
  const items = []
  Object.entries(params || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    if (Array.isArray(value)) {
      value.forEach((item) => {
        if (item !== undefined && item !== null && item !== '') items.push(`${encodeURIComponent(key)}=${encodeURIComponent(item)}`)
      })
      return
    }
    items.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
  })
  return items.length ? `?${items.join('&')}` : ''
}

export function getStorage(key, fallback = '') {
  try {
    const value = uni.getStorageSync(key)
    return value === undefined || value === null || value === '' ? fallback : value
  } catch {}
  try {
    const value = localStorage.getItem(key)
    return value === undefined || value === null || value === '' ? fallback : value
  } catch {}
  return fallback
}

export function setStorage(key, value) {
  try {
    uni.setStorageSync(key, value)
    return
  } catch {}
  try { localStorage.setItem(key, value) } catch {}
}

export function removeStorage(key) {
  try { uni.removeStorageSync(key) } catch {}
  try { localStorage.removeItem(key) } catch {}
}

export function getJSONStorage(key, fallback = null) {
  const raw = getStorage(key, '')
  if (!raw) return fallback
  if (typeof raw === 'object') return raw
  try { return JSON.parse(raw) } catch { return fallback }
}

export function setJSONStorage(key, value) {
  setStorage(key, JSON.stringify(value))
}

export function authTokenKey(role) {
  return `${role}_token`
}

export function authHeaders(role = 'customer') {
  const token = getStorage(authTokenKey(role), '')
  return token ? { token } : {}
}

export function adminHeaders() {
  const headers = authHeaders('admin')
  if (headers.token) headers.Authorization = `Bearer ${headers.token}`
  return headers
}

export function storeAuthToken(role, resp = {}) {
  const token = resp?.token || resp?.access_token || resp?.data?.token || resp?.data?.access_token || ''
  if (token) setStorage(authTokenKey(role), token)
  return token
}

export function clearAuthToken(role = 'customer') {
  removeStorage(authTokenKey(role))
}

export async function unifiedLogin(role, payload = {}) {
  const adminSmsLogin = role === 'admin' && payload.phone && !payload.password
  const loginPath = role === 'admin' ? (adminSmsLogin ? '/api/sys/sms' : '/api/sys/pwd') : '/api/auth/login'
  const data = role === 'admin'
    ? adminSmsLogin
      ? { phone: payload.phone, code: payload.code || '' }
      : {
          account: payload.account || payload.username || payload.phone || '',
          password: payload.password || '',
          code: payload.code || '123456'
        }
    : { role, ...payload }
  const resp = await request(loginPath, { method: 'POST', data })
  storeAuthToken(role, resp)
  return resp
}

export function readTokenClaims(role = 'customer') {
  const token = getStorage(authTokenKey(role), '')
  if (!token) return null
  const parts = token.split('.')
  if (parts.length < 2) return null
  try {
    let base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    while (base64.length % 4) base64 += '='
    const json = decodeURIComponent(Array.from(atobCompat(base64), (char) => `%${char.charCodeAt(0).toString(16).padStart(2, '0')}`).join(''))
    return JSON.parse(json)
  } catch {
    return null
  }
}

function atobCompat(base64) {
  if (typeof atob === 'function') return atob(base64)
  return decodeBase64ToBinaryString(base64)
}

function decodeBase64ToBinaryString(base64) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
  const input = String(base64 || '').replace(/=+$/, '')
  let buffer = 0
  let bits = 0
  let output = ''
  for (let i = 0; i < input.length; i += 1) {
    const value = chars.indexOf(input[i])
    if (value < 0) continue
    buffer = (buffer << 6) | value
    bits += 6
    if (bits >= 8) {
      bits -= 8
      output += String.fromCharCode((buffer >> bits) & 0xff)
    }
  }
  return output
}

export function hasValidTokenRole(role = 'customer') {
  const token = getStorage(authTokenKey(role), '')
  if (!token) return false
  const claims = readTokenClaims(role)
  if (!claims) return false
  const claimRole = claims.role || claims.kind || claims.identity || claims.Type || claims.type
  if (claimRole && !isCompatibleTokenRole(role, claimRole)) return false
  const expire = Number(claims.exp || claims.Expire || claims.expire || 0)
  return !expire || expire * 1000 > Date.now()
}

function isCompatibleTokenRole(role, claimRole) {
  const value = String(claimRole || '')
  if (value === role) return true
  return role === 'customer' && ['mini', 'front', 'user'].includes(value)
}

export function money(cents = 0) {
  const value = Number(cents || 0) / 100
  return `¥${value.toFixed(value % 1 ? 1 : 0)}`
}

export function dateTime(value) {
  if (!value) return '未设置'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

export function listData(resp, fallbackKeys = []) {
  if (Array.isArray(resp)) return resp
  if (Array.isArray(resp?.data)) return resp.data
  for (const key of fallbackKeys) if (Array.isArray(resp?.[key])) return resp[key]
  return []
}

export function normalizeList(resp, fallbackKeys = []) {
  const data = listData(resp, fallbackKeys)
  return { ...(resp || {}), data, total: Number(resp?.total ?? resp?.count ?? data.length) }
}

export function normalizeListWith(resp, normalizer, fallbackKeys = []) {
  const list = normalizeList(resp, fallbackKeys)
  return { ...list, data: list.data.map(normalizer) }
}

export function merchantIdFrom(value) {
  if (value && typeof value === 'object') return value.id || value.merchant_id || value.merchantId || value.shop_id || value.shopId || ''
  return value || ''
}

export function withMerchantID(params = {}, id) {
  const next = { ...(params || {}) }
  const merchantID = merchantIdFrom(id || next)
  if (merchantID) next.merchant_id = merchantID
  delete next.merchantId
  delete next.shop_id
  delete next.shopId
  return next
}
