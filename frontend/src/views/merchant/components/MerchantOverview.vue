<template>
  <section class="m-grid">
    <article :class="['m-panel', 'm-hero', session ? 'is-live' : 'is-idle']" :style="heroStyle">
      <div v-if="session" class="m-hero-live-copy">
        <span class="m-pill">出摊中</span>
        <strong>截止 {{ time(session.expected_end_at) }}</strong>
      </div>
      <button class="m-btn primary m-hero-action" @click="$emit(session ? 'end-session' : 'open-session')">
        {{ session ? '结束出摊' : '出摊' }}
      </button>
    </article>

    <section class="m-kpis">
      <article class="m-panel m-kpi"><span>待处理预定</span><strong>{{ dashboard.pending_orders || 0 }}</strong></article>
      <article class="m-panel m-kpi"><span>今日预定</span><strong>{{ dashboard.today_orders || 0 }}</strong></article>
      <article class="m-panel m-kpi"><span>可售商品</span><strong>{{ products.filter(p => p.status === 'on_sale' && p.stock > 0).length }}</strong></article>
      <article class="m-panel m-kpi"><span>低库存</span><strong>{{ products.filter(p => p.stock <= 3).length }}</strong></article>
    </section>

    <section class="m-panel m-card">
      <div class="m-card-head"><div><h2>{{ shop?.name || '店铺' }}</h2></div><button class="m-btn secondary" @click="$emit('copy-link')">复制</button></div>
      <img v-if="qrDataUrl" class="m-qr" :src="qrDataUrl" alt="店铺二维码">
      <p class="muted">{{ qrUrl || '店铺链接生成中…' }}</p>
    </section>
  </section>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({ dashboard: Object, shop: Object, session: Object, products: { type: Array, default: () => [] }, qrUrl: String, qrDataUrl: String })
defineEmits(['open-session', 'end-session', 'copy-link'])

const heroStyle = computed(() => {
  const photo = props.session?.photo_url
  return photo ? { '--stall-photo': `url("${String(photo).replaceAll('"', '%22')}")` } : {}
})

function time(value) {
  return value ? new Date(value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : '—'
}
</script>
