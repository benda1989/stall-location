import { authHeaders, clearAuthToken, money, normalizeListWith, queryString, request } from './client'
import { ensureCustomerLogin, normalizeProduct } from './customer'

const CUSTOMER_API = '/api/customer'

function merchantHeaders() {
  return authHeaders('customer')
}

function jsonOptions(method, payload) {
  return { method, data: payload }
}

async function merchantRequest(path, options = {}, retry = true) {
  await ensureCustomerLogin()
  try {
    return await request(path, { ...options, headers: { ...merchantHeaders(), ...(options.headers || {}) } })
  } catch (error) {
    if (!retry || !isLoginRequiredError(error)) throw error
    clearAuthToken('customer')
    await ensureCustomerLogin(true)
    return merchantRequest(path, options, false)
  }
}

function isLoginRequiredError(error) {
  const ui = error?.ui || error?.data?.ui || {}
  return Number(error?.status || error?.code || error?.data?.code) === 401
    && (ui.target === 'login' || /token|登录|login/i.test(error?.message || ''))
}

export function normalizeApplication(application = {}) {
  return {
    ...application,
    id: application.id ?? application.ID,
    merchant_name: application.merchant_name || application.merchantName || '',
    contact_name: application.contact_name || application.contactName || '',
    contact_phone: application.contact_phone || application.contactPhone || '',
    category: application.category || '',
    photo_url: application.photo_url || application.photoUrl || '',
    usual_area: application.usual_area || application.usualArea || '',
    remark: application.remark || '',
    status: application.status || '',
    review_reason: application.review_reason || application.reviewReason || '',
    application_no: application.application_no || application.applicationNo || ''
  }
}

export function normalizeMerchant(merchant = {}) {
  return {
    ...merchant,
    id: merchant.id ?? merchant.merchant_id ?? merchant.shop_id,
    display_name: merchant.display_name || merchant.displayName || merchant.name || merchant.shop_name || '',
    category: merchant.category || '其他摊位',
    contact_phone: merchant.contact_phone || merchant.phone || '',
    avatar_url: merchant.avatar_url || merchant.avatarUrl || '',
    share_code: merchant.share_code || merchant.shareCode || '',
    share_url: merchant.share_url || merchant.shareUrl || '',
    share_poster_url: merchant.share_poster_url || merchant.sharePosterURL || merchant.sharePosterUrl || merchant.share_qrcode_url || merchant.shareQRCodeURL || merchant.shareQrcodeUrl || '',
    share_qrcode_url: merchant.share_qrcode_url || merchant.shareQRCodeURL || merchant.shareQrcodeUrl || merchant.share_poster_url || merchant.sharePosterURL || merchant.sharePosterUrl || '',
    share_qrcode_channel: merchant.share_qrcode_channel || merchant.shareQRCodeChannel || merchant.shareQrcodeChannel || '',
    share_page: merchant.share_page || merchant.sharePage || ''
  }
}

function normalizeStallSession(session = null) {
  if (!session || typeof session !== 'object') return null
  return {
    ...session,
    id: session.id ?? session.ID,
    merchant_id: session.merchant_id ?? session.merchantId ?? session.shop_id ?? session.shopId,
    status: session.status || '',
    lat: Number(session.lat || 0),
    lng: Number(session.lng || 0),
    address: session.address || '',
    expected_end_at: session.expected_end_at || session.expectedEndAt || '',
    photo_url: session.photo_url || session.photoUrl || '',
    location_accuracy: session.location_accuracy ?? session.locationAccuracy
  }
}

function normalizeDashboard(resp = {}) {
  const merchant = normalizeMerchant(resp.merchant || resp.shop || {})
  return {
    ...resp,
    merchant,
    shop: resp.shop || merchant,
    stall_session: normalizeStallSession(resp.stall_session || resp.stallSession)
  }
}

export function normalizeApplicationStatus(resp = {}) {
  const source = resp.application || (resp.id || resp.status || resp.merchant_name ? resp : null)
  const application = source ? normalizeApplication(source) : null
  const nextAction = resp.next_action || resp.nextAction || applicationNextAction(application?.status)
  return {
    ...resp,
    application,
    merchant: resp.merchant ? normalizeMerchant(resp.merchant) : null,
    next_action: nextAction
  }
}

function applicationNextAction(status = '') {
  if (status === 'approved') return 'dashboard'
  if (status === 'rejected') return 'application_rejected'
  if (status) return 'application_pending'
  return 'create_application'
}

function applicationPayload(payload = {}) {
  const next = {
    merchant_name: payload.merchant_name || '',
    contact_name: payload.contact_name || '',
    contact_phone: payload.contact_phone || '',
    category: payload.category || '',
    photo_url: payload.photo_url || '',
    usual_area: payload.usual_area || '',
    remark: payload.remark || ''
  }
  if (payload.id) next.id = payload.id
  return next
}

function productPayload(product = {}) {
  const payload = {
    name: product.name || '',
    description: product.description || '',
    price_cents: Number(product.price_cents ?? Math.round(Number(product.price || 0) * 100)),
    stock: Number(product.stock ?? 9999),
    image_url: product.image_url || '',
    status: product.status || 'on_sale',
    sort_order: Number(product.sort_order || 0)
  }
  if (product.id) payload.id = product.id
  return payload
}

export const merchantApi = {
  async getApplicationStatus() {
    return merchantRequest('/api/customer/applications').then(normalizeApplicationStatus)
  },
  async updateApplication(id, payload) {
    return merchantRequest(`${CUSTOMER_API}/applications`, jsonOptions('PUT', applicationPayload({ ...payload, id })))
  },
  async getDashboard() {
    return merchantRequest('/api/merchant/me').then((merchant) => normalizeDashboard({ merchant }))
  },
  async listProducts(params = {}) {
    return merchantRequest(`/api/merchant/products${queryString(params)}`).then((resp) => normalizeListWith(resp, normalizeProduct, ['products']))
  },
  async createProduct(payload) {
    return merchantRequest('/api/merchant/products', jsonOptions('PUT', productPayload(payload)))
  },
  async updateProduct(payload) {
    return merchantRequest('/api/merchant/products', jsonOptions('PUT', productPayload(payload)))
  },
  async deleteProduct(id) {
    return merchantRequest(`/api/merchant/products${queryString({ id })}`, jsonOptions('DELETE'))
  },
  async pinProduct(id) {
    return merchantRequest('/api/merchant/products/pin', jsonOptions('PUT', { id }))
  },
  async unpinProduct(id) {
    return merchantRequest('/api/merchant/products/unpin', jsonOptions('PUT', { id }))
  },
  async listStallSessions(params = {}) {
    return merchantRequest(`/api/merchant/stalls${queryString(params)}`).then((resp) => {
      if (Array.isArray(resp?.data)) return normalizeListWith(resp, normalizeStallSession, ['stall_sessions', 'sessions'])
      return normalizeStallSession(resp)
    })
  },
  async startStallSession(payload) {
    return merchantRequest('/api/merchant/stalls/start', jsonOptions('POST', payload))
  },
  async endStallSession() {
    return merchantRequest('/api/merchant/stalls/end', jsonOptions('POST'))
  },
  async createFeedback(payload) {
    return merchantRequest(`${CUSTOMER_API}/feedback`, jsonOptions('POST', { source: 'merchant', ...payload }))
  }
}

export function statusTitle(nextAction = 'create_application') {
  return ({
    create_application: '先提交流动摊位申请',
    application_pending: '申请已提交，等待审核',
    application_rejected: '申请未通过，可修改重提',
    dashboard: '审核已通过',
    disabled: '商户已被禁用'
  })[nextAction] || '申请状态'
}

export function statusDescription(status = {}) {
  const nextAction = status.next_action || 'create_application'
  return ({
    create_application: '填写基础信息后，平台会在后台审核。',
    application_pending: '平台正在处理你的申请，请保持电话畅通。',
    application_rejected: '查看驳回原因，修改资料后可以再次提交。',
    dashboard: '你已经可以使用商户工作台管理出摊和商品。',
    disabled: status.merchant?.disabled_reason || '该商户暂不可经营，请联系平台处理。'
  })[nextAction] || ''
}

export { money }
