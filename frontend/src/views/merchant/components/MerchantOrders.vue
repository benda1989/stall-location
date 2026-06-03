<template>
  <section class="m-grid">
    <MerchantPullSearch v-model="query" placeholder="检索顾客、订单号、取货码、商品" />

    <section class="m-panel m-order-tools">
      <div class="m-status-filters" aria-label="订单状态筛选">
        <button v-for="item in statusFilters" :key="item.value" :class="['m-filter-chip', activeStatus === item.value && 'is-active']" type="button" @click="activeStatus = item.value">
          <span>{{ item.label }}</span><b>{{ countByStatus(item.value) }}</b>
        </button>
      </div>
    </section>

    <div v-if="!filteredOrders.length" class="m-panel m-card muted">{{ emptyText }}</div>
    <div v-else class="m-list">
      <article v-for="order in filteredOrders" :key="order.id" class="m-panel m-card">
        <div class="m-card-head">
          <div>
            <h2>{{ order.customer_name }}</h2>
            <p class="muted">{{ order.order_no }} · 取货码 {{ order.pickup_code }}</p>
          </div>
          <span :class="['m-pill', ['rejected','canceled'].includes(order.status) && 'red']">{{ statusText(order.status) }}</span>
        </div>
        <p>{{ productSummary(order) }}</p>
        <div class="m-card-actions"><button v-for="action in actions(order.status)" :key="action.key" :class="['m-btn', action.tone || 'secondary']" @click="$emit('order-action', order, action.key)">{{ action.label }}</button></div>
      </article>
    </div>
    <button class="m-feedback-fab" @click="$emit('feedback')">反馈</button>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { statusText } from '../../../api/client'
import MerchantPullSearch from './MerchantPullSearch.vue'

const props = defineProps({ orders: Array })
defineEmits(['order-action', 'feedback'])
const query = ref('')
const activeStatus = ref('pending')
const statusFilters = [
  { value: 'pending', label: '待处理' },
  { value: 'accepted', label: '已接单' },
  { value: 'ready', label: '可取货' },
  { value: 'all', label: '全部' }
]
const acceptedStatuses = ['accepted', 'preparing']
const filteredOrders = computed(() => (props.orders || [])
  .filter((order) => matchStatus(order.status))
  .filter((order) => matchQuery(order)))
const emptyText = computed(() => query.value ? '没有匹配订单，换个关键词试试。' : '当前筛选下没有订单。')

function matchStatus(status) {
  if (activeStatus.value === 'all') return true
  if (activeStatus.value === 'pending') return status === 'pending_accept'
  if (activeStatus.value === 'accepted') return acceptedStatuses.includes(status)
  if (activeStatus.value === 'ready') return status === 'ready'
  return true
}
function matchQuery(order) {
  const keyword = query.value.trim().toLowerCase()
  if (!keyword) return true
  return [order.customer_name, order.customer_phone, order.order_no, order.pickup_code, order.status, statusText(order.status), productSummary(order)]
    .join(' ')
    .toLowerCase()
    .includes(keyword)
}
function countByStatus(value) {
  return (props.orders || []).filter((order) => {
    if (value === 'all') return true
    if (value === 'pending') return order.status === 'pending_accept'
    if (value === 'accepted') return acceptedStatuses.includes(order.status)
    if (value === 'ready') return order.status === 'ready'
    return true
  }).length
}
function productSummary(order) { return (order.items || []).map((item) => `${item.product_name} x${item.quantity}`).join(' / ') || '预购商品' }
function actions(status) {
  if (status === 'pending_accept') return [{ key: 'accept', label: '接单', tone: 'primary' }, { key: 'reject', label: '拒单', tone: 'danger' }]
  if (status === 'accepted') return [{ key: 'prepare', label: '开始备货', tone: 'primary' }]
  if (status === 'preparing') return [{ key: 'ready', label: '标记可取', tone: 'primary' }]
  if (status === 'ready') return [{ key: 'complete', label: '核销完成', tone: 'primary' }]
  return []
}
</script>
