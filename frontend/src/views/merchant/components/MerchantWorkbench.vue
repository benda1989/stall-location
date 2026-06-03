<template>
  <main class="merchant-workbench">
    <nav class="merchant-tabs">
      <button :class="{ 'is-active': tab === 'overview' }" @click="tab = 'overview'">总览</button>
      <button :class="{ 'is-active': tab === 'orders' }" @click="tab = 'orders'">订单</button>
      <button :class="{ 'is-active': tab === 'products' }" @click="tab = 'products'">商品</button>
    </nav>

    <MerchantOverview
      v-if="tab === 'overview'"
      :dashboard="dashboard"
      :shop="shop"
      :session="session"
      :products="products"
      :qr-url="qrUrl"
      :qr-data-url="qrDataUrl"
      @open-session="sessionOpen = true"
      @end-session="endSession"
      @copy-link="copyLink"
    />
    <MerchantOrders
      v-else-if="tab === 'orders'"
      :orders="orders"
      @order-action="transitionOrder"
      @feedback="feedbackOpen = true"
    />
    <MerchantProducts
      v-else
      :products="products"
      @create="createProduct"
      @update="updateProduct"
    />

    <MerchantFeedbackModal :open="feedbackOpen" :defaults="feedbackDefaults" @close="feedbackOpen = false" />
    <StallSessionModal :open="sessionOpen" @close="sessionOpen = false" @submit="startSession" />
  </main>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { showToast } from 'vant'
import { apiFetch, merchantHeaders } from '../../../api/client'
import '../merchant.css'
import MerchantOverview from './MerchantOverview.vue'
import MerchantOrders from './MerchantOrders.vue'
import MerchantProducts from './MerchantProducts.vue'
import MerchantFeedbackModal from './MerchantFeedbackModal.vue'
import StallSessionModal from './StallSessionModal.vue'

const props = defineProps({ initialScreen: String })
const tab = ref(props.initialScreen || 'overview')
const dashboard = ref({})
const shop = ref(null)
const session = ref(null)
const products = ref([])
const orders = ref([])
const qrUrl = ref('')
const qrDataUrl = ref('')
const feedbackOpen = ref(false)
const sessionOpen = ref(false)
const feedbackDefaults = computed(() => ({ name: shop.value?.name || '', phone: shop.value?.contact_phone || localStorage.getItem('merchant_phone') || '' }))

watch(() => props.initialScreen, (value) => {
  if (value) tab.value = value
})

async function load() {
  const [dashboardResp, productsResp, ordersResp, qrResp] = await Promise.all([
    apiFetch('/api/merchant/dashboard', { headers: merchantHeaders() }),
    apiFetch('/api/merchant/products', { headers: merchantHeaders() }),
    apiFetch('/api/merchant/orders', { headers: merchantHeaders() }),
    apiFetch('/api/merchant/qrcode', { headers: merchantHeaders() })
  ])
  dashboard.value = dashboardResp
  shop.value = dashboardResp.shop || qrResp.shop
  session.value = dashboardResp.stall_session
  products.value = productsResp.products || []
  orders.value = ordersResp.orders || []
  qrUrl.value = qrResp.url || ''
  qrDataUrl.value = qrResp.qr_data_url || ''
}

async function safeLoad() {
  try {
    await load()
  } catch (error) {
    showToast(error.message)
    if (/401|403|authorization|token/i.test(error.message || '')) {
      localStorage.removeItem('merchant_token')
      window.location.href = '/merchant/login'
    }
  }
}

async function transitionOrder(order, action) {
  try {
    await apiFetch(`/api/merchant/orders/${order.id}/${action}`, { method: 'POST', headers: merchantHeaders(), body: JSON.stringify({}) })
    await load()
    showToast('订单已更新')
  } catch (error) {
    showToast(error.message)
  }
}

async function createProduct(payload) {
  try {
    await apiFetch('/api/merchant/products', { method: 'POST', headers: merchantHeaders(), body: JSON.stringify(payload) })
    await load()
    showToast('商品已添加')
  } catch (error) {
    showToast(error.message)
  }
}

async function updateProduct(product, patch) {
  try {
    const payload = { ...product, ...patch }
    await apiFetch(`/api/merchant/products/${product.id}`, { method: 'PUT', headers: merchantHeaders(), body: JSON.stringify(payload) })
    await load()
    showToast('商品已更新')
  } catch (error) {
    showToast(error.message)
  }
}

async function startSession(payload) {
  try {
    await apiFetch('/api/merchant/stall-sessions/start', { method: 'POST', headers: merchantHeaders(), body: JSON.stringify(payload) })
    sessionOpen.value = false
    await load()
    showToast('已开始出摊')
  } catch (error) {
    showToast(error.message)
  }
}

async function endSession() {
  if (!session.value?.id) return
  try {
    await apiFetch(`/api/merchant/stall-sessions/${session.value.id}/end`, { method: 'POST', headers: merchantHeaders(), body: JSON.stringify({}) })
    await load()
    showToast('已结束出摊')
  } catch (error) {
    showToast(error.message)
  }
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(qrUrl.value)
    showToast('链接已复制')
  } catch {
    showToast(qrUrl.value || '链接生成中')
  }
}

onMounted(safeLoad)
</script>
