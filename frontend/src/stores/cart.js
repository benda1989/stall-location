import { defineStore } from 'pinia'

export const useCartStore = defineStore('cart', {
  state: () => ({ shopCode: '', items: [] }),
  getters: {
    totalQuantity: (state) => state.items.reduce((sum, item) => sum + item.quantity, 0),
    totalAmount: (state) => state.items.reduce((sum, item) => sum + item.quantity * item.price_cents, 0)
  },
  actions: {
    load(shopCode) {
      const raw = localStorage.getItem(`cart:${shopCode}`)
      this.shopCode = shopCode
      this.items = raw ? JSON.parse(raw) : []
    },
    save() {
      if (this.shopCode) localStorage.setItem(`cart:${this.shopCode}`, JSON.stringify(this.items))
    },
    add(product) {
      const found = this.items.find((item) => item.product_id === product.id)
      if (found) found.quantity += 1
      else this.items.push({ product_id: product.id, name: product.name, price_cents: product.price_cents, quantity: 1 })
      this.save()
    },
    remove(productID) {
      this.items = this.items.filter((item) => item.product_id !== productID)
      this.save()
    },
    clear() {
      this.items = []
      this.save()
    }
  }
})
