<template>
  <section class="list-page">
    <PullSearch v-model="query" placeholder="检索收藏摊位" />
    <div v-if="!filtered.length" class="c-panel empty-panel"><div><strong>{{ query ? '没有匹配收藏' : '暂无收藏' }}</strong><p>{{ query ? '换个关键词试试。' : '在摊位详情点收藏后会出现在这里。' }}</p></div></div>
    <div v-else class="card-list">
      <article v-for="vendor in filtered" :key="vendor.id" class="c-panel customer-card">
        <div class="card-head">
          <div>
            <h2><button class="card-title-link" type="button" @click="$emit('view', vendor.id)">{{ vendor.name }}</button></h2>
            <p class="muted">{{ vendor.category }} · {{ vendor.address }}</p>
          </div>
          <span :class="['status-pill', vendor.isOpen && 'open']">{{ vendor.statusText }}</span>
        </div>
        <div class="favorite-card-foot">
          <span v-if="vendor.orderCount" class="favorite-order-count">订单 {{ vendor.orderCount }}</span>
          <div class="card-actions"><button class="c-btn secondary" type="button" @click="$emit('remove', vendor.id)">取消收藏</button><button class="c-btn primary" type="button" @click="$emit('preorder', vendor.id)">预定</button></div>
        </div>
      </article>
    </div>
    <button class="fab" type="button" @click="$emit('join')">加入</button>
  </section>
</template>
<script setup>
import { computed, ref } from 'vue'
import PullSearch from './PullSearch.vue'
const props = defineProps({ vendors: Array })
defineEmits(['view', 'remove', 'preorder', 'join'])
const query = ref('')
const filtered = computed(() => (props.vendors || []).filter((vendor) => !query.value || [vendor.name, vendor.category, vendor.address, vendor.statusText].join(' ').includes(query.value)))
</script>
