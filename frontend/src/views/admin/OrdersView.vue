<template>
  <AppLayout>
    <TablePageLayout>
      <template #header>
        <!-- Stats Cards -->
        <div class="mb-6 grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div class="card p-4">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.todayOrders') }}</p>
            <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
              {{ stats.today_orders }}
            </p>
            <p class="mt-1 text-sm text-primary-600 dark:text-primary-400">
              ¥{{ stats.today_amount.toFixed(2) }}
            </p>
          </div>
          <div class="card p-4">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.totalPaid') }}</p>
            <p class="mt-1 text-2xl font-bold text-emerald-600 dark:text-emerald-400">
              {{ stats.paid_orders }}
            </p>
            <p class="mt-1 text-sm text-emerald-600 dark:text-emerald-400">
              ¥{{ stats.paid_amount.toFixed(2) }}
            </p>
          </div>
          <div class="card p-4">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.pendingOrders') }}</p>
            <p class="mt-1 text-2xl font-bold text-amber-600 dark:text-amber-400">
              {{ stats.pending_orders }}
            </p>
            <p class="mt-1 text-sm text-amber-600 dark:text-amber-400">
              ¥{{ stats.pending_amount.toFixed(2) }}
            </p>
          </div>
          <div class="card p-4">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.totalOrders') }}</p>
            <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
              {{ stats.total_orders }}
            </p>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              ¥{{ stats.total_amount.toFixed(2) }}
            </p>
          </div>
        </div>
      </template>

      <template #actions>
        <div class="flex justify-end gap-3">
          <button
            v-if="selectedOrders.length > 0"
            @click="handleBatchDelete"
            class="btn btn-danger"
          >
            <Icon name="trash" size="sm" class="mr-1.5" />
            {{ t('admin.orders.batchDelete') }} ({{ selectedOrders.length }})
          </button>
          <button
            @click="loadOrders"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="max-w-md flex-1">
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('admin.orders.searchOrders')"
              class="input"
              @input="handleSearch"
            />
          </div>
          <div class="flex flex-wrap gap-2">
            <Select
              v-model="filters.status"
              :options="statusOptions"
              class="w-32"
              @change="loadOrders"
            />
            <Select
              v-model="filters.order_type"
              :options="typeOptions"
              class="w-32"
              @change="loadOrders"
            />
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="orders" :loading="loading">
          <template #header-select>
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allDeletableSelected"
              :disabled="deletableOrders.length === 0"
              @click.stop
              @change="toggleSelectAllDeletable"
            />
          </template>

          <template #cell-select="{ row }">
            <input
              v-if="canDeleteOrder(row.status)"
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedOrderIds.has(row.id)"
              @click.stop
              @change="toggleSelectRow(row.id, $event)"
            />
          </template>

          <template #cell-order_no="{ value }">
            <code class="font-mono text-sm text-gray-900 dark:text-gray-100">
              {{ value }}
            </code>
          </template>

          <template #cell-user="{ row }">
            <div v-if="row.user">
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ row.user.email }}
              </p>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                ID: {{ row.user_id }}
              </p>
            </div>
            <span v-else class="text-gray-400">-</span>
          </template>

          <template #cell-order_type="{ value }">
            <span
              :class="[
                'badge',
                value === 'balance' ? 'badge-blue' : 'badge-purple'
              ]"
            >
              {{ t(`admin.orders.type.${value}`) }}
            </span>
          </template>

          <template #cell-amount_cny="{ value }">
            <span class="font-semibold text-gray-900 dark:text-white">
              ¥{{ value.toFixed(2) }}
            </span>
          </template>

          <template #cell-amount_value="{ value, row }">
            <span class="text-sm text-gray-600 dark:text-gray-300">
              <template v-if="row.order_type === 'balance'">
                ${{ value.toFixed(2) }}
              </template>
              <template v-else>
                {{ value }} {{ t('admin.orders.days') }}
              </template>
            </span>
          </template>

          <template #cell-payment_method="{ value }">
            <div class="flex items-center gap-1.5">
              <svg v-if="value === 'alipay'" class="h-4 w-4" viewBox="0 0 24 24" fill="#1677FF">
                <path d="M21.422 15.358c-.263-.054-3.58-1.198-5.847-2.123.948-2.124 1.58-4.51 1.743-7.023H9.354v-1.57h4.766V3.071H9.354V1.5H7.57v1.571H2.782v1.571H7.57v1.571H4.565v1.429h9.02c-.142 1.875-.627 3.696-1.391 5.339-1.838-.666-3.863-1.189-5.99-1.5l-.364 1.429c2.13.318 4.15.839 5.99 1.5-1.564 2.589-3.882 4.607-6.733 5.624l.728 1.429c3.136-1.18 5.717-3.473 7.444-6.392 2.464 1.044 5.85 2.267 6.153 2.355 1.054.304 1.875-.108 2.018-.875.142-.767-.356-1.393-1.018-1.693z"/>
              </svg>
              <svg v-else-if="value === 'wechat'" class="h-4 w-4" viewBox="0 0 24 24" fill="#07C160">
                <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348z"/>
              </svg>
              <span class="text-sm text-gray-600 dark:text-gray-300">
                {{ t(`admin.orders.method.${value}`) }}
              </span>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', getStatusClass(value)]">
              {{ t(`admin.orders.status.${value}`) }}
            </span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
              <button
                @click="handleViewDetail(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="t('common.viewDetail')"
              >
                <Icon name="eye" size="sm" />
              </button>
              <button
                v-if="row.status === 'auditing'"
                @click="handleApprove(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400"
                :title="t('admin.orders.approve')"
              >
                <Icon name="checkCircle" size="sm" />
              </button>
              <button
                v-if="row.status === 'auditing'"
                @click="handleReject(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('admin.orders.reject')"
              >
                <Icon name="xCircle" size="sm" />
              </button>
              <button
                v-if="row.status === 'pending'"
                @click="handleManualFulfill(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-400"
                :title="t('admin.orders.manualFulfill')"
              >
                <Icon name="handRaised" size="sm" />
              </button>
              <button
                v-if="canDeleteOrder(row.status)"
                @click="handleDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('common.delete')"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Order Detail Dialog -->
    <BaseDialog
      :show="showDetailDialog"
      :title="t('admin.orders.orderDetail')"
      width="normal"
      @close="showDetailDialog = false"
    >
      <div v-if="detailOrder" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.orderNo') }}</p>
            <p class="font-mono text-sm font-medium text-gray-900 dark:text-white">
              {{ detailOrder.order_no }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.tradeNo') }}</p>
            <p class="font-mono text-sm font-medium text-gray-900 dark:text-white">
              {{ detailOrder.trade_no || '-' }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.userEmail') }}</p>
            <p class="text-sm font-medium text-gray-900 dark:text-white">
              {{ detailOrder.user?.email || '-' }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('common.status') }}</p>
            <span :class="['badge', getStatusClass(detailOrder.status)]">
              {{ t(`admin.orders.status.${detailOrder.status}`) }}
            </span>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.amount') }}</p>
            <p class="text-lg font-bold text-gray-900 dark:text-white">
              ¥{{ detailOrder.amount_cny.toFixed(2) }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.value') }}</p>
            <p class="text-sm font-medium text-gray-900 dark:text-white">
              <template v-if="detailOrder.order_type === 'balance'">
                ${{ detailOrder.amount_value.toFixed(2) }}
              </template>
              <template v-else>
                {{ detailOrder.amount_value }} {{ t('admin.orders.days') }}
              </template>
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.paymentMethod') }}</p>
            <p class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t(`admin.orders.method.${detailOrder.payment_method}`) }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('common.createdAt') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">
              {{ formatDateTime(detailOrder.created_at) }}
            </p>
          </div>
          <div v-if="detailOrder.paid_at" class="col-span-2">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.paidAt') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">
              {{ formatDateTime(detailOrder.paid_at) }}
            </p>
          </div>
          <div v-if="detailOrder.remark" class="col-span-2">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.orders.remark') }}</p>
            <p class="text-sm text-gray-900 dark:text-white">
              {{ detailOrder.remark }}
            </p>
          </div>
        </div>
      </div>
    </BaseDialog>

    <!-- Action Confirmation Dialog -->
    <BaseDialog
      :show="showActionDialog"
      :title="actionDialogTitle"
      width="narrow"
      @close="showActionDialog = false"
    >
      <div class="space-y-4">
        <p class="text-gray-600 dark:text-gray-400">
          {{ actionDialogMessage }}
        </p>
        <!-- Reject: reason field (required) -->
        <div v-if="actionType === 'reject'">
          <label class="input-label">
            {{ t('admin.orders.rejectReason') }}
            <span class="ml-1 text-xs font-normal text-red-500">*</span>
          </label>
          <textarea
            v-model="actionRejectReason"
            class="input"
            rows="2"
            :placeholder="t('admin.orders.rejectReasonPlaceholder')"
          ></textarea>
        </div>
        <!-- Fulfill: trade_no field (required) -->
        <div v-if="actionType === 'fulfill'">
          <label class="input-label">
            {{ t('admin.orders.tradeNo') }}
            <span class="ml-1 text-xs font-normal text-red-500">*</span>
          </label>
          <input
            v-model="actionTradeNo"
            type="text"
            class="input"
            :placeholder="t('admin.orders.tradeNoPlaceholder')"
          />
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="showActionDialog = false">
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            :class="['btn', actionType === 'reject' ? 'btn-danger' : 'btn-primary']"
            :disabled="submitting || !canConfirmAction"
            @click="confirmAction"
          >
            {{ t('common.confirm') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="deleteTarget ? t('admin.orders.deleteOrder') : t('admin.orders.batchDeleteTitle')"
      :message="deleteTarget
        ? t('admin.orders.deleteConfirm', { orderNo: deleteTarget.order_no })
        : t('admin.orders.batchDeleteConfirm', { count: selectedOrders.length })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      :loading="submitting"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api'
import type { PaymentOrder } from '@/api/payment'
import type { PaymentOrderWithUser, OrderStatsResponse } from '@/api/admin/orders'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import { useDebounceFn } from '@vueuse/core'

const { t } = useI18n()
const appStore = useAppStore()

// State
const orders = ref<PaymentOrderWithUser[]>([])
const stats = ref<OrderStatsResponse>({
  total_orders: 0,
  total_amount: 0,
  paid_orders: 0,
  paid_amount: 0,
  pending_orders: 0,
  pending_amount: 0,
  today_orders: 0,
  today_amount: 0
})
const loading = ref(false)
const submitting = ref(false)
const searchQuery = ref('')

const showDetailDialog = ref(false)
const showActionDialog = ref(false)
const showDeleteDialog = ref(false)
const detailOrder = ref<PaymentOrderWithUser | null>(null)
const actionTarget = ref<PaymentOrder | null>(null)
const actionType = ref<'approve' | 'reject' | 'fulfill'>('approve')
const actionRejectReason = ref('')
const actionTradeNo = ref('')
const deleteTarget = ref<PaymentOrder | null>(null)
const selectedOrderIds = ref<Set<number>>(new Set())

// Filters
const filters = reactive({
  status: '',
  order_type: ''
})

// Pagination
const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

// Table columns
const columns = computed(() => [
  { key: 'select', label: '', width: '40px' },
  { key: 'order_no', label: t('admin.orders.orderNo') },
  { key: 'user', label: t('admin.orders.user') },
  { key: 'order_type', label: t('admin.orders.orderType') },
  { key: 'amount_cny', label: t('admin.orders.amount'), sortable: true },
  { key: 'amount_value', label: t('admin.orders.value') },
  { key: 'payment_method', label: t('admin.orders.paymentMethod') },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'created_at', label: t('common.createdAt'), sortable: true },
  { key: 'actions', label: t('common.actions'), align: 'right' as const }
])

// Filter options
const statusOptions = computed(() => [
  { value: '', label: t('common.allStatus') },
  { value: 'pending', label: t('admin.orders.status.pending') },
  { value: 'paid', label: t('admin.orders.status.paid') },
  { value: 'auditing', label: t('admin.orders.status.auditing') },
  { value: 'failed', label: t('admin.orders.status.failed') },
  { value: 'refunded', label: t('admin.orders.status.refunded') }
])

const typeOptions = computed(() => [
  { value: '', label: t('admin.orders.allTypes') },
  { value: 'balance', label: t('admin.orders.type.balance') },
  { value: 'subscription', label: t('admin.orders.type.subscription') }
])

const actionDialogTitle = computed(() => {
  switch (actionType.value) {
    case 'approve':
      return t('admin.orders.approveOrder')
    case 'reject':
      return t('admin.orders.rejectOrder')
    case 'fulfill':
      return t('admin.orders.manualFulfillOrder')
    default:
      return ''
  }
})

const actionDialogMessage = computed(() => {
  switch (actionType.value) {
    case 'approve':
      return t('admin.orders.approveConfirm')
    case 'reject':
      return t('admin.orders.rejectConfirm')
    case 'fulfill':
      return t('admin.orders.fulfillConfirm')
    default:
      return ''
  }
})

const canConfirmAction = computed(() => {
  switch (actionType.value) {
    case 'approve':
      return true
    case 'reject':
      return actionRejectReason.value.trim().length > 0
    case 'fulfill':
      return actionTradeNo.value.trim().length > 0
    default:
      return false
  }
})

// Selection computed properties
const deletableStatuses = ['pending', 'auditing', 'failed']

const canDeleteOrder = (status: string) => deletableStatuses.includes(status)

const deletableOrders = computed(() =>
  orders.value.filter(order => canDeleteOrder(order.status))
)

const selectedOrders = computed(() =>
  deletableOrders.value.filter(order => selectedOrderIds.value.has(order.id))
)

const allDeletableSelected = computed(() => {
  if (deletableOrders.value.length === 0) return false
  return deletableOrders.value.every(order => selectedOrderIds.value.has(order.id))
})

// Methods
const getStatusClass = (status: string) => {
  switch (status) {
    case 'paid':
      return 'badge-green'
    case 'pending':
      return 'badge-yellow'
    case 'auditing':
      return 'badge-blue'
    case 'failed':
    case 'refunded':
      return 'badge-red'
    default:
      return 'badge-gray'
  }
}

const loadOrders = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      page_size: pagination.page_size
    }
    if (searchQuery.value) params.search = searchQuery.value
    if (filters.status) params.status = filters.status
    if (filters.order_type) params.order_type = filters.order_type

    const result = await adminAPI.orders.listOrders(params)
    orders.value = result.items as PaymentOrderWithUser[]
    pagination.total = result.total
    // Clear selection on reload
    selectedOrderIds.value = new Set()
  } catch (error) {
    console.error('Failed to load orders:', error)
    appStore.showError(t('admin.orders.loadFailed'))
  } finally {
    loading.value = false
  }
}

const loadStats = async () => {
  try {
    stats.value = await adminAPI.orders.getOrderStats()
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const handleSearch = useDebounceFn(() => {
  pagination.page = 1
  loadOrders()
}, 300)

const handlePageChange = (page: number) => {
  pagination.page = page
  loadOrders()
}

const handlePageSizeChange = (size: number) => {
  pagination.page_size = size
  pagination.page = 1
  loadOrders()
}

const handleViewDetail = async (order: PaymentOrderWithUser) => {
  try {
    detailOrder.value = await adminAPI.orders.getOrder(order.id)
    showDetailDialog.value = true
  } catch (error) {
    console.error('Failed to load order detail:', error)
  }
}

const handleApprove = (order: PaymentOrder) => {
  actionTarget.value = order
  actionType.value = 'approve'
  showActionDialog.value = true
}

const handleReject = (order: PaymentOrder) => {
  actionTarget.value = order
  actionType.value = 'reject'
  actionRejectReason.value = ''
  showActionDialog.value = true
}

const handleManualFulfill = (order: PaymentOrder) => {
  actionTarget.value = order
  actionType.value = 'fulfill'
  actionTradeNo.value = ''
  showActionDialog.value = true
}

const confirmAction = async () => {
  if (!actionTarget.value) return
  submitting.value = true
  try {
    switch (actionType.value) {
      case 'approve':
        await adminAPI.orders.approveOrder(actionTarget.value.id)
        appStore.showSuccess(t('admin.orders.approveSuccess'))
        break
      case 'reject':
        await adminAPI.orders.rejectOrder(actionTarget.value.id, actionRejectReason.value.trim())
        appStore.showSuccess(t('admin.orders.rejectSuccess'))
        break
      case 'fulfill':
        await adminAPI.orders.fulfillOrder(actionTarget.value.id, actionTradeNo.value.trim())
        appStore.showSuccess(t('admin.orders.fulfillSuccess'))
        break
    }
    showActionDialog.value = false
    actionTarget.value = null
    loadOrders()
    loadStats()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('common.operationFailed'))
  } finally {
    submitting.value = false
  }
}

// Selection methods
const toggleSelectRow = (id: number, event: Event) => {
  const target = event.target as HTMLInputElement
  const next = new Set(selectedOrderIds.value)
  if (target.checked) {
    next.add(id)
  } else {
    next.delete(id)
  }
  selectedOrderIds.value = next
}

const toggleSelectAllDeletable = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.checked) {
    selectedOrderIds.value = new Set(deletableOrders.value.map(o => o.id))
  } else {
    selectedOrderIds.value = new Set()
  }
}

// Delete methods
const handleDelete = (order: PaymentOrder) => {
  deleteTarget.value = order
  showDeleteDialog.value = true
}

const handleBatchDelete = () => {
  if (selectedOrders.value.length === 0) return
  deleteTarget.value = null
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  submitting.value = true
  try {
    if (deleteTarget.value) {
      // Single delete
      await adminAPI.orders.deleteOrder(deleteTarget.value.id)
      appStore.showSuccess(t('admin.orders.deleteSuccess'))
    } else {
      // Batch delete
      const ids = selectedOrders.value.map(o => o.id)
      const result = await adminAPI.orders.batchDeleteOrders(ids)
      appStore.showSuccess(t('admin.orders.batchDeleteSuccess', { count: result.deleted }))
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('common.operationFailed'))
  } finally {
    submitting.value = false
    showDeleteDialog.value = false
    deleteTarget.value = null
    // Always refresh list to remove any stale entries
    loadOrders()
    loadStats()
  }
}

onMounted(() => {
  loadOrders()
  loadStats()
})
</script>
