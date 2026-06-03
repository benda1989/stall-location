<template>
  <section class="m-grid">
    <div class="m-actions">
      <button class="m-btn primary" @click="showForm = !showForm">+ 添加商品</button>
    </div>

    <form v-if="showForm" class="m-panel m-card m-form" @submit.prevent="submit">
      <label class="m-field">名称<input v-model="form.name" class="m-input" required></label>
      <label class="m-field">价格(元)<input v-model.number="form.price" class="m-input" type="number" min="0" step="0.01" required></label>
      <label class="m-field">库存<input v-model.number="form.stock" class="m-input" type="number" min="0" required></label>
      <label class="m-field full">商品图片<input class="m-input" type="file" accept="image/*" multiple required @change="readImages"><small>可多选，卡片背景会自动轮播。</small></label>
      <div v-if="form.image_urls.length" class="product-image-preview"><img v-for="image in form.image_urls" :key="image" :src="image" alt="商品图预览"></div>
      <div v-if="error" class="m-result error full">{{ error }}</div>
      <div class="m-field full m-actions">
        <button class="m-btn secondary" type="button" @click="showForm = false">取消</button>
        <button class="m-btn primary">保存</button>
      </div>
    </form>

    <div class="m-list">
      <article v-for="product in products" :key="product.id" class="m-panel m-card product-row" :class="{ 'has-carousel': productImages(product).length > 1 }">
        <div class="merchant-product-bg" aria-hidden="true">
          <span v-for="(image, index) in productImages(product)" :key="`${product.id}-${index}`" :style="{ backgroundImage: `url(${image})` }"></span>
        </div>
        <div class="m-card-head">
          <div>
            <h2>{{ product.name }}</h2>
            <p class="muted">当前库存 {{ product.stock }} · ¥{{ (product.price_cents / 100).toFixed(2) }}</p>
          </div>
          <button class="m-btn status-toggle" :class="product.status === 'on_sale' ? 'is-on' : 'is-off'" type="button" @click="toggleStatus(product)">{{ product.status === 'on_sale' ? '上架中' : '已下架' }}</button>
        </div>
        <div class="product-editor">
          <label class="m-field compact">库存数量<input v-model.number="drafts[product.id].stock" class="m-input" type="number" min="0"></label>
          <label class="m-field compact">价格(元)<input v-model.number="drafts[product.id].price" class="m-input" type="number" min="0" step="0.01"></label>
          <label class="m-field image-field">更换图片<input class="m-input" type="file" accept="image/*" multiple @change="readProductImages(product, $event)"></label>
          <button class="m-btn secondary" type="button" @click="saveProduct(product)">保存商品</button>
        </div>
        <div v-if="drafts[product.id]?.images?.length" class="product-image-preview inline"><img v-for="image in drafts[product.id].images" :key="image" :src="image" alt="新商品图预览"></div>
      </article>
    </div>
  </section>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'

const emit = defineEmits(['create', 'update'])
const props = defineProps({ products: Array })
const showForm = ref(false)
const error = ref('')
const form = reactive({ name: '', price: 0, stock: 1, image_urls: [] })
const drafts = reactive({})

watch(() => props.products, (products = []) => {
  products.forEach((product) => {
    if (!drafts[product.id]) drafts[product.id] = { stock: Number(product.stock || 0), price: Number(product.price_cents || 0) / 100, images: [] }
    else {
      drafts[product.id].stock = Number(product.stock || 0)
      drafts[product.id].price = Number(product.price_cents || 0) / 100
    }
  })
}, { immediate: true, deep: true })

function readImages(event) {
  readFiles(event).then((images) => { form.image_urls = images })
}
function readProductImages(product, event) {
  readFiles(event).then((images) => { drafts[product.id].images = images })
}
function readFiles(event) {
  const files = [...(event.target.files || [])].filter((item) => item.type.startsWith('image/'))
  if (!files.length) return Promise.resolve([])
  return Promise.all(files.map((file) => new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.readAsDataURL(file)
  }))).then((images) => images.filter(Boolean))
}

function submit() {
  if (!form.image_urls.length) {
    error.value = '新增商品至少需要一张图片'
    return
  }
  const imageURL = encodeImages(form.image_urls)
  emit('create', { name: form.name, price_cents: Math.round(Number(form.price) * 100), stock: form.stock, image_url: imageURL, status: 'on_sale' })
  Object.assign(form, { name: '', price: 0, stock: 1, image_urls: [] })
  error.value = ''
  showForm.value = false
}

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
function encodeImages(images) {
  return images.length === 1 ? images[0] : JSON.stringify(images)
}
function saveProduct(product) {
  const draft = drafts[product.id]
  const patch = {
    stock: Math.max(0, Number(draft?.stock || 0)),
    price_cents: Math.round(Number(draft?.price || 0) * 100)
  }
  if (draft?.images?.length) patch.image_url = encodeImages(draft.images)
  emit('update', product, patch)
}
function toggleStatus(product) {
  emit('update', product, { status: product.status === 'on_sale' ? 'off_sale' : 'on_sale' })
}
</script>
