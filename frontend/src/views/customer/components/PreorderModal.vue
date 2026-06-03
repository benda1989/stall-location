<template>
  <div v-if="open" class="modal-backdrop" @click.self="$emit('close')">
    <section class="sheet" role="dialog" aria-modal="true">
      <div class="sheet-head"><div><h2>确认预定</h2><p class="muted">{{ vendor?.name }}</p></div><button class="c-btn ghost" type="button" @click="$emit('close')">关闭</button></div>
      <div class="card-list">
        <article v-for="product in products" :key="product.id" :class="['product-select', quantity(product) > 0 && 'is-selected']">
          <div class="product-thumb" aria-hidden="true">
            <img :src="primaryProductImage(product)" alt="">
          </div>
          <div class="product-select-content">
            <strong>{{ product.name }}</strong>
            <p class="product-stock">余 {{ product.stock }}</p>
            <div class="product-buy-row">
              <strong class="product-price">{{ money(productPriceCents(product)) }}</strong>
              <div class="qty"><button type="button" :disabled="quantity(product) <= 0" @click="step(product, -1)">-</button><span>{{ quantity(product) }}</span><button type="button" :disabled="quantity(product) >= product.stock" @click="step(product, 1)">+</button></div>
            </div>
          </div>
        </article>
      </div>
      <div class="form-grid">
        <label class="field full">取货时间<select v-model="form.pickup" class="c-select"><option>尽快到摊取</option><option>15 分钟后取</option><option>30 分钟后取</option></select></label>
        <label class="field">取货人<input v-model="form.name" class="c-input" autocomplete="name"></label>
        <label class="field">手机号<input v-model="form.phone" class="c-input" inputmode="tel" autocomplete="tel" placeholder="请输入手机号"></label>
        <label class="field full">备注<input v-model="form.note" class="c-input" placeholder="少辣、不要葱等，可留空"></label>
      </div>
      <div v-if="error" class="result error">{{ error }}</div>
      <div class="card-actions"><button class="c-btn secondary" type="button" @click="$emit('close')">取消</button><button class="c-btn primary" type="button" :disabled="submitting || !selectedItems.length" @click="submit">{{ submitting ? '提交中…' : `确认预定 · ${totalText}` }}</button></div>
    </section>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { apiFetch, clearAuthToken, customerHeaders, devCustomerOpenID, hasValidTokenRole, money, unifiedLogin } from '../../../api/client'
const props = defineProps({ open: Boolean, vendor: Object, storedContact: Object, ensureLogin: Function })
const emit = defineEmits(['close', 'success', 'save-contact', 'ensure-login'])
const quantities = reactive({})
const form = reactive({ name: '', phone: '', note: '', pickup: '尽快到摊取' })
const error = ref('')
const submitting = ref(false)
watch(() => props.open, (open) => {
  if (!open || !props.vendor) return
  Object.keys(quantities).forEach((key) => delete quantities[key])
  const first = props.vendor.products?.find((item) => item.stock > 0)
  if (first) quantities[first.id] = 1
  form.name = props.storedContact?.name || ''
  form.phone = props.storedContact?.phone || ''
  form.note = ''
  form.pickup = '尽快到摊取'
  error.value = ''
})
const products = computed(() => props.vendor?.products || [])
const selectedItems = computed(() => products.value.map((product) => ({ product, quantity: quantities[product.id] || 0 })).filter((item) => item.quantity > 0))
const totalText = computed(() => money(selectedItems.value.reduce((sum, item) => sum + Number(productPriceCents(item.product) || 0) * item.quantity, 0)))
function quantity(product) { return quantities[product.id] || 0 }
function step(product, delta) { quantities[product.id] = Math.max(0, Math.min(product.stock, quantity(product) + delta)); if (!quantities[product.id]) delete quantities[product.id] }
function validPhone(phone) { const digits = String(phone || '').replace(/\D/g, ''); return (digits.length === 11 && digits.startsWith('1')) || digits.length === 8 || (digits.length === 11 && digits.startsWith('852')) }
function productPriceCents(product) { return product?.price_cents ?? product?.priceCents ?? Number(product?.price || 0) * 100 }
function primaryProductImage(product) { return productImages(product)[0] }
function productImages(product) {
  const parsed = parseImages(product?.image_urls || product?.images || product?.image_url)
  return parsed.length ? parsed : generatedImages(product?.name || '商品')
}
function parseImages(value) {
  if (!value) return []
  if (Array.isArray(value)) return value.filter(Boolean)
  const raw = String(value).trim()
  if (!raw) return []
  if (raw.startsWith('[')) {
    try { return JSON.parse(raw).filter(Boolean) } catch {}
  }
  if (raw.startsWith('data:image')) return [raw]
  return raw.split(/[\n|,，]+/).map((item) => item.trim()).filter(Boolean)
}
function generatedImages(name) {
  return [placeholderImage(name, '#f59e0b', '#fff3d8'), placeholderImage(name, '#16a34a', '#fef3c7')]
}
function placeholderImage(name, color, bg) {
  const safe = String(name || '商品').slice(0, 8)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360" viewBox="0 0 640 360"><defs><linearGradient id="g" x1="0" x2="1" y1="0" y2="1"><stop stop-color="${bg}"/><stop offset="1" stop-color="${color}" stop-opacity=".42"/></linearGradient></defs><rect width="640" height="360" fill="url(#g)"/><circle cx="520" cy="58" r="96" fill="${color}" opacity=".2"/><circle cx="104" cy="300" r="132" fill="#2f1f0d" opacity=".08"/><text x="52" y="196" font-family="PingFang SC, Microsoft YaHei, sans-serif" font-size="56" font-weight="900" fill="#2f1f0d">${safe}</text><text x="56" y="248" font-family="PingFang SC, Microsoft YaHei, sans-serif" font-size="22" font-weight="700" fill="#7a6751">今日现做 · 到摊自取</text></svg>`
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}
function pickupTime(label) {
  const minutes = label.includes('30') ? 30 : label.includes('15') ? 15 : 5
  return new Date(Date.now() + minutes * 60 * 1000).toISOString()
}
async function submit() {
  if (!form.name || !form.phone) { error.value = '请填写取货人和手机号'; return }
  if (!validPhone(form.phone)) { error.value = '请输入有效手机号'; return }
  submitting.value = true; error.value = ''
  try {
    const resp = await submitOrder()
    emit('save-contact', { name: form.name, phone: form.phone })
    emit('success', resp.order)
    emit('close')
  } catch (e) {
    if ([401, 403].includes(e.status) && import.meta.env.DEV) {
      try {
        await ensureDevCustomerLogin(true)
        const resp = await submitOrder({ skipEnsureLogin: true })
        emit('save-contact', { name: form.name, phone: form.phone })
        emit('success', resp.order)
        emit('close')
        return
      } catch (retryError) {
        error.value = retryError.message || '预定失败，请重新打开顾客页'
        return
      }
    }
    error.value = e.message || '预定失败'
  } finally { submitting.value = false }
}

async function submitOrder(options = {}) {
  if (!options.skipEnsureLogin) await ensureCustomerLoginForAction()
  await ensureDevCustomerLogin(false)
  return apiFetch('/api/orders', { method: 'POST', headers: orderHeaders(), body: orderBody() })
}

async function ensureCustomerLoginForAction() {
  if (typeof props.ensureLogin === 'function') {
    await props.ensureLogin(false)
    return
  }
  emit('ensure-login')
}

async function ensureDevCustomerLogin(force) {
  emit('ensure-login')
  if (!import.meta.env.DEV && !force) return
  if (!force && hasValidTokenRole('customer')) return
  clearAuthToken('customer')
  try {
    const resp = await unifiedLogin('customer', { dev_openid: devCustomerOpenID() })
    localStorage.setItem('customer_token', resp.token)
  } catch (error) {
    if (!import.meta.env.DEV) throw error
  }
}

function orderHeaders(forceDevIdentity = false) {
  const headers = customerHeaders()
  if (forceDevIdentity && import.meta.env.DEV) {
    delete headers.Authorization
    headers['X-Dev-Customer'] = '1'
    headers['X-Dev-OpenID'] = devCustomerOpenID()
  }
  return headers
}

function orderBody() {
  return JSON.stringify({ shop_code: props.vendor.shopCode, customer_name: form.name, customer_phone: form.phone, pickup_time: pickupTime(form.pickup), remark: form.note, items: selectedItems.value.map(({ product, quantity }) => ({ product_id: product.id, quantity })) })
}
</script>
