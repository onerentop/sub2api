<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex justify-end gap-3">
          <button
            @click="loadProducts"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button @click="showCreateDialog = true" class="btn btn-primary">
            <Icon name="plus" size="md" class="mr-1" />
            {{ t('admin.products.createProduct') }}
          </button>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="max-w-md flex-1">
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('admin.products.searchProducts')"
              class="input"
              @input="handleSearch"
            />
          </div>
          <div class="flex gap-2">
            <Select
              v-model="filters.type"
              :options="typeOptions"
              class="w-36"
              @change="loadProducts"
            />
            <Select
              v-model="filters.is_active"
              :options="statusOptions"
              class="w-32"
              @change="loadProducts"
            />
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="products" :loading="loading">
          <template #cell-name="{ value, row }">
            <div>
              <p class="font-medium text-gray-900 dark:text-white">{{ value }}</p>
              <p v-if="row.description" class="text-xs text-gray-500 dark:text-dark-400">
                {{ row.description }}
              </p>
            </div>
          </template>

          <template #cell-type="{ value }">
            <span
              :class="[
                'badge',
                value === 'balance' ? 'badge-blue' : 'badge-purple'
              ]"
            >
              {{ t(`admin.products.type.${value}`) }}
            </span>
          </template>

          <template #cell-price_cny="{ value }">
            <span class="font-semibold text-gray-900 dark:text-white">
              ¥{{ value.toFixed(2) }}
            </span>
          </template>

          <template #cell-value="{ value, row }">
            <span class="text-sm text-gray-600 dark:text-gray-300">
              <template v-if="row.type === 'balance'">
                ${{ value.toFixed(2) }}
              </template>
              <template v-else>
                {{ value }} {{ t('admin.products.days') }}
              </template>
            </span>
          </template>

          <template #cell-group="{ row }">
            <span v-if="row.group" class="badge badge-gray">
              {{ row.group.name }}
            </span>
            <span v-else class="text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-is_active="{ value }">
            <span
              :class="[
                'badge',
                value ? 'badge-green' : 'badge-gray'
              ]"
            >
              {{ value ? t('common.enabled') : t('common.disabled') }}
            </span>
          </template>

          <template #cell-sort_order="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ value }}
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
                @click="handleToggleActive(row)"
                :class="[
                  'flex flex-col items-center gap-0.5 rounded-lg p-1.5 transition-colors',
                  row.is_active
                    ? 'text-gray-500 hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-400'
                    : 'text-gray-500 hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400'
                ]"
                :title="row.is_active ? t('common.disable') : t('common.enable')"
              >
                <Icon :name="row.is_active ? 'eyeSlash' : 'eye'" size="sm" />
              </button>
              <button
                @click="handleEdit(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
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

    <!-- Create/Edit Dialog -->
    <BaseDialog
      :show="showCreateDialog || showEditDialog"
      :title="showEditDialog ? t('admin.products.editProduct') : t('admin.products.createProduct')"
      width="normal"
      @close="closeDialogs"
    >
      <form id="product-form" @submit.prevent="handleSubmit" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.products.name') }}</label>
          <input
            v-model="form.name"
            type="text"
            required
            class="input"
            :placeholder="t('admin.products.namePlaceholder')"
          />
        </div>

        <div>
          <label class="input-label">
            {{ t('admin.products.description') }}
            <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
          </label>
          <textarea
            v-model="form.description"
            class="input"
            rows="2"
            :placeholder="t('admin.products.descriptionPlaceholder')"
          ></textarea>
        </div>

        <div>
          <label class="input-label">{{ t('admin.products.productType') }}</label>
          <Select
            v-model="form.type"
            :options="productTypeOptions"
            :disabled="showEditDialog"
          />
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.products.priceCny') }}</label>
            <div class="relative">
              <span class="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500">¥</span>
              <input
                v-model.number="form.price_cny"
                type="number"
                step="0.01"
                min="0"
                required
                class="input pl-8"
              />
            </div>
          </div>
          <div>
            <label class="input-label">
              {{ form.type === 'balance' ? t('admin.products.balanceValue') : t('admin.products.subscriptionDays') }}
            </label>
            <div class="relative">
              <span v-if="form.type === 'balance'" class="absolute inset-y-0 left-0 flex items-center pl-3 text-gray-500">$</span>
              <input
                v-model.number="form.value"
                type="number"
                :step="form.type === 'balance' ? '0.01' : '1'"
                min="0"
                required
                :class="['input', form.type === 'balance' ? 'pl-8' : '']"
              />
            </div>
          </div>
        </div>

        <div v-if="form.type === 'subscription'">
          <label class="input-label">
            {{ t('admin.products.group') }}
            <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
          </label>
          <Select
            v-model="form.group_id"
            :options="groupOptions"
            :placeholder="t('admin.products.selectGroup')"
          />
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.products.sortOrder') }}</label>
            <input
              v-model.number="form.sort_order"
              type="number"
              min="0"
              class="input"
            />
          </div>
          <div class="flex items-end">
            <label class="flex items-center gap-2">
              <input
                v-model="form.is_active"
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">
                {{ t('admin.products.enableProduct') }}
              </span>
            </label>
          </div>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeDialogs">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="product-form"
            class="btn btn-primary"
            :disabled="submitting"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
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
            {{ showEditDialog ? t('common.save') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <BaseDialog
      :show="showDeleteDialog"
      :title="t('admin.products.deleteProduct')"
      width="narrow"
      @close="showDeleteDialog = false"
    >
      <p class="text-gray-600 dark:text-gray-400">
        {{ t('admin.products.deleteConfirm', { name: deleteTarget?.name }) }}
      </p>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="showDeleteDialog = false">
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="btn btn-danger"
            :disabled="submitting"
            @click="confirmDelete"
          >
            {{ t('common.delete') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api'
import type { Product } from '@/api/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import { useDebounceFn } from '@vueuse/core'

const { t } = useI18n()
const appStore = useAppStore()

// State
const products = ref<Product[]>([])
const groups = ref<{ id: number; name: string }[]>([])
const loading = ref(false)
const submitting = ref(false)
const searchQuery = ref('')

const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const editTarget = ref<Product | null>(null)
const deleteTarget = ref<Product | null>(null)

// Filters
const filters = reactive({
  type: '',
  is_active: ''
})

// Pagination
const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

// Form
const defaultForm = {
  name: '',
  description: '',
  type: 'balance' as 'balance' | 'subscription',
  price_cny: 0,
  value: 0,
  group_id: null as number | null,
  is_active: true,
  sort_order: 0
}

const form = reactive({ ...defaultForm })

// Table columns
const columns = computed(() => [
  { key: 'name', label: t('admin.products.name'), sortable: true },
  { key: 'type', label: t('admin.products.type.label'), sortable: true },
  { key: 'price_cny', label: t('admin.products.price'), sortable: true },
  { key: 'value', label: t('admin.products.value'), sortable: true },
  { key: 'group', label: t('admin.products.group') },
  { key: 'is_active', label: t('common.status'), sortable: true },
  { key: 'sort_order', label: t('admin.products.sortOrder'), sortable: true },
  { key: 'created_at', label: t('common.createdAt'), sortable: true },
  { key: 'actions', label: t('common.actions'), align: 'right' as const }
])

// Filter options
const typeOptions = computed(() => [
  { value: '', label: t('admin.products.allTypes') },
  { value: 'balance', label: t('admin.products.type.balance') },
  { value: 'subscription', label: t('admin.products.type.subscription') }
])

const statusOptions = computed(() => [
  { value: '', label: t('common.allStatus') },
  { value: 'true', label: t('common.enabled') },
  { value: 'false', label: t('common.disabled') }
])

const productTypeOptions = computed(() => [
  { value: 'balance', label: t('admin.products.type.balance') },
  { value: 'subscription', label: t('admin.products.type.subscription') }
])

const groupOptions = computed(() => [
  { value: null, label: t('admin.products.noGroup') },
  ...groups.value.map(g => ({ value: g.id, label: g.name }))
])

// Methods
const loadProducts = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      page_size: pagination.page_size
    }
    if (searchQuery.value) params.search = searchQuery.value
    if (filters.type) params.type = filters.type
    if (filters.is_active !== '') params.is_active = filters.is_active === 'true'

    const result = await adminAPI.products.listProducts(params)
    products.value = result.items
    pagination.total = result.total
  } catch (error) {
    console.error('Failed to load products:', error)
    appStore.showError(t('admin.products.loadFailed'))
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  try {
    const result = await adminAPI.groups.list(1, 100)
    groups.value = result.items.map((g: any) => ({ id: g.id, name: g.name }))
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const handleSearch = useDebounceFn(() => {
  pagination.page = 1
  loadProducts()
}, 300)

const handlePageChange = (page: number) => {
  pagination.page = page
  loadProducts()
}

const handlePageSizeChange = (size: number) => {
  pagination.page_size = size
  pagination.page = 1
  loadProducts()
}

const resetForm = () => {
  Object.assign(form, defaultForm)
}

const closeDialogs = () => {
  showCreateDialog.value = false
  showEditDialog.value = false
  editTarget.value = null
  resetForm()
}

const handleEdit = (product: Product) => {
  editTarget.value = product
  form.name = product.name
  form.description = product.description || ''
  form.type = product.type
  form.price_cny = product.price_cny
  form.value = product.value
  form.group_id = product.group_id || null
  form.is_active = product.is_active
  form.sort_order = product.sort_order
  showEditDialog.value = true
}

const handleDelete = (product: Product) => {
  deleteTarget.value = product
  showDeleteDialog.value = true
}

const handleToggleActive = async (product: Product) => {
  try {
    await adminAPI.products.toggleProductActive(product.id)
    appStore.showSuccess(
      product.is_active
        ? t('admin.products.disabled')
        : t('admin.products.enabled')
    )
    loadProducts()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('common.operationFailed'))
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    if (showEditDialog.value && editTarget.value) {
      await adminAPI.products.updateProduct(editTarget.value.id, {
        name: form.name,
        description: form.description || undefined,
        price_cny: form.price_cny,
        value: form.value,
        group_id: form.type === 'subscription' ? form.group_id : null,
        is_active: form.is_active,
        sort_order: form.sort_order
      })
      appStore.showSuccess(t('admin.products.updateSuccess'))
    } else {
      await adminAPI.products.createProduct({
        name: form.name,
        description: form.description || undefined,
        type: form.type,
        price_cny: form.price_cny,
        value: form.value,
        group_id: form.type === 'subscription' ? form.group_id || undefined : undefined,
        is_active: form.is_active,
        sort_order: form.sort_order
      })
      appStore.showSuccess(t('admin.products.createSuccess'))
    }
    closeDialogs()
    loadProducts()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('common.operationFailed'))
  } finally {
    submitting.value = false
  }
}

const confirmDelete = async () => {
  if (!deleteTarget.value) return
  submitting.value = true
  try {
    await adminAPI.products.deleteProduct(deleteTarget.value.id)
    appStore.showSuccess(t('admin.products.deleteSuccess'))
    showDeleteDialog.value = false
    deleteTarget.value = null
    loadProducts()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('common.operationFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadProducts()
  loadGroups()
})
</script>
