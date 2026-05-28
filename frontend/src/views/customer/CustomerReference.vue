<template>
  <main v-if="showWeChatGuide" class="page wechat-guide" aria-label="微信打开引导">
    <section class="hero wechat-guide-hero">
      <div>
        <div class="eyebrow">WeChat H5</div>
        <h1>请在微信中打开</h1>
        <p>顾客端需要微信环境完成静默识别、定位和后续支付能力。复制链接到微信，或用微信扫一扫打开当前页面。</p>
      </div>
      <div class="wechat-qr-card">
        <span class="pill green">当前入口</span>
        <strong>{{ route.params.shopCode ? '单摊导航' : '附近地图' }}</strong>
        <small>{{ currentURL }}</small>
      </div>
    </section>
    <section class="card wechat-guide-actions">
      <button class="guide-action primary" type="button" @click="copyLink">复制链接到微信</button>
      <p class="muted">顾客端仅在微信环境中进入完整链路；请复制链接到微信或使用微信扫码打开。</p>
    </section>
  </main>

  <main v-else class="reference-host" aria-label="顾客端参考实现">
    <iframe
      ref="frameEl"
      class="reference-frame"
      :src="frameSrc"
      title="附近摊位地图"
      @load="applyCustomerRoute"
    ></iframe>
  </main>
</template>

<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { showToast } from 'vant'

const route = useRoute()
const frameEl = ref(null)
const isWeChat = /MicroMessenger/i.test(navigator.userAgent || '')
const currentURL = ref('')
const customerToken = ref(localStorage.getItem('customer_token') || '')
const showWeChatGuide = computed(() => !isWeChat)

const frameSrc = computed(() => {
  const query = new URLSearchParams()
  if (route.params.shopCode) query.set('shopCode', route.params.shopCode)
  if (route.query.favorite) query.set('favorite', String(route.query.favorite))
  if (route.query.merchantId) query.set('merchantId', String(route.query.merchantId))
  if (route.query.shopId) query.set('shopId', String(route.query.shopId))
  if (import.meta.env.VITE_API_BASE) query.set('apiBase', import.meta.env.VITE_API_BASE)
  if (import.meta.env.VITE_TENCENT_MAP_KEY) query.set('tencentMapKey', import.meta.env.VITE_TENCENT_MAP_KEY)
  if (customerToken.value) query.set('customerToken', customerToken.value)
  const raw = query.toString()
  return `/reference/customer.html${raw ? `?${raw}` : ''}`
})

function syncCustomerTokenFromQuery() {
  const incoming = route.query.customer_token
  if (!incoming) return
  customerToken.value = String(incoming)
  localStorage.setItem('customer_token', customerToken.value)
}

function applyCustomerRoute() {
  const frameWindow = frameEl.value?.contentWindow
  if (!frameWindow) return
  try {
    if (typeof frameWindow.setCustomerContext === 'function') {
      frameWindow.setCustomerContext({
        shopCode: route.params.shopCode || '',
        favorite: route.query.favorite || '',
        merchantId: route.query.merchantId || '',
        shopId: route.query.shopId || ''
      })
    } else if (route.params.shopCode && typeof frameWindow.selectVendor === 'function') {
      frameWindow.selectVendor(route.params.shopCode)
    } else if (typeof frameWindow.setActiveTab === 'function') {
      frameWindow.setActiveTab('map')
    }
  } catch (error) {
    // The reference file is same-origin in dev; ignore until iframe finishes booting.
  }
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(currentURL.value)
    showToast('链接已复制，打开微信粘贴访问')
  } catch (error) {
    showToast('复制失败，请手动复制地址栏链接')
  }
}

onMounted(() => {
  currentURL.value = window.location.href
  syncCustomerTokenFromQuery()
  if (isWeChat && !customerToken.value) {
    window.location.href = `/api/wechat/oauth/silent/start?redirect=${encodeURIComponent(window.location.href)}`
  }
})

watch(() => route.fullPath, async () => {
  currentURL.value = window.location.href
  syncCustomerTokenFromQuery()
  await nextTick()
  applyCustomerRoute()
})
</script>
