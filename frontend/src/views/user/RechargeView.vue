<template>
  <AppLayout>
    <div class="mx-auto max-w-3xl space-y-6">
      <!-- Current Balance Card -->
      <div class="card overflow-hidden">
        <div class="bg-gradient-to-br from-primary-500 to-primary-600 px-6 py-8 text-center">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm"
          >
            <Icon name="creditCard" size="xl" class="text-white" />
          </div>
          <p class="text-sm font-medium text-primary-100">{{ t('recharge.currentBalance') }}</p>
          <p class="mt-2 text-4xl font-bold text-white">
            ${{ user?.balance?.toFixed(2) || '0.00' }}
          </p>
          <p class="mt-2 text-sm text-primary-100">
            {{ t('recharge.concurrency') }}: {{ user?.concurrency || 0 }} {{ t('recharge.requests') }}
          </p>
        </div>
      </div>

      <!-- Payment Not Enabled Warning -->
      <div
        v-if="!paymentConfig?.enabled"
        class="card border-amber-200 bg-amber-50 dark:border-amber-800/50 dark:bg-amber-900/20"
      >
        <div class="p-6">
          <div class="flex items-start gap-4">
            <div
              class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-amber-100 dark:bg-amber-900/30"
            >
              <Icon name="exclamationTriangle" size="md" class="text-amber-600 dark:text-amber-400" />
            </div>
            <div class="flex-1">
              <h3 class="text-sm font-semibold text-amber-800 dark:text-amber-300">
                {{ t('recharge.paymentDisabled') }}
              </h3>
              <p class="mt-2 text-sm text-amber-700 dark:text-amber-400">
                {{ t('recharge.paymentDisabledHint') }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Recharge Options -->
      <div v-else class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('recharge.selectProduct') }}
          </h2>
        </div>
        <div class="p-6">
          <!-- Loading State -->
          <div v-if="loadingProducts" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
          </div>

          <!-- Product Grid -->
          <div v-else-if="products.length > 0" class="space-y-6">
            <!-- Balance Products -->
            <div v-if="balanceProducts.length > 0">
              <h3 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('recharge.balanceProducts') }}
              </h3>
              <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
                <button
                  v-for="product in balanceProducts"
                  :key="product.id"
                  @click="selectProduct(product)"
                  :class="[
                    'relative rounded-xl border-2 p-4 text-left transition-all',
                    selectedProduct?.id === product.id
                      ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                      : 'border-gray-200 hover:border-primary-300 dark:border-dark-600 dark:hover:border-primary-700'
                  ]"
                >
                  <div class="text-lg font-bold text-gray-900 dark:text-white">
                    ¥{{ product.price_cny.toFixed(2) }}
                  </div>
                  <div class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                    {{ t('recharge.getValue') }} ${{ product.value.toFixed(2) }}
                  </div>
                  <div
                    v-if="selectedProduct?.id === product.id"
                    class="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-primary-500"
                  >
                    <Icon name="check" size="xs" class="text-white" />
                  </div>
                </button>
              </div>
            </div>

            <!-- Subscription Products -->
            <div v-if="subscriptionProducts.length > 0">
              <h3 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('recharge.subscriptionProducts') }}
              </h3>
              <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <button
                  v-for="product in subscriptionProducts"
                  :key="product.id"
                  @click="selectProduct(product)"
                  :class="[
                    'relative rounded-xl border-2 p-4 text-left transition-all',
                    selectedProduct?.id === product.id
                      ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                      : 'border-gray-200 hover:border-primary-300 dark:border-dark-600 dark:hover:border-primary-700'
                  ]"
                >
                  <div class="flex items-center justify-between">
                    <div>
                      <div class="text-lg font-bold text-gray-900 dark:text-white">
                        {{ product.name }}
                      </div>
                      <div class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                        {{ product.description || t('recharge.subscriptionDesc', { days: product.value }) }}
                      </div>
                    </div>
                    <div class="text-right">
                      <div class="text-xl font-bold text-primary-600 dark:text-primary-400">
                        ¥{{ product.price_cny.toFixed(2) }}
                      </div>
                      <div v-if="product.group" class="mt-1 text-xs text-gray-400">
                        {{ product.group.name }}
                      </div>
                    </div>
                  </div>
                  <div
                    v-if="selectedProduct?.id === product.id"
                    class="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-primary-500"
                  >
                    <Icon name="check" size="xs" class="text-white" />
                  </div>
                </button>
              </div>
            </div>

            <!-- Custom Amount -->
            <div>
              <h3 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('recharge.customAmount') }}
              </h3>
              <div class="flex items-center gap-3">
                <div class="relative flex-1">
                  <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                    <span class="text-gray-500 dark:text-dark-400">¥</span>
                  </div>
                  <input
                    v-model.number="customAmount"
                    type="number"
                    :min="paymentConfig?.min_amount || 10"
                    :max="paymentConfig?.max_amount || 10000"
                    :placeholder="t('recharge.customAmountPlaceholder', { min: paymentConfig?.min_amount || 10, max: paymentConfig?.max_amount || 10000 })"
                    @focus="selectCustomAmount"
                    class="input py-3 pl-8"
                  />
                </div>
                <div class="text-sm text-gray-500 dark:text-dark-400">
                  ≈ ${{ (customAmount || 0).toFixed(2) }}
                </div>
              </div>
              <p class="mt-2 text-xs text-gray-400 dark:text-dark-500">
                {{ t('recharge.customAmountHint', { rate: paymentConfig?.cny_usd_rate || 1 }) }}
              </p>
            </div>

            <!-- Payment Method -->
            <div>
              <h3 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('recharge.paymentMethod') }}
              </h3>
              <div class="flex gap-3">
                <button
                  v-if="paymentConfig?.payment_methods?.includes('alipay')"
                  @click="paymentMethod = 'alipay'"
                  :class="[
                    'flex flex-1 items-center justify-center gap-2 rounded-xl border-2 py-3 transition-all',
                    paymentMethod === 'alipay'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 hover:border-blue-300 dark:border-dark-600'
                  ]"
                >
                  <svg class="h-6 w-6" viewBox="0 0 24 24" fill="#1677FF">
                    <path d="M21.422 15.358c-.263-.054-3.58-1.198-5.847-2.123.948-2.124 1.58-4.51 1.743-7.023H9.354v-1.57h4.766V3.071H9.354V1.5H7.57v1.571H2.782v1.571H7.57v1.571H4.565v1.429h9.02c-.142 1.875-.627 3.696-1.391 5.339-1.838-.666-3.863-1.189-5.99-1.5l-.364 1.429c2.13.318 4.15.839 5.99 1.5-1.564 2.589-3.882 4.607-6.733 5.624l.728 1.429c3.136-1.18 5.717-3.473 7.444-6.392 2.464 1.044 5.85 2.267 6.153 2.355 1.054.304 1.875-.108 2.018-.875.142-.767-.356-1.393-1.018-1.693z"/>
                  </svg>
                  <span class="font-medium text-gray-900 dark:text-white">{{ t('recharge.alipay') }}</span>
                </button>
                <button
                  v-if="paymentConfig?.payment_methods?.includes('wechat')"
                  @click="paymentMethod = 'wechat'"
                  :class="[
                    'flex flex-1 items-center justify-center gap-2 rounded-xl border-2 py-3 transition-all',
                    paymentMethod === 'wechat'
                      ? 'border-green-500 bg-green-50 dark:bg-green-900/20'
                      : 'border-gray-200 hover:border-green-300 dark:border-dark-600'
                  ]"
                >
                  <svg class="h-6 w-6" viewBox="0 0 24 24" fill="#07C160">
                    <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178A1.17 1.17 0 014.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178 1.17 1.17 0 01-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 01.598.082l1.584.926a.272.272 0 00.14.045c.134 0 .24-.11.24-.245 0-.06-.023-.118-.039-.177l-.325-1.233a.49.49 0 01.177-.554c1.522-1.12 2.499-2.77 2.499-4.6 0-3.398-2.928-6.143-7.06-6.143zm-2.44 3.316c.535 0 .969.44.969.983a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.983.97-.983zm4.844 0c.535 0 .969.44.969.983a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.983.97-.983z"/>
                  </svg>
                  <span class="font-medium text-gray-900 dark:text-white">{{ t('recharge.wechat') }}</span>
                </button>
              </div>
            </div>

            <!-- Submit Button -->
            <button
              @click="handleCreateOrder"
              :disabled="!canSubmit || submitting"
              class="btn btn-primary w-full py-3"
            >
              <svg
                v-if="submitting"
                class="-ml-1 mr-2 h-5 w-5 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
              {{ submitting ? t('recharge.creating') : t('recharge.payNow', { amount: displayAmount }) }}
            </button>

            <!-- Audit Warning -->
            <p
              v-if="needsAudit"
              class="text-center text-xs text-amber-600 dark:text-amber-400"
            >
              {{ t('recharge.auditWarning', { threshold: paymentConfig?.audit_threshold || 1000 }) }}
            </p>
          </div>

          <!-- No Products -->
          <div v-else class="empty-state py-8">
            <div
              class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800"
            >
              <Icon name="shoppingBag" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('recharge.noProducts') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Recent Orders -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('recharge.recentOrders') }}
          </h2>
        </div>
        <div class="p-6">
          <!-- Loading State -->
          <div v-if="loadingOrders" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
          </div>

          <!-- Orders List -->
          <div v-else-if="orders.length > 0" class="space-y-3">
            <div
              v-for="order in orders"
              :key="order.id"
              class="flex items-center justify-between rounded-xl bg-gray-50 p-4 dark:bg-dark-800"
            >
              <div class="flex items-center gap-4">
                <div
                  :class="[
                    'flex h-10 w-10 items-center justify-center rounded-xl',
                    getOrderStatusColor(order.status).bg
                  ]"
                >
                  <Icon
                    :name="getOrderStatusIcon(order.status)"
                    size="md"
                    :class="getOrderStatusColor(order.status).text"
                  />
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ order.order_type === 'balance' ? t('recharge.balanceRecharge') : t('recharge.subscriptionPurchase') }}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ formatDateTime(order.created_at) }}
                  </p>
                </div>
              </div>
              <div class="text-right">
                <p class="text-sm font-semibold text-gray-900 dark:text-white">
                  ¥{{ order.amount_cny.toFixed(2) }}
                </p>
                <p
                  :class="[
                    'text-xs',
                    getOrderStatusColor(order.status).text
                  ]"
                >
                  {{ t(`recharge.status.${order.status}`) }}
                </p>
              </div>
            </div>
          </div>

          <!-- Empty State -->
          <div v-else class="empty-state py-8">
            <div
              class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800"
            >
              <Icon name="receipt" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('recharge.noOrders') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Information Card -->
      <div
        class="card border-primary-200 bg-primary-50 dark:border-primary-800/50 dark:bg-primary-900/20"
      >
        <div class="p-6">
          <div class="flex items-start gap-4">
            <div
              class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-100 dark:bg-primary-900/30"
            >
              <Icon name="infoCircle" size="md" class="text-primary-600 dark:text-primary-400" />
            </div>
            <div class="flex-1">
              <h3 class="text-sm font-semibold text-primary-800 dark:text-primary-300">
                {{ t('recharge.aboutRecharge') }}
              </h3>
              <ul
                class="mt-2 list-inside list-disc space-y-1 text-sm text-primary-700 dark:text-primary-400"
              >
                <li>{{ t('recharge.rule1') }}</li>
                <li>{{ t('recharge.rule2') }}</li>
                <li>{{ t('recharge.rule3') }}</li>
                <li>{{ t('recharge.rule4') }}</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { paymentAPI, type Product, type PaymentOrder, type PaymentConfigResponse } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const user = computed(() => authStore.user)

// State
const paymentConfig = ref<PaymentConfigResponse | null>(null)
const products = ref<Product[]>([])
const orders = ref<PaymentOrder[]>([])
const loadingProducts = ref(false)
const loadingOrders = ref(false)
const submitting = ref(false)

const selectedProduct = ref<Product | null>(null)
const customAmount = ref<number | null>(null)
const paymentMethod = ref<'alipay' | 'wechat'>('alipay')

// Polling for order status
let pollingTimer: ReturnType<typeof setInterval> | null = null
const pendingOrderNo = ref<string | null>(null)

// Computed
const balanceProducts = computed(() =>
  products.value.filter((p) => p.type === 'balance')
)

const subscriptionProducts = computed(() =>
  products.value.filter((p) => p.type === 'subscription')
)

const displayAmount = computed(() => {
  if (selectedProduct.value) {
    return `¥${selectedProduct.value.price_cny.toFixed(2)}`
  }
  if (customAmount.value && customAmount.value > 0) {
    return `¥${customAmount.value.toFixed(2)}`
  }
  return '¥0'
})

const canSubmit = computed(() => {
  if (!paymentMethod.value) return false
  if (selectedProduct.value) return true
  if (customAmount.value && customAmount.value >= (paymentConfig.value?.min_amount || 10)) {
    return customAmount.value <= (paymentConfig.value?.max_amount || 10000)
  }
  return false
})

const needsAudit = computed(() => {
  const threshold = paymentConfig.value?.audit_threshold || 1000
  if (selectedProduct.value) {
    return selectedProduct.value.price_cny >= threshold
  }
  if (customAmount.value) {
    return customAmount.value >= threshold
  }
  return false
})

// Methods
const selectProduct = (product: Product) => {
  selectedProduct.value = product
  customAmount.value = null
}

const selectCustomAmount = () => {
  selectedProduct.value = null
}

const getOrderStatusColor = (status: string) => {
  switch (status) {
    case 'paid':
      return {
        bg: 'bg-emerald-100 dark:bg-emerald-900/30',
        text: 'text-emerald-600 dark:text-emerald-400'
      }
    case 'pending':
      return {
        bg: 'bg-amber-100 dark:bg-amber-900/30',
        text: 'text-amber-600 dark:text-amber-400'
      }
    case 'auditing':
      return {
        bg: 'bg-blue-100 dark:bg-blue-900/30',
        text: 'text-blue-600 dark:text-blue-400'
      }
    case 'failed':
    case 'refunded':
      return {
        bg: 'bg-red-100 dark:bg-red-900/30',
        text: 'text-red-600 dark:text-red-400'
      }
    default:
      return {
        bg: 'bg-gray-100 dark:bg-gray-900/30',
        text: 'text-gray-600 dark:text-gray-400'
      }
  }
}

const getOrderStatusIcon = (status: string) => {
  switch (status) {
    case 'paid':
      return 'checkCircle'
    case 'pending':
      return 'clock'
    case 'auditing':
      return 'eye'
    case 'failed':
      return 'xCircle'
    case 'refunded':
      return 'arrowUturnLeft'
    default:
      return 'questionCircle'
  }
}

const fetchPaymentConfig = async () => {
  try {
    paymentConfig.value = await paymentAPI.getPaymentConfig()
    // Set default payment method
    if (paymentConfig.value.payment_methods?.length > 0) {
      paymentMethod.value = paymentConfig.value.payment_methods[0] as 'alipay' | 'wechat'
    }
  } catch (error) {
    console.error('Failed to fetch payment config:', error)
  }
}

const fetchProducts = async () => {
  loadingProducts.value = true
  try {
    products.value = await paymentAPI.getProducts()
  } catch (error) {
    console.error('Failed to fetch products:', error)
  } finally {
    loadingProducts.value = false
  }
}

const fetchOrders = async () => {
  loadingOrders.value = true
  try {
    const result = await paymentAPI.getOrderHistory({ page: 1, page_size: 10 })
    orders.value = result.items
  } catch (error) {
    console.error('Failed to fetch orders:', error)
  } finally {
    loadingOrders.value = false
  }
}

const handleCreateOrder = async () => {
  if (!canSubmit.value) return

  submitting.value = true
  try {
    const request = {
      product_id: selectedProduct.value?.id,
      custom_amount: selectedProduct.value ? undefined : customAmount.value || undefined,
      payment_method: paymentMethod.value
    }

    const result = await paymentAPI.createOrder(request)

    // Store pending order for polling
    pendingOrderNo.value = result.order_no

    // Open payment URL in new window
    window.open(result.payment_url, '_blank')

    // Start polling for order status
    startPolling()

    // Show success message
    appStore.showSuccess(t('recharge.orderCreated'))

    // Refresh orders
    await fetchOrders()

    // Reset selection
    selectedProduct.value = null
    customAmount.value = null
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('recharge.createOrderFailed'))
  } finally {
    submitting.value = false
  }
}

const startPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer)
  }

  pollingTimer = setInterval(async () => {
    if (!pendingOrderNo.value) {
      stopPolling()
      return
    }

    try {
      const order = await paymentAPI.getOrderStatus(pendingOrderNo.value)
      if (order.status === 'paid') {
        appStore.showSuccess(t('recharge.paymentSuccess'))
        await authStore.refreshUser()
        await fetchOrders()
        stopPolling()
      } else if (order.status === 'failed' || order.status === 'refunded') {
        stopPolling()
      }
    } catch (error) {
      console.error('Failed to poll order status:', error)
    }
  }, 3000) // Poll every 3 seconds
}

const stopPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
  pendingOrderNo.value = null
}

onMounted(async () => {
  await fetchPaymentConfig()
  if (paymentConfig.value?.enabled) {
    await Promise.all([fetchProducts(), fetchOrders()])
  }
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
