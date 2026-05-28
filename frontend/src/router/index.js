import { createRouter, createWebHistory } from 'vue-router'
import Home from '../views/Home.vue'
import CustomerReference from '../views/customer/CustomerReference.vue'
import MerchantReference from '../views/merchant/MerchantReference.vue'
import AdminConsole from '../views/admin/AdminConsole.vue'

const merchantRoutes = [
  '/merchant/login',
  '/merchant/apply',
  '/merchant/application-status',
  '/merchant/onboarding',
  '/merchant/dashboard',
  '/merchant/stall-session',
  '/merchant/products',
  '/merchant/orders',
  '/merchant/qrcode'
]

const routes = [
  { path: '/', redirect: '/nearby' },
  { path: '/home', component: Home },
  { path: '/nearby', component: CustomerReference },
  { path: '/s/:shopCode', component: CustomerReference },
  { path: '/merchant', redirect: '/merchant/dashboard' },
  ...merchantRoutes.map((path) => ({ path, component: MerchantReference })),
  { path: '/admin', component: AdminConsole }
]

export default createRouter({ history: createWebHistory(), routes })
