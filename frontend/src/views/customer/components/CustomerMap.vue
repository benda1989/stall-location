<template>
  <section class="map-screen" aria-label="顾客地图">
    <div ref="mapEl" class="map-canvas" @click="$emit('close-detail')"></div>
    <div class="map-fallback-layer" aria-label="摊位点位">
      <button v-for="(vendor, index) in vendors" :key="vendor.id" class="map-pin" type="button" :style="pinStyle(vendor, index)" @click.stop="selectDomVendor(vendor.id)">
        <i><img :src="categoryIconDataUri(vendor.category)" alt=""></i><span>{{ vendor.name }}</span>
      </button>
    </div>

    <div class="map-toolbar">
      <div class="map-search is-open">
        <div class="search-input-wrap">
          <input ref="searchInput" class="c-input" :value="query" type="search" placeholder="搜索摊位 / 品类 / 街区，回车确认" @input="$emit('update:query', $event.target.value)" @keyup.enter="submitSearch" @search="handleSearchInput">
          <button v-if="query" class="search-clear" type="button" aria-label="清空检索" @click="clearSearch">×</button>
        </div>
      </div>
      <button class="category-toggle" type="button" @click.stop="categoryOpen = !categoryOpen">
        <span class="category-toggle-icon"><img :src="categoryIconDataUri(selectedCategories[0] || '其他摊位')" alt=""><b v-if="selectedCategories.length">{{ selectedCategories.length }}</b></span>
      </button>
      <div v-if="selectedCategories.length" class="selected-category-row" aria-label="已选分类">
        <button v-for="category in selectedCategories" :key="category" class="selected-category-chip" type="button" @click.stop="removeCategory(category)">
          <i><img :src="categoryIconDataUri(category)" alt=""></i><span>{{ category }}</span><b aria-label="清除分类">×</b>
        </button>
      </div>
      <div v-if="categoryOpen" class="category-popover" @click.stop>
        <button v-for="category in categories" :key="category" :class="['category-option', selectedCategories.includes(category) && 'is-active']" type="button" @click="addCategory(category)">
          <i><img :src="categoryIconDataUri(category)" alt=""></i><span>{{ category }}</span>
        </button>
      </div>
      <select v-if="devLocationEnabled" class="dev-location-select" aria-label="开发环境定位" @change="chooseDevLocation">
        <option value="">开发定位</option>
        <option v-for="location in devLocations" :key="location.name" :value="location.name">{{ location.name }}</option>
      </select>
    </div>

    <div class="map-count">{{ openCount }} 个摊位</div>
    <div v-if="mapStatus !== 'ready'" class="map-status">{{ mapStatusText }}</div>
    <aside v-if="selectedVendor" class="c-panel detail-card" @click.stop>
      <button class="detail-close" type="button" aria-label="关闭摊位详情" @click="$emit('close-detail')">×</button>
      <div class="detail-head">
        <div><span class="status-pill open">{{ selectedVendor.statusText }}</span><h2>{{ selectedVendor.name }}</h2></div>
      </div>
      <div class="detail-grid">
        <div class="mini-stat"><span>位置</span><strong>{{ selectedVendor.address || selectedVendor.area }}</strong></div>
        <div class="mini-stat"><span>营业时间</span><strong>{{ businessText(selectedVendor) }}</strong></div>
      </div>
      <div class="card-list">
        <div v-for="product in selectedVendor.products" :key="product.id || product.name" class="product-chip"><span>{{ product.name }}</span><strong>{{ money(product.price_cents ?? product.priceCents ?? product.price * 100) }}</strong></div>
      </div>
      <div class="card-actions">
        <button class="c-btn secondary" type="button" @click="$emit('toggle-favorite', selectedVendor.id)">{{ isFavorite ? '已收藏' : '收藏' }}</button>
        <button class="c-btn primary" type="button" :disabled="!selectedVendor.isOpen" @click="$emit('preorder', selectedVendor.id)">预定</button>
      </div>
    </aside>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { money } from '../../../api/client'
import { hasTencentMapKey, loadTencentMap } from '../../../api/tencentMap'
import { STALL_CATEGORY_OPTIONS, categoryIconDataUri } from '../categoryIcons'

const props = defineProps({ vendors: { type: Array, default: () => [] }, selectedId: String, favoriteIds: Object, query: String, selectedCategories: { type: Array, default: () => [] }, userLocation: Object, focused: Boolean, devLocationEnabled: Boolean })
const emit = defineEmits(['select', 'preorder', 'toggle-favorite', 'close-detail', 'update:query', 'search', 'category-change', 'viewport-change', 'dev-location'])
const categories = STALL_CATEGORY_OPTIONS
const devLocations = [
  { name: '旺角 E2 口', lat: 22.3193, lng: 114.1694, accuracy: 12 },
  { name: '创意园东门', lat: 22.3221, lng: 114.1712, accuracy: 18 },
  { name: '社区南门', lat: 22.3168, lng: 114.1678, accuracy: 15 },
  { name: '青榕广场', lat: 22.3242, lng: 114.1669, accuracy: 20 }
]
const mapEl = ref(null)
const mapStatus = ref(hasTencentMapKey() ? 'loading' : 'fallback')
const searchInput = ref(null)
const categoryOpen = ref(false)
let TMap = null
let map = null
let vendorMarkers = null
let userMarker = null
let viewportTimer = null
let markerClickHandler = null
let mapClickHandler = null
let viewportHandlers = []
let viewportUserIntent = false
let ignoreNextMapClick = false
let positionTimers = []
let pinPositionFrozen = false

const selectedVendor = computed(() => props.vendors.find((item) => item.id === props.selectedId) || null)
const isFavorite = computed(() => selectedVendor.value && props.favoriteIds?.has(selectedVendor.value.id))
const openCount = computed(() => props.vendors.filter((item) => item.isOpen).length)
const mapStatusText = computed(() => mapStatus.value === 'error' ? '腾讯地图加载失败，已切换点位预览' : mapStatus.value === 'fallback' ? '未配置腾讯地图 Key，使用点位预览' : '正在连接腾讯地图')
const domPinPositions = ref({})

function submitSearch() { categoryOpen.value = false; searchInput.value?.blur?.(); emit('search') }
function handleSearchInput(event) {
  if (!event.target.value) clearSearch()
}
function clearSearch() {
  emit('update:query', '')
  emit('search')
  searchInput.value?.focus?.()
}
function chooseDevLocation(event) {
  const location = devLocations.find((item) => item.name === event.target.value)
  if (!location) return
  event.target.value = ''
  categoryOpen.value = false
  emit('dev-location', location)
}
function addCategory(category) {
  const next = props.selectedCategories.includes(category) ? props.selectedCategories : [...props.selectedCategories, category]
  categoryOpen.value = false
  emit('close-detail')
  emit('category-change', next)
}
function removeCategory(category) {
  categoryOpen.value = false
  emit('close-detail')
  emit('category-change', props.selectedCategories.filter((item) => item !== category))
}
function pinStyle(vendor, index) {
  const projected = domPinPositions.value[vendor.id]
  if (projected) return { left: `${projected.x}px`, top: `${projected.y}px` }
  const x = Number.isFinite(vendor.x) ? vendor.x : 20 + (index * 17) % 64
  const y = Number.isFinite(vendor.y) ? vendor.y : 28 + (index * 23) % 48
  return { left: `${x}%`, top: `${y}%` }
}
function selectDomVendor(id) {
  ignoreNextMapClick = true
  pinPositionFrozen = true
  categoryOpen.value = false
  emit('select', id)
  setTimeout(() => { ignoreNextMapClick = false }, 160)
}
function businessText(vendor) { return vendor.isOpen ? `营业至 ${vendor.endText || '今日收摊'}` : '当前未营业' }
function validPoint(point) { return Number.isFinite(Number(point?.lat)) && Number.isFinite(Number(point?.lng)) && Number(point.lat) !== 0 && Number(point.lng) !== 0 }
function mapCenter() {
  if (selectedVendor.value && validPoint(selectedVendor.value)) return selectedVendor.value
  const valid = props.vendors.filter(validPoint)
  if (valid.length) return { lat: valid.reduce((sum, item) => sum + Number(item.lat), 0) / valid.length, lng: valid.reduce((sum, item) => sum + Number(item.lng), 0) / valid.length }
  if (props.userLocation && validPoint(props.userLocation)) return props.userLocation
  return { lat: 22.3193, lng: 114.1694 }
}
function clearMarkers() {
  if (vendorMarkers) vendorMarkers.setMap(null)
  if (userMarker) userMarker.setMap(null)
  vendorMarkers = null
  userMarker = null
}
function readBounds() {
  if (!map || typeof map.getBounds !== 'function') return null
  const bounds = map.getBounds()
  const sw = bounds?.getSouthWest?.() || bounds?.southWest
  const ne = bounds?.getNorthEast?.() || bounds?.northEast
  if (!sw || !ne) return null
  return { min_lat: Number(sw.lat), max_lat: Number(ne.lat), min_lng: Number(sw.lng), max_lng: Number(ne.lng), zoom: typeof map.getZoom === 'function' ? map.getZoom() : undefined }
}
function scheduleViewportLoad() {
  if (props.focused || !viewportUserIntent) return
  clearTimeout(viewportTimer)
  viewportTimer = setTimeout(() => {
    const bounds = readBounds()
    if (bounds && Object.values(bounds).some((value) => Number.isFinite(Number(value)))) emit('viewport-change', bounds)
  }, 450)
}
function emitCurrentViewport(delay = 120) {
  if (props.focused) return
  setTimeout(() => {
    const bounds = readBounds()
    if (bounds && Object.values(bounds).some((value) => Number.isFinite(Number(value)))) emit('viewport-change', bounds)
  }, delay)
}
async function initMap() {
  if (!hasTencentMapKey() || !mapEl.value) {
    mapStatus.value = 'fallback'
    emit('viewport-change', null)
    return
  }
  try {
    TMap = await loadTencentMap()
    const center = mapCenter()
    map = new TMap.Map(mapEl.value, { center: new TMap.LatLng(Number(center.lat), Number(center.lng)), zoom: props.focused ? 18 : 17, viewMode: '2D' })
    mapEl.value?.addEventListener('pointerdown', markViewportIntent, { passive: true })
    mapEl.value?.addEventListener('wheel', markViewportIntent, { passive: true })
    mapEl.value?.addEventListener('touchstart', markViewportIntent, { passive: true })
    mapClickHandler = () => {
      if (ignoreNextMapClick) {
        ignoreNextMapClick = false
        return
      }
      categoryOpen.value = false
      emit('close-detail')
    }
    map.on?.('click', mapClickHandler)
    for (const eventName of ['dragend', 'zoomend']) {
      const handler = () => scheduleViewportLoad()
      map.on?.(eventName, handler)
      viewportHandlers.push([eventName, handler])
    }
    for (const eventName of ['drag', 'dragend', 'zoom', 'zoomend', 'bounds_changed']) {
      const handler = () => updateDomPinPositions()
      map.on?.(eventName, handler)
      viewportHandlers.push([eventName, handler])
    }
    mapStatus.value = 'ready'
    renderMarkers(true)
    emitCurrentViewport()
    schedulePositionRefresh()
  } catch {
    mapStatus.value = 'error'
    emit('viewport-change', null)
  }
}
function markViewportIntent() {
  viewportUserIntent = true
  pinPositionFrozen = false
}
function renderMarkers(recenter = false) {
  if (!map || !TMap) return
  const validVendors = props.vendors.filter(validPoint)
  if (recenter) {
    const center = mapCenter()
    map.setCenter(new TMap.LatLng(Number(center.lat), Number(center.lng)))
    if (typeof map.setZoom === 'function') map.setZoom(props.focused || validVendors.length <= 1 ? 18 : 17)
  }
  clearMarkers()
  updateDomPinPositions()
  if (props.userLocation && validPoint(props.userLocation)) {
    userMarker = new TMap.MultiMarker({
      map,
      styles: { user: new TMap.MarkerStyle({ width: 28, height: 28, anchor: { x: 14, y: 14 }, src: userMarkerIconDataUri() }) },
      geometries: [{ id: 'user-location', styleId: 'user', position: new TMap.LatLng(Number(props.userLocation.lat), Number(props.userLocation.lng)), rank: 10000 }]
    })
  }
}
function updateDomPinPositions() {
  if (pinPositionFrozen) return
  if (!map || !TMap || typeof map.projectToContainer !== 'function') return
  const next = {}
  for (const vendor of props.vendors) {
    if (!validPoint(vendor)) continue
    const point = map.projectToContainer(new TMap.LatLng(Number(vendor.lat), Number(vendor.lng)))
    if (point && Number.isFinite(Number(point.x)) && Number.isFinite(Number(point.y))) next[vendor.id] = { x: Number(point.x), y: Number(point.y) }
  }
  domPinPositions.value = next
}
function schedulePositionRefresh() {
  positionTimers.forEach((timer) => clearTimeout(timer))
  positionTimers = [80, 300, 900].map((delay) => setTimeout(updateDomPinPositions, delay))
}

onMounted(initMap)
onBeforeUnmount(() => {
  clearTimeout(viewportTimer)
  positionTimers.forEach((timer) => clearTimeout(timer))
  clearMarkers()
  mapEl.value?.removeEventListener('pointerdown', markViewportIntent)
  mapEl.value?.removeEventListener('wheel', markViewportIntent)
  mapEl.value?.removeEventListener('touchstart', markViewportIntent)
  if (map && mapClickHandler) map.off?.('click', mapClickHandler)
  viewportHandlers.forEach(([eventName, handler]) => map?.off?.(eventName, handler))
  if (map && typeof map.destroy === 'function') map.destroy()
})
watch(() => props.vendors, () => { renderMarkers(false); schedulePositionRefresh() }, { deep: true })
watch(() => props.userLocation, () => {
  if (map && TMap && validPoint(props.userLocation)) {
    map.setCenter(new TMap.LatLng(Number(props.userLocation.lat), Number(props.userLocation.lng)))
    emitCurrentViewport(180)
  }
  renderMarkers(false)
  schedulePositionRefresh()
}, { deep: true })

function userMarkerIconDataUri() {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 28 28" fill="none"><circle cx="14" cy="14" r="12" fill="#2F1F0D" stroke="#FFF8EA" stroke-width="3"/><circle cx="14" cy="14" r="4" fill="#FFF8EA"/></svg>`
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}
</script>
