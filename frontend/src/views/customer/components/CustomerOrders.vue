<template>
  <section class="list-page">
    <PullSearch v-model="query" placeholder="检索订单、摊位、商品" />
    <div v-if="!filtered.length" class="c-panel empty-panel"><div><strong>{{ query ? '没有匹配订单' : '还没有预定' }}</strong><p>{{ query ? '换个关键词试试。' : '从地图选择摊位后可预定。' }}</p></div></div>
    <div v-else class="card-list">
      <article v-for="order in filtered" :key="order.id || order.order_no" class="c-panel customer-card">
        <div class="order-card-head">
          <div>
            <h2>
              <button class="order-shop-link" type="button" @click="$emit('view-shop', order.shop?.shop_code || order.shopCode)">
                {{ order.shop?.name || order.vendorName || '流动摊位' }}
              </button>
            </h2>
            <p class="muted">{{ order.order_no || '预定订单' }}</p>
          </div>
          <button v-if="canCancel(order.status)" class="c-btn secondary order-cancel" type="button" @click="$emit('cancel', order)">取消预定</button>
        </div>

        <div class="order-products" aria-label="预定商品">
          <span v-for="item in orderItems(order)" :key="item.key" class="order-product-chip">{{ item.name }} x{{ item.quantity }}</span>
        </div>

        <div class="order-card-foot">
          <div class="order-meta"><span>金额</span><strong>{{ money(order.total_amount_cents || order.total || 0) }}</strong></div>
          <div class="order-meta"><span>取货码</span><strong>{{ order.pickup_code || order.code || '待接单' }}</strong></div>
          <div class="order-meta"><span>状态</span><strong class="status-pill">{{ statusLabel(order.status) }}</strong></div>
        </div>
      </article>
    </div>
    <button class="fab" type="button" @click="$emit('feedback')">反馈</button>
  </section>
</template>
<script setup>
import { computed, ref } from 'vue'
import { money, statusText } from '../../../api/client'
import PullSearch from './PullSearch.vue'
const props = defineProps({ orders: Array })
defineEmits(['view-shop', 'cancel', 'feedback'])
const query = ref('')
const filtered = computed(() => (props.orders || []).filter((order) => !query.value || [order.shop?.name, order.vendorName, productSummary(order), order.status, statusLabel(order.status)].join(' ').includes(query.value)))
function productSummary(order) { return (order.items || []).map((item) => `${item.product_name} x${item.quantity}`).join(' / ') || order.productName || '预购商品' }
function orderItems(order) {
  if (order.items?.length) return order.items.map((item) => ({ key: item.id || `${item.product_name}-${item.quantity}`, name: item.product_name || '预购商品', quantity: item.quantity || 1 }))
  return [{ key: 'fallback', name: order.productName || '预购商品', quantity: order.quantity || 1 }]
}
function statusLabel(status) { return statusText(status) }
function canCancel(status) { return ['pending_accept', 'accepted', 'preparing', 'reserved'].includes(status) }
</script>
