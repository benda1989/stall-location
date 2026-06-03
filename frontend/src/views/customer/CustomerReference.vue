<template>
  <main v-if="showWeChatGuide" class="page wechat-guide" aria-label="微信打开引导">
    <section class="hero wechat-guide-hero"><div><div class="eyebrow">WeChat H5</div><h1>请在微信中打开</h1><p>顾客端需要微信环境完成静默识别、定位和后续支付能力。复制链接到微信，或用微信扫一扫打开当前页面。</p></div><div class="wechat-qr-card"><span class="pill green">当前入口</span><strong>{{ route.params.shareCode || route.params.shopCode ? '单摊导航' : '附近地图' }}</strong><small>{{ currentURL }}</small></div></section>
    <section class="card wechat-guide-actions"><button class="guide-action primary" type="button" @click="copyLink">复制链接到微信</button><p class="muted">顾客端仅在微信环境中进入完整链路；请复制链接到微信或使用微信扫码打开。</p></section>
  </main>
  <main v-else :class="['customer-app', activeTab === 'map' && 'is-map-tab']" aria-label="顾客端">
    <div class="customer-shell">
      <CustomerMap
        v-if="activeTab === 'map'"
        v-model:query="query"
        :vendors="visibleVendors"
        :selected-id="selectedId"
        :favorite-ids="favoriteIds"
        :selected-categories="selectedCategories"
        :user-location="userLocation"
        :focused="Boolean(focusedShopCode || route.params.shopCode)"
        :dev-location-enabled="allowLocalPreview"
        @search="loadMap"
        @category-change="onCategoryChange"
        @viewport-change="onViewportChange"
        @dev-location="setDevLocation"
        @select="selectVendor"
        @preorder="openPreorder"
        @toggle-favorite="toggleFavorite"
        @close-detail="selectedId = ''"
      />
      <CustomerOrders v-else-if="activeTab === 'orders'" :orders="orders" @view-shop="viewShop" @cancel="cancelOrder" @feedback="feedbackOpen = true" />
      <CustomerFavorites v-else :vendors="favoriteVendors" @view="selectFromFavorites" @remove="removeFavorite" @preorder="openPreorder" @join="joinOpen = true" />
    </div>
    <nav class="customer-tabs"><button :class="{ 'is-active': activeTab === 'map' }" @click="activeTab = 'map'">地图</button><button :class="{ 'is-active': activeTab === 'orders' }" @click="activeTab = 'orders'">订单</button><button :class="{ 'is-active': activeTab === 'favorites' }" @click="activeTab = 'favorites'">收藏</button></nav>
    <PreorderModal :open="preorderOpen" :vendor="preorderVendor" :stored-contact="storedContact" :ensure-login="ensureCustomerLogin" @close="preorderOpen = false" @success="onOrderCreated" @save-contact="saveContact" />
    <FeedbackModal :open="feedbackOpen" :stored-contact="storedContact" @close="feedbackOpen = false" @save-contact="saveContact" />
    <JoinApplicationModal :open="joinOpen" :stored-contact="storedContact" @close="joinOpen = false" @save-contact="saveContact" />
  </main>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { showToast } from 'vant'
import { apiFetch, clearAuthToken, customerHeaders, devCustomerOpenID, hasValidTokenRole, unifiedLogin } from '../../api/client'
import { getLocation } from '../../api/wechat'
import { hasTencentMapKey } from '../../api/tencentMap'
import './customer.css'
import CustomerMap from './components/CustomerMap.vue'
import CustomerOrders from './components/CustomerOrders.vue'
import CustomerFavorites from './components/CustomerFavorites.vue'
import PreorderModal from './components/PreorderModal.vue'
import FeedbackModal from './components/FeedbackModal.vue'
import JoinApplicationModal from './components/JoinApplicationModal.vue'

const route = useRoute()
const isWeChat = /MicroMessenger/i.test(navigator.userAgent || '')
const currentURL = ref('')
const customerToken = ref(localStorage.getItem('customer_token') || '')
const allowLocalPreview = computed(() => import.meta.env.DEV || route.query.preview === '1')
const showWeChatGuide = computed(() => !isWeChat && !allowLocalPreview.value)
const activeTab = ref('map')
const vendors = ref([])
const orders = ref([])
const favoriteItems = ref([])
const selectedId = ref('')
const focusedShopCode = ref('')
const query = ref('')
const selectedCategories = ref([])
const userLocation = ref(null)
const viewportBounds = ref(null)
const apiError = ref('')
const preorderOpen = ref(false)
const preorderVendorId = ref('')
const feedbackOpen = ref(false)
const joinOpen = ref(false)
const favoriteIds = ref(new Set())
const storedContact = ref(JSON.parse(localStorage.getItem('mplzCustomerContact') || '{}'))

const visibleVendors = computed(() => vendors.value)
const favoriteVendors = computed(() => {
  const byID = new Map(favoriteItems.value.map((vendor) => [vendor.id, vendor]))
  vendors.value.forEach((vendor) => { if (favoriteIds.value.has(vendor.id)) byID.set(vendor.id, vendor) })
  return [...byID.values()].map((vendor) => ({ ...vendor, orderCount: orderCountForVendor(vendor) }))
})
const preorderVendor = computed(() => [...vendors.value, ...favoriteItems.value].find((vendor) => vendor.id === preorderVendorId.value) || null)

function syncCustomerTokenFromQuery() { const incoming = route.query.customer_token; if (incoming) { customerToken.value = String(incoming); localStorage.setItem('customer_token', customerToken.value) } }
function loadFavoriteIds() {
  const next = new Set()
  for (const key of ['mplzCustomerFavorites', 'mplzCustomerFavoritesVue']) {
    try { JSON.parse(localStorage.getItem(key) || '[]').forEach((id) => next.add(id)) } catch {}
  }
  return next
}
favoriteIds.value = loadFavoriteIds()
function saveFavorites() {
  const payload = JSON.stringify([...favoriteIds.value])
  localStorage.setItem('mplzCustomerFavorites', payload)
  localStorage.setItem('mplzCustomerFavoritesVue', payload)
}
function loadStoredOrders() {
  try { return JSON.parse(localStorage.getItem('mplzCustomerOrders') || '[]') || [] } catch { return [] }
}
function persistOrders() {
  localStorage.setItem('mplzCustomerOrders', JSON.stringify(orders.value.slice(0, 80)))
}
function saveContact(contact) { storedContact.value = { name: contact.name || '', phone: contact.phone || '' }; localStorage.setItem('mplzCustomerContact', JSON.stringify(storedContact.value)) }
async function copyLink() { try { await navigator.clipboard.writeText(currentURL.value); showToast('链接已复制，打开微信粘贴访问') } catch { showToast('复制失败，请手动复制地址栏链接') } }
function orderCountForVendor(vendor) {
  return (orders.value || []).filter((order) => {
    const shop = order.shop || {}
    return (vendor.shopId && Number(order.shop_id || shop.id) === Number(vendor.shopId)) || (vendor.shopCode && (shop.shop_code || order.shopCode) === vendor.shopCode)
  }).length
}
function normalizeItem(item, index = 0) {
  const shop = item.shop || {}
  const session = item.stall_session || {}
  return { id: shop.shop_code || String(shop.id || index), shopCode: shop.shop_code, shopId: shop.id, merchantId: shop.merchant_id, name: shop.name || '流动摊位', category: shop.category || '其他摊位', address: session.address || shop.announcement || '摊主当前位置', area: session.address || '摊主当前位置', lat: Number(session.lat || 0), lng: Number(session.lng || 0), x: 18 + (index * 19) % 66, y: 30 + (index * 23) % 48, isOpen: item.display_status !== 'unavailable' && item.display_status !== 'recent' && Boolean(session.id), statusText: item.display_status === 'recent' ? '最近在线' : session.id ? '营业中' : '未出摊', endText: session.expected_end_at ? new Date(session.expected_end_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : '', products: (item.products || []).map((p) => ({ ...p, price_cents: p.price_cents ?? p.priceCents ?? 0 })) }
}
async function resolveShare() {
  const shareCode = route.params.shareCode
  if (!shareCode) return false
  const data = await apiFetch(`/api/shares/${encodeURIComponent(shareCode)}`)
  focusedShopCode.value = data.share?.shop_code || data.shop?.shop_code || ''
  return Boolean(focusedShopCode.value)
}
async function ensureCustomerLogin(force = false) {
  if (!force && customerToken.value && hasValidTokenRole('customer')) return
  clearAuthToken('customer')
  customerToken.value = ''
  if (!allowLocalPreview.value) {
    if (isWeChat) {
      const redirect = encodeURIComponent(window.location.href)
      window.location.href = `/api/wechat/oauth/silent/start?redirect=${redirect}`
    }
    return
  }
  try {
    await loginDevCustomer()
  } catch (error) {
    if (!allowLocalPreview.value) throw error
    showToast(error.message || '开发顾客登录失败，请检查后端服务')
  }
}
async function loginDevCustomer() {
  const resp = await unifiedLogin('customer', { dev_openid: devCustomerOpenID() })
  customerToken.value = resp.token
  localStorage.setItem('customer_token', resp.token)
}
async function loadMap() {
  try {
    apiError.value = ''
    if (focusedShopCode.value || route.params.shopCode) {
      const code = focusedShopCode.value || route.params.shopCode
      const data = await apiFetch(`/api/shops/${encodeURIComponent(code)}/map-state`)
      vendors.value = data.shop ? [normalizeItem(data, 0)] : []
      if (route.params.shareCode && vendors.value[0]) await addFavorite(vendors.value[0])
    } else {
      const params = new URLSearchParams()
      if (query.value) params.set('q', query.value)
      selectedCategories.value.forEach((category) => params.append('category', category))
      if (userLocation.value?.lat && userLocation.value?.lng) {
        params.set('lat', userLocation.value.lat)
        params.set('lng', userLocation.value.lng)
      }
      if (viewportBounds.value && !query.value) {
        for (const [key, value] of Object.entries(viewportBounds.value)) {
          if (Number.isFinite(Number(value))) params.set(key, value)
        }
      }
      const data = await apiFetch(`/api/stalls/nearby${params.toString() ? `?${params}` : ''}`)
      vendors.value = (data.stalls || []).map(normalizeItem)
      if (!vendors.value.some((vendor) => vendor.id === selectedId.value)) selectedId.value = ''
    }
  } catch (error) { apiError.value = error.message || '地图数据加载失败'; showToast(apiError.value) }
}
async function loadOrders() {
  try {
    if (customerToken.value || allowLocalPreview.value) {
      const data = await apiFetch('/api/customer/orders', { headers: customerHeaders() })
      orders.value = data.orders || []
      persistOrders()
    } else {
      orders.value = []
    }
  } catch (error) {
    if ([401, 403].includes(error.status) && allowLocalPreview.value) {
      await ensureCustomerLogin(true)
      return loadOrders()
    }
    orders.value = loadStoredOrders()
    showToast(error.message || '订单加载失败，请重新登录')
  }
}
async function loadFavorites() {
  if (!customerToken.value) return
  try {
    const data = await apiFetch('/api/customer/favorites', { headers: customerHeaders() })
    favoriteItems.value = (data.stalls || []).map(normalizeItem)
    favoriteIds.value = new Set(favoriteItems.value.map((vendor) => vendor.id))
    saveFavorites()
  } catch (error) {
    if ([401, 403].includes(error.status) && allowLocalPreview.value) {
      await ensureCustomerLogin(true)
      return loadFavorites()
    }
    showToast(error.message || '收藏加载失败')
  }
}
function selectVendor(id) { selectedId.value = id }
function onCategoryChange(categories) { selectedCategories.value = categories; loadMap() }
function onViewportChange(bounds) { viewportBounds.value = bounds || null; loadMap() }
async function addFavorite(vendor) {
  if (!vendor) return
  try {
    await apiFetch('/api/customer/favorites', { method: 'POST', headers: customerHeaders(), body: JSON.stringify({ shop_id: vendor.shopId, shop_code: vendor.shopCode }) })
    const next = new Set(favoriteIds.value)
    next.add(vendor.id)
    favoriteIds.value = next
    if (!favoriteItems.value.some((item) => item.id === vendor.id)) favoriteItems.value = [vendor, ...favoriteItems.value]
    saveFavorites()
  } catch (error) {
    if ([401, 403].includes(error.status) && allowLocalPreview.value) {
      await ensureCustomerLogin(true)
      return addFavorite(vendor)
    }
    showToast(error.message || '收藏失败')
  }
}
async function removeFavorite(id) {
  const vendor = [...vendors.value, ...favoriteItems.value].find((item) => item.id === id)
  const shopID = vendor?.shopId
  try {
    if (shopID) await apiFetch(`/api/customer/favorites/${encodeURIComponent(shopID)}`, { method: 'DELETE', headers: customerHeaders() })
    const next = new Set(favoriteIds.value)
    next.delete(id)
    favoriteIds.value = next
    favoriteItems.value = favoriteItems.value.filter((item) => item.id !== id)
    saveFavorites()
  } catch (error) {
    if ([401, 403].includes(error.status) && allowLocalPreview.value) {
      await ensureCustomerLogin(true)
      return removeFavorite(id)
    }
    showToast(error.message || '取消收藏失败')
  }
}
async function toggleFavorite(id) {
  if (favoriteIds.value.has(id)) return removeFavorite(id)
  const vendor = vendors.value.find((item) => item.id === id)
  return addFavorite(vendor)
}
function openPreorder(id) { preorderVendorId.value = id; preorderOpen.value = true }
function selectFromFavorites(id) { activeTab.value = 'map'; selectedId.value = id }
function viewShop(code) { if (!code) return; activeTab.value = 'map'; selectedId.value = code }
async function cancelOrder(order) { if (!order.order_no) return; try { await apiFetch(`/api/orders/${encodeURIComponent(order.order_no)}/cancel`, { method: 'POST', headers: customerHeaders(), body: JSON.stringify({}) }); orders.value = orders.value.map((item) => item.order_no === order.order_no ? { ...item, status: 'canceled' } : item); persistOrders() } catch (error) { showToast(error.message) } }
function onOrderCreated(order) { orders.value = [order, ...orders.value]; persistOrders(); activeTab.value = 'orders'; loadMap(); loadOrders() }
async function setDevLocation(location) {
  userLocation.value = { ...location, x: 50, y: 52 }
  localStorage.setItem('mplzDevLocation', JSON.stringify(location))
  viewportBounds.value = null
  showToast(`已切换定位：${location.name}`)
  if (!hasTencentMapKey()) await loadMap()
}
async function bootstrap() {
  currentURL.value = window.location.href
  syncCustomerTokenFromQuery()
  if (isWeChat && !customerToken.value) { window.location.href = `/api/wechat/oauth/silent/start?redirect=${encodeURIComponent(window.location.href)}`; return }
  await ensureCustomerLogin()
  try {
    await requestUserLocation()
    const focused = await resolveShare()
    if (focused || route.params.shopCode || !hasTencentMapKey()) await loadMap()
    await loadFavorites()
    await loadOrders()
    return focused
  } catch (error) {
    if (route.params.shareCode) {
      apiError.value = error.message || '分享链接无效'
      vendors.value = []
      selectedId.value = ''
    showToast(apiError.value)
      await loadFavorites()
      await loadOrders()
      return false
    }
    await loadMap()
    await loadFavorites()
    await loadOrders()
    return false
  }
}
async function requestUserLocation() {
  if (userLocation.value) return
  try {
    const location = await getLocation()
    userLocation.value = { ...location, x: 50, y: 52 }
  } catch {
    userLocation.value = null
  }
}
onMounted(bootstrap)
watch(() => route.fullPath, bootstrap)
</script>
