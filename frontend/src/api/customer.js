import {
  authHeaders,
  clearAuthToken,
  hasValidTokenRole,
  merchantIdFrom,
  normalizeListWith,
  queryString,
  request,
  storeAuthToken
} from './client'

const favoriteIdByMerchantID = new Map()
const debugCustomerToken = String(import.meta.env.VITE_CUSTOMER_TOKEN || '').trim()
const isH5DevPreview = import.meta.env.DEV && typeof window !== 'undefined'
const PUB_API = '/api/pub'
const CUSTOMER_API = '/api/customer'
let customerLoginPromise = null

export async function ensureCustomerLogin(force = false) {
  if (debugCustomerToken) {
    clearAuthToken('customer')
    storeAuthToken('customer', { token: debugCustomerToken })
    return true
  }
  if (!force && hasValidTokenRole('customer')) return true
  if (!force && customerLoginPromise) return customerLoginPromise
  customerLoginPromise = loginCustomerWithMiniCode().finally(() => {
    customerLoginPromise = null
  })
  return customerLoginPromise
}

async function loginCustomerWithMiniCode() {
  clearAuthToken('customer')
  const code = await wxMiniLoginCode()
  const resp = await request('/api/Wx/mini', {
    method: 'POST',
    data: {
      code,
      nickname: '小程序顾客'
    }
  })
  storeAuthToken('customer', resp)
  return true
}

export function wxMiniLoginCode() {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: 'weixin',
      success: (res) => {
        if (res?.code) resolve(res.code)
        else reject(new Error('微信登录未返回 code'))
      },
      fail: (err) => reject(new Error(err?.errMsg || '微信登录失败'))
    })
  })
}

function jsonOptions(method, payload) {
  return { method, data: payload }
}

async function authedRequest(path, options = {}, retry = true) {
  await ensureCustomerLogin()
  try {
    return await request(path, { ...options, headers: { ...authHeaders('customer'), ...(options.headers || {}) } })
  } catch (error) {
    if (!retry || !isLoginRequiredError(error)) throw error
    if (debugCustomerToken) throw error
    clearAuthToken('customer')
    await ensureCustomerLogin(true)
    return authedRequest(path, options, false)
  }
}

function isLoginRequiredError(error) {
  const ui = error?.ui || error?.data?.ui || {}
  return Number(error?.status || error?.code || error?.data?.code) === 401
    && (ui.target === 'login' || /token|登录|login/i.test(error?.message || ''))
}

function rememberFavorite(favorite) {
  const merchantID = Number(favorite?.merchant_id ?? favorite?.merchantId ?? favorite?.shop_id ?? favorite?.shopId ?? favorite?.merchant?.id ?? favorite?.shop?.id)
  const favoriteID = Number(favorite?.favoriteId ?? favorite?.favorite_id ?? favorite?.id)
  if (merchantID && favoriteID) favoriteIdByMerchantID.set(merchantID, favoriteID)
  return favorite
}

function normalizeFavoriteItem(favorite = {}) {
  const vendor = normalizeMapItem({ ...favorite, favorite_id: favorite.id || favorite.favorite_id || favorite.favoriteId })
  vendor.favoriteId = favorite.id || favorite.favorite_id || favorite.favoriteId || vendor.favoriteId || null
  return rememberFavorite(vendor)
}

function normalizeMerchant(merchant = {}) {
  const id = merchant.id ?? merchant.merchant_id ?? merchant.shop_id
  const name = merchant.display_name || merchant.name || merchant.shop_name || ''
  return {
    ...merchant,
    id,
    merchant_id: merchant.merchant_id ?? id,
    shop_id: merchant.shop_id ?? id,
    display_name: name,
    name
  }
}

function normalizeShareDetail(resp = {}, code = '') {
  if (!resp || typeof resp !== 'object') return resp
  const merchant = normalizeMerchant(resp.merchant || resp.shop || resp)
  const share = resp.share || {}
  return {
    ...resp,
    merchant,
    shop: resp.shop || merchant,
    share: {
      ...share,
      code: share.code || merchant?.share_code || code,
      url: share.url || merchant?.share_url || '',
      merchant_id: share.merchant_id ?? share.shop_id ?? merchant?.id,
      shop_id: share.shop_id ?? share.merchant_id ?? merchant?.id
    }
  }
}

export function normalizeMapItem(item = {}, index = 0) {
  const merchant = normalizeMerchant(item.merchant || item.shop || {})
  const session = item.stall_session || item.session || {}
  const merchantId = merchant.id ?? item.merchant_id ?? item.shop_id
  const productsLoaded = Object.prototype.hasOwnProperty.call(item, 'products') || Object.prototype.hasOwnProperty.call(merchant, 'products')
  const products = (item.products || merchant.products || []).map(normalizeProduct)
  const productsTotalRaw = item.products_total ?? item.productsTotal ?? item.product_total ?? item.productTotal ?? item.products_count ?? item.productsCount
  const productsTotal = productsTotalRaw === undefined || productsTotalRaw === null ? null : Number(productsTotalRaw)
  const productsCompleteRaw = item.products_complete ?? item.productsComplete
  return {
    raw: item,
    id: String(merchantId || index),
    merchantId,
    favoriteId: item.favorite_id || item.favoriteId || null,
    name: merchant.display_name || item.display_name || '流动摊位',
    category: merchant.category || item.category || '其他摊位',
    address: session.address || merchant.announcement || item.address || '摊主当前位置',
    area: session.address || item.area || '摊主当前位置',
    lat: Number(session.lat || item.lat || 0),
    lng: Number(session.lng || item.lng || 0),
    isOpen: item.display_status !== 'unavailable' && item.display_status !== 'recent' && Boolean(session.id || session.lat || item.lat),
    statusText: item.display_status === 'recent' ? '最近在线' : (session.id || session.lat || item.lat) ? '营业中' : '未出摊',
    endText: session.expected_end_at ? formatTime(session.expected_end_at) : '',
    photoUrl: session.photo_url || session.photoURL || merchant.photo_url || '',
    avatarUrl: merchant.avatar_url || merchant.avatarURL || '',
    distanceMeters: normalizedNumber(item.distance_meters ?? item.distanceMeters),
    walkMinutes: normalizedNumber(item.walk_minutes ?? item.walkMinutes),
    products,
    productsTotal,
    productsLoaded,
    productsComplete: productsCompleteRaw === undefined ? null : Boolean(productsCompleteRaw)
  }
}


export function normalizePageState(resp = {}) {
  const pageMode = resp.page_mode || resp.pageMode || (resp.is_merchant || resp.isMerchant ? 'merchant' : (resp.has_application || resp.hasApplication ? 'application' : 'customer'))
  return {
    ...resp,
    user_id: resp.user_id ?? resp.userId,
    page_mode: pageMode,
    next_action: resp.next_action || resp.nextAction || (pageMode === 'merchant' ? 'dashboard' : pageMode === 'application' ? 'application_pending' : 'create_application'),
    is_merchant: Boolean(resp.is_merchant ?? resp.isMerchant ?? pageMode === 'merchant'),
    has_application: Boolean(resp.has_application ?? resp.hasApplication ?? pageMode === 'application'),
    application_id: resp.application_id ?? resp.applicationId ?? '',
    application_status: resp.application_status || resp.applicationStatus || '',
    merchant_id: resp.merchant_id ?? resp.merchantId ?? null
  }
}

export function normalizeProduct(product = {}) {
  return {
    ...product,
    id: product.id ?? product.product_id,
    name: product.name || product.product_name || '商品',
    price_cents: product.price_cents ?? product.priceCents ?? Number(product.price || 0) * 100,
    stock: Number(product.stock ?? product.remaining_stock ?? 99),
    image_url: firstImage(product.image_url || product.imageUrl || product.image || product.image_urls || product.images),
    pinned_at: product.pinned_at || product.pinnedAt || null
  }
}

function normalizedNumber(value) {
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function formatTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function firstImage(value) {
  if (!value) return ''
  if (Array.isArray(value)) return value.find(Boolean) || ''
  const text = String(value || '').trim()
  if (!text) return ''
  if (text.startsWith('[')) {
    try {
      const parsed = JSON.parse(text)
      if (Array.isArray(parsed)) return parsed.find(Boolean) || ''
    } catch {}
  }
  return text.split(/[\n|,，]+/).map((item) => item.trim()).find(Boolean) || ''
}

function favoritePayload(payload = {}) {
  return { merchant_id: payload.merchant_id ?? payload.shop_id }
}

function favoriteQuery(params = {}) {
  const next = { ...(params || {}) }
  const merchantID = merchantIdFrom(next)
  if (merchantID) next.merchant_id = merchantID
  delete next.id
  delete next.merchantId
  delete next.shop_id
  delete next.shopId
  delete next.merchant_code
  delete next.merchantCode
  if (typeof next.name === 'string') {
    next.name = next.name.trim()
    if (!next.name) delete next.name
  }
  return next
}

function uniqueMapList(resp) {
  const list = normalizeListWith(resp, normalizeMapItem)
  const seen = new Set()
  const data = []
  list.data.forEach((item) => {
    const key = item.merchantId || item.id
    if (key && seen.has(key)) return
    if (key) seen.add(key)
    data.push(item)
  })
  return { ...list, data, total: data.length }
}

export const customerApi = {
  async getPageState() {
    return authedRequest(`${CUSTOMER_API}/me`).then(normalizePageState)
  },
  async nearbyStalls(params = {}) {
    return request(`${PUB_API}/stalls/nearby${queryString(params)}`)
      .then(uniqueMapList)
      .catch((error) => {
        if (isH5DevPreview && debugCustomerToken && [401, 403].includes(error.status)) return uniqueMapList(mockNearbyStalls())
        throw error
      })
  },
  async getShare(code) {
    return request(`${PUB_API}/merchants/detail${queryString({ share_code: code })}`)
      .then((resp) => normalizeShareDetail(resp, code))
  },
  async getMerchantMapState(merchantID, params = {}) {
    const id = merchantIdFrom(merchantID)
    return request(`${PUB_API}/merchants/${encodeURIComponent(id)}/map-state${queryString(params)}`).then(normalizeMapItem)
  },
  async listProducts(merchantRef, params = {}) {
    const id = merchantIdFrom(merchantRef || params)
    if (!id) return { data: [], total: 0 }
    return request(`${PUB_API}/merchants/products${queryString({ ...params, id })}`)
      .then((resp) => normalizeListWith(resp, normalizeProduct, ['products']))
  },
  async listFavorites(params = {}) {
    return authedRequest(`${CUSTOMER_API}/favorites${queryString(favoriteQuery(params))}`)
      .then((resp) => normalizeListWith(resp, normalizeFavoriteItem, ['favorites', 'stalls']))
  },
  async addFavorite(payload) {
    return authedRequest(`${CUSTOMER_API}/favorites`, jsonOptions('POST', favoritePayload(payload)))
  },
  async removeFavorite(id) {
    return authedRequest(`${CUSTOMER_API}/favorites${queryString({ id })}`, jsonOptions('DELETE'))
  },
  async createOrder(payload) {
    throw new Error('订单/预定接口当前未启用')
  },
  async getApplication() {
    return authedRequest(`${CUSTOMER_API}/applications`)
  },
  async createApplication(payload) {
    return authedRequest(`${CUSTOMER_API}/applications`, jsonOptions('PUT', payload))
  },
  async createFeedback(payload) {
    return authedRequest(`${CUSTOMER_API}/feedback`, jsonOptions('POST', payload))
  }
}

function mockNearbyStalls() {
  const rows = [
    ['pancake-demo', '阿强流动煎饼铺', '早餐小吃', '旺角地铁站 A1 出口', 1, 'https://images.unsplash.com/photo-1514933651103-005eec06c04b?auto=format&fit=crop&w=900&q=80', [['酸梅汤', 800], ['冰豆浆', 600]]],
    ['fruit-demo', '甜橙鲜切水果摊', '水果鲜切', '社区南门', 93, 'https://images.unsplash.com/photo-1490474418585-ba9bad8fd0ea?auto=format&fit=crop&w=900&q=80', [['西瓜杯', 1200], ['芒果酸奶杯', 1800]]],
    ['dumpling-demo', '胖姐手工水饺', '早餐小吃', '旺角街市口', 122, 'https://images.unsplash.com/photo-1496116218417-1a781b1c416c?auto=format&fit=crop&w=900&q=80', [['紫菜蛋花汤', 600], ['韭菜鸡蛋煎饺', 1600]]],
    ['burger-demo', '迷你汉堡研究所', '便当快餐', '宏达金属建材门前', 124, 'https://images.unsplash.com/photo-1550547660-d9450f859349?auto=format&fit=crop&w=900&q=80', [['薯条杯', 1200], ['鸡排堡', 2200]]],
    ['coffee-demo', '青榕手冲咖啡车', '咖啡饮品', '创意园东门', 137, 'https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=900&q=80', [['燕麦拿铁', 2600], ['拿铁', 2200]]],
    ['lunch-demo', '阿兰便当快餐', '便当快餐', '福民大厦西侧', 158, 'https://images.unsplash.com/photo-1543353071-873f17a7a088?auto=format&fit=crop&w=900&q=80', [['素菜双拼饭', 2200], ['番茄牛腩饭', 3200]]]
  ]
  return {
    data: rows.map(([code, name, category, address, distance, image, products], index) => ({
      merchant: { id: index + 1, display_name: name, category },
      stall_session: { id: index + 10, lat: 22.3193, lng: 114.1694, address, photo_url: image },
      products: products.map(([productName, price], productIndex) => ({ id: `${code}-${productIndex}`, name: productName, price_cents: price, stock: 99 })),
      distance_meters: distance,
      walk_minutes: Math.max(1, Math.ceil(distance / 80)),
      display_status: 'active'
    })),
    total: rows.length
  }
}
