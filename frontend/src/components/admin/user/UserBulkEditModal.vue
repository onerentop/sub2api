<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.bulkActions.editTitle', { count: userIds.length })"
    width="normal"
    @close="$emit('close')"
  >
    <form id="bulk-edit-user-form" @submit.prevent="handleSubmit" class="space-y-5">
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
        {{ t('admin.users.bulkActions.editDescription') }}
      </p>

      <!-- Concurrency -->
      <div>
        <div class="flex items-center gap-2 mb-2">
          <input
            v-model="enabledFields.concurrency"
            type="checkbox"
            id="bulk-concurrency-enable"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label for="bulk-concurrency-enable" class="input-label mb-0">
            {{ t('admin.users.columns.concurrency') }}
          </label>
        </div>
        <input
          v-model.number="form.concurrency"
          type="number"
          min="1"
          class="input"
          :disabled="!enabledFields.concurrency"
          :class="{ 'opacity-50': !enabledFields.concurrency }"
        />
      </div>

      <!-- Allowed Groups -->
      <div>
        <div class="flex items-center gap-2 mb-2">
          <input
            v-model="enabledFields.allowed_groups"
            type="checkbox"
            id="bulk-groups-enable"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label for="bulk-groups-enable" class="input-label mb-0">
            {{ t('admin.users.allowedGroups.title') }}
          </label>
        </div>
        <GroupSelector
          v-model="form.allowed_groups"
          :groups="groups"
          :disabled="!enabledFields.allowed_groups"
          :class="{ 'opacity-50': !enabledFields.allowed_groups }"
        />
      </div>

      <!-- Balance Quota Section -->
      <div class="border-t pt-4">
        <label class="block mb-2 font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.users.balanceQuota.title') }}
        </label>
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
          {{ t('admin.users.bulkActions.quotaDescription') }}
        </p>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <div class="flex items-center gap-2 mb-2">
              <input
                v-model="enabledFields.balance_daily_quota"
                type="checkbox"
                id="bulk-daily-enable"
                class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <label for="bulk-daily-enable" class="input-label mb-0">
                {{ t('admin.users.balanceQuota.dailyQuota') }}
              </label>
            </div>
            <input
              v-model.number="form.balance_daily_quota"
              type="number"
              step="0.01"
              min="0"
              class="input"
              :disabled="!enabledFields.balance_daily_quota"
              :class="{ 'opacity-50': !enabledFields.balance_daily_quota }"
              :placeholder="t('admin.users.balanceQuota.useGroupDefault')"
            />
          </div>
          <div>
            <div class="flex items-center gap-2 mb-2">
              <input
                v-model="enabledFields.balance_weekly_quota"
                type="checkbox"
                id="bulk-weekly-enable"
                class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <label for="bulk-weekly-enable" class="input-label mb-0">
                {{ t('admin.users.balanceQuota.weeklyQuota') }}
              </label>
            </div>
            <input
              v-model.number="form.balance_weekly_quota"
              type="number"
              step="0.01"
              min="0"
              class="input"
              :disabled="!enabledFields.balance_weekly_quota"
              :class="{ 'opacity-50': !enabledFields.balance_weekly_quota }"
              :placeholder="t('admin.users.balanceQuota.useGroupDefault')"
            />
          </div>
        </div>
      </div>

      <!-- Balance Adjustment -->
      <div class="border-t pt-4">
        <div class="flex items-center gap-2 mb-2">
          <input
            v-model="enabledFields.balance_adjustment"
            type="checkbox"
            id="bulk-balance-enable"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label for="bulk-balance-enable" class="input-label mb-0">
            {{ t('admin.users.bulkActions.balanceAdjustment') }}
          </label>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-2">
          {{ t('admin.users.bulkActions.balanceAdjustmentHint') }}
        </p>
        <input
          v-model.number="form.balance_adjustment"
          type="number"
          step="0.01"
          class="input"
          :disabled="!enabledFields.balance_adjustment"
          :class="{ 'opacity-50': !enabledFields.balance_adjustment }"
          :placeholder="t('admin.users.bulkActions.balanceAdjustmentPlaceholder')"
        />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="bulk-edit-user-form"
          :disabled="submitting || !hasEnabledFields"
          class="btn btn-primary"
        >
          {{ submitting ? t('admin.users.updating') : t('admin.users.bulkActions.applyChanges') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Group } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'

const props = defineProps<{ show: boolean; userIds: number[] }>()
const emit = defineEmits(['close', 'updated'])
const { t } = useI18n()
const appStore = useAppStore()

const submitting = ref(false)
const groups = ref<Group[]>([])

// Enable flags for each field
const enabledFields = reactive({
  concurrency: false,
  allowed_groups: false,
  balance_daily_quota: false,
  balance_weekly_quota: false,
  balance_adjustment: false
})

const form = reactive({
  concurrency: 1,
  allowed_groups: [] as number[],
  balance_daily_quota: null as number | null,
  balance_weekly_quota: null as number | null,
  balance_adjustment: null as number | null
})

const hasEnabledFields = computed(() =>
  Object.values(enabledFields).some(v => v)
)

// Reset form when modal opens
watch(() => props.show, (isOpen) => {
  if (isOpen) {
    Object.assign(enabledFields, {
      concurrency: false,
      allowed_groups: false,
      balance_daily_quota: false,
      balance_weekly_quota: false,
      balance_adjustment: false
    })
    Object.assign(form, {
      concurrency: 1,
      allowed_groups: [],
      balance_daily_quota: null,
      balance_weekly_quota: null,
      balance_adjustment: null
    })
  }
})

const loadGroups = async () => {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch (e) {
    console.error('Failed to load groups:', e)
  }
}

const handleSubmit = async () => {
  if (props.userIds.length === 0) return
  if (!hasEnabledFields.value) {
    appStore.showError(t('admin.users.bulkActions.noFieldsSelected'))
    return
  }

  submitting.value = true
  try {
    const updates: Record<string, any> = {}

    if (enabledFields.concurrency) updates.concurrency = form.concurrency
    if (enabledFields.allowed_groups) updates.allowed_groups = form.allowed_groups
    if (enabledFields.balance_daily_quota) updates.balance_daily_quota = form.balance_daily_quota
    if (enabledFields.balance_weekly_quota) updates.balance_weekly_quota = form.balance_weekly_quota
    if (enabledFields.balance_adjustment) updates.balance_adjustment = form.balance_adjustment

    const result = await adminAPI.users.bulkUpdate(props.userIds, updates)

    if (result.success > 0) {
      appStore.showSuccess(t('admin.users.bulkActions.updateSuccess', { count: result.success }))
    }
    if (result.failed > 0) {
      appStore.showError(t('admin.users.bulkActions.updateFailed', { count: result.failed }))
    }

    emit('updated')
  } catch (e: any) {
    appStore.showError(e.response?.data?.detail || t('admin.users.bulkActions.updateError'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadGroups()
})
</script>
