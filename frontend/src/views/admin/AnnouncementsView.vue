<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex justify-end gap-3">
          <button
            @click="loadAnnouncements"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button @click="showCreateDialog = true" class="btn btn-primary">
            <Icon name="plus" size="md" class="mr-1" />
            {{ t('admin.announcements.create') }}
          </button>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex gap-2">
            <Select
              v-model="filters.enabled"
              :options="enabledFilterOptions"
              class="w-36"
              @change="loadAnnouncements"
            />
            <Select
              v-model="filters.type"
              :options="typeFilterOptions"
              class="w-36"
              @change="loadAnnouncements"
            />
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="announcements" :loading="loading">
          <template #cell-content="{ value, row }">
            <div class="max-w-md">
              <div v-if="row.title" class="font-medium text-gray-900 dark:text-white">
                {{ row.title }}
              </div>
              <div
                class="line-clamp-2 text-sm text-gray-600 dark:text-gray-300"
                v-html="truncateHtml(value, 100)"
              ></div>
            </div>
          </template>

          <template #cell-type="{ value }">
            <span :class="['badge', getTypeBadgeClass(value)]">
              {{ t(`admin.announcements.types.${value}`) }}
            </span>
          </template>

          <template #cell-enabled="{ value }">
            <span :class="['badge', value ? 'badge-success' : 'badge-secondary']">
              {{ value ? t('common.enabled') : t('common.disabled') }}
            </span>
          </template>

          <template #cell-schedule="{ row }">
            <div class="text-sm text-gray-500 dark:text-dark-400">
              <div v-if="row.start_time" class="flex items-center gap-1">
                <Icon name="calendar" size="xs" />
                {{ formatDateTime(row.start_time) }}
              </div>
              <div v-if="row.end_time" class="flex items-center gap-1 text-orange-500">
                <Icon name="clock" size="xs" />
                {{ formatDateTime(row.end_time) }}
              </div>
              <span v-if="!row.start_time && !row.end_time">-</span>
            </div>
          </template>

          <template #cell-sort_order="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ value }}</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
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

    <!-- Create Dialog -->
    <BaseDialog
      :show="showCreateDialog"
      :title="t('admin.announcements.create')"
      width="wide"
      @close="showCreateDialog = false"
    >
      <form id="create-announcement-form" @submit.prevent="handleCreate" class="space-y-4">
        <div>
          <label class="input-label">
            {{ t('admin.announcements.title') }}
            <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
          </label>
          <input
            v-model="createForm.title"
            type="text"
            class="input"
            :placeholder="t('admin.announcements.titlePlaceholder')"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.announcements.content') }}</label>
          <textarea
            v-model="createForm.content"
            rows="4"
            required
            class="input"
            :placeholder="t('admin.announcements.contentPlaceholder')"
          ></textarea>
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.announcements.htmlSupported') }}</p>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.announcements.type') }}</label>
            <Select v-model="createForm.type" :options="typeOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.status') }}</label>
            <div class="flex items-center h-10">
              <label class="relative inline-flex cursor-pointer items-center">
                <input v-model="createForm.enabled" type="checkbox" class="peer sr-only" />
                <div
                  class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-500 peer-checked:after:translate-x-full peer-checked:after:border-white dark:border-dark-600 dark:bg-dark-600"
                ></div>
                <span class="ml-2 text-sm text-gray-700 dark:text-gray-300">
                  {{ createForm.enabled ? t('common.enabled') : t('common.disabled') }}
                </span>
              </label>
            </div>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">
              {{ t('admin.announcements.startTime') }}
              <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input v-model="createForm.start_time_str" type="datetime-local" class="input" />
          </div>
          <div>
            <label class="input-label">
              {{ t('admin.announcements.endTime') }}
              <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input v-model="createForm.end_time_str" type="datetime-local" class="input" />
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" @click="showCreateDialog = false" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="create-announcement-form" :disabled="creating" class="btn btn-primary">
            {{ creating ? t('common.creating') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog
      :show="showEditDialog"
      :title="t('admin.announcements.edit')"
      width="wide"
      @close="closeEditDialog"
    >
      <form id="edit-announcement-form" @submit.prevent="handleUpdate" class="space-y-4">
        <div>
          <label class="input-label">
            {{ t('admin.announcements.title') }}
            <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
          </label>
          <input v-model="editForm.title" type="text" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.announcements.content') }}</label>
          <textarea v-model="editForm.content" rows="4" required class="input"></textarea>
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.announcements.htmlSupported') }}</p>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.announcements.type') }}</label>
            <Select v-model="editForm.type" :options="typeOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.status') }}</label>
            <div class="flex items-center h-10">
              <label class="relative inline-flex cursor-pointer items-center">
                <input v-model="editForm.enabled" type="checkbox" class="peer sr-only" />
                <div
                  class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-500 peer-checked:after:translate-x-full peer-checked:after:border-white dark:border-dark-600 dark:bg-dark-600"
                ></div>
                <span class="ml-2 text-sm text-gray-700 dark:text-gray-300">
                  {{ editForm.enabled ? t('common.enabled') : t('common.disabled') }}
                </span>
              </label>
            </div>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">
              {{ t('admin.announcements.startTime') }}
              <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <div class="flex gap-2">
              <input v-model="editForm.start_time_str" type="datetime-local" class="input flex-1" />
              <button
                v-if="editForm.start_time_str"
                type="button"
                @click="editForm.start_time_str = ''; editForm.clear_start_time = true"
                class="btn btn-secondary"
                :title="t('common.clear')"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>
          <div>
            <label class="input-label">
              {{ t('admin.announcements.endTime') }}
              <span class="ml-1 text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <div class="flex gap-2">
              <input v-model="editForm.end_time_str" type="datetime-local" class="input flex-1" />
              <button
                v-if="editForm.end_time_str"
                type="button"
                @click="editForm.end_time_str = ''; editForm.clear_end_time = true"
                class="btn btn-secondary"
                :title="t('common.clear')"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.announcements.sortOrder') }}</label>
          <input v-model.number="editForm.sort_order" type="number" min="0" class="input w-32" />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" @click="closeEditDialog" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="edit-announcement-form" :disabled="updating" class="btn btn-primary">
            {{ updating ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirm Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.announcements.deleteConfirmTitle')"
      :message="t('admin.announcements.deleteConfirmMessage')"
      :confirm-text="t('common.delete')"
      confirm-variant="danger"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { Announcement, AnnouncementType } from '@/types'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

// State
const announcements = ref<Announcement[]>([])
const loading = ref(false)
const creating = ref(false)
const updating = ref(false)
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const selectedAnnouncement = ref<Announcement | null>(null)

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const filters = reactive({
  enabled: '',
  type: ''
})

// Forms
const createForm = reactive({
  title: '',
  content: '',
  type: 'info' as AnnouncementType,
  enabled: true,
  start_time_str: '',
  end_time_str: ''
})

const editForm = reactive({
  id: 0,
  title: '',
  content: '',
  type: 'info' as AnnouncementType,
  enabled: true,
  sort_order: 0,
  start_time_str: '',
  end_time_str: '',
  clear_start_time: false,
  clear_end_time: false
})

// Options
const typeOptions = computed(() => [
  { value: 'info', label: t('admin.announcements.types.info') },
  { value: 'success', label: t('admin.announcements.types.success') },
  { value: 'warning', label: t('admin.announcements.types.warning') },
  { value: 'error', label: t('admin.announcements.types.error') }
])

const typeFilterOptions = computed(() => [
  { value: '', label: t('admin.announcements.allTypes') },
  ...typeOptions.value
])

const enabledFilterOptions = computed(() => [
  { value: '', label: t('admin.announcements.allStatus') },
  { value: 'true', label: t('common.enabled') },
  { value: 'false', label: t('common.disabled') }
])

// Table columns
const columns = computed(() => [
  { key: 'content', label: t('admin.announcements.content'), sortable: false },
  { key: 'type', label: t('admin.announcements.type'), width: '100px' },
  { key: 'enabled', label: t('admin.announcements.status'), width: '100px' },
  { key: 'schedule', label: t('admin.announcements.schedule'), width: '180px' },
  { key: 'sort_order', label: t('admin.announcements.sortOrder'), width: '80px' },
  { key: 'created_at', label: t('common.createdAt'), width: '160px' },
  { key: 'actions', label: t('common.actions'), width: '100px' }
])

// Functions
function getTypeBadgeClass(type: AnnouncementType): string {
  const classes: Record<AnnouncementType, string> = {
    info: 'badge-info',
    success: 'badge-success',
    warning: 'badge-warning',
    error: 'badge-danger'
  }
  return classes[type] || 'badge-secondary'
}

function truncateHtml(html: string, maxLength: number): string {
  // Simple truncation - for display purposes
  const text = html.replace(/<[^>]*>/g, '')
  if (text.length <= maxLength) return html
  return text.substring(0, maxLength) + '...'
}

function datetimeLocalToTimestamp(str: string): number | null {
  if (!str) return null
  return Math.floor(new Date(str).getTime() / 1000)
}

function timestampToDatetimeLocal(timestamp: string | null): string {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const offset = date.getTimezoneOffset() * 60000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

async function loadAnnouncements() {
  loading.value = true
  try {
    const response = await adminAPI.announcements.list(
      pagination.page,
      pagination.page_size,
      {
        enabled: filters.enabled ? filters.enabled === 'true' : undefined,
        type: filters.type || undefined
      }
    )
    announcements.value = response.items || []
    pagination.total = response.total || 0
  } catch (error: any) {
    appStore.showError(error.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadAnnouncements()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  loadAnnouncements()
}

function resetCreateForm() {
  createForm.title = ''
  createForm.content = ''
  createForm.type = 'info'
  createForm.enabled = true
  createForm.start_time_str = ''
  createForm.end_time_str = ''
}

async function handleCreate() {
  if (!createForm.content.trim()) {
    appStore.showError(t('admin.announcements.contentRequired'))
    return
  }

  creating.value = true
  try {
    await adminAPI.announcements.create({
      title: createForm.title || undefined,
      content: createForm.content,
      type: createForm.type,
      enabled: createForm.enabled,
      start_time: datetimeLocalToTimestamp(createForm.start_time_str),
      end_time: datetimeLocalToTimestamp(createForm.end_time_str)
    })
    appStore.showSuccess(t('admin.announcements.createSuccess'))
    showCreateDialog.value = false
    resetCreateForm()
    loadAnnouncements()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.announcements.createFailed'))
  } finally {
    creating.value = false
  }
}

function handleEdit(announcement: Announcement) {
  selectedAnnouncement.value = announcement
  editForm.id = announcement.id
  editForm.title = announcement.title || ''
  editForm.content = announcement.content
  editForm.type = announcement.type
  editForm.enabled = announcement.enabled
  editForm.sort_order = announcement.sort_order
  editForm.start_time_str = timestampToDatetimeLocal(announcement.start_time)
  editForm.end_time_str = timestampToDatetimeLocal(announcement.end_time)
  editForm.clear_start_time = false
  editForm.clear_end_time = false
  showEditDialog.value = true
}

function closeEditDialog() {
  showEditDialog.value = false
  selectedAnnouncement.value = null
}

async function handleUpdate() {
  if (!editForm.content.trim()) {
    appStore.showError(t('admin.announcements.contentRequired'))
    return
  }

  updating.value = true
  try {
    await adminAPI.announcements.update(editForm.id, {
      title: editForm.title || null,
      content: editForm.content,
      type: editForm.type,
      enabled: editForm.enabled,
      sort_order: editForm.sort_order,
      start_time: datetimeLocalToTimestamp(editForm.start_time_str),
      end_time: datetimeLocalToTimestamp(editForm.end_time_str),
      clear_start_time: editForm.clear_start_time,
      clear_end_time: editForm.clear_end_time
    })
    appStore.showSuccess(t('admin.announcements.updateSuccess'))
    closeEditDialog()
    loadAnnouncements()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.announcements.updateFailed'))
  } finally {
    updating.value = false
  }
}

function handleDelete(announcement: Announcement) {
  selectedAnnouncement.value = announcement
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!selectedAnnouncement.value) return

  try {
    await adminAPI.announcements.delete(selectedAnnouncement.value.id)
    appStore.showSuccess(t('admin.announcements.deleteSuccess'))
    showDeleteDialog.value = false
    selectedAnnouncement.value = null
    loadAnnouncements()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.announcements.deleteFailed'))
  }
}

onMounted(() => {
  loadAnnouncements()
})
</script>
