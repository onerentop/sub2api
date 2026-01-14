<template>
  <div v-if="quotaInfo && hasQuota" class="card p-4">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('dashboard.quota.title') }}
      </h3>
      <div class="flex items-center gap-2">
        <span
          v-if="quotaInfo.source"
          class="text-xs px-2 py-0.5 rounded-full"
          :class="getSourceBadgeClass(quotaInfo.source)"
        >
          {{ getSourceLabel(quotaInfo.source) }}
        </span>
        <button
          v-if="hasGroups"
          @click="toggleExpanded"
          class="text-xs text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1"
        >
          <span>{{ expanded ? t('dashboard.quota.collapse') : t('dashboard.quota.expandGroups') }}</span>
          <svg class="w-3.5 h-3.5 transition-transform" :class="{ 'rotate-180': expanded }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Global Quota Section -->
    <div class="space-y-4">
      <div v-if="hasGlobalQuota" class="border-b border-gray-200 dark:border-gray-700 pb-4">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-400 mb-3">
          {{ t('dashboard.quota.globalLimit') }}
        </div>

        <!-- Global Daily Quota -->
        <QuotaBar
          v-if="quotaInfo.global.daily.limit !== null"
          :label="t('dashboard.quota.daily')"
          :used="quotaInfo.global.daily.used"
          :limit="quotaInfo.global.daily.limit"
          :remaining="quotaInfo.global.daily.remaining"
          :reset-at="quotaInfo.global.daily.reset_at"
        />

        <!-- Global Weekly Quota -->
        <QuotaBar
          v-if="quotaInfo.global.weekly.limit !== null"
          :label="t('dashboard.quota.weekly')"
          :used="quotaInfo.global.weekly.used"
          :limit="quotaInfo.global.weekly.limit"
          :remaining="quotaInfo.global.weekly.remaining"
          :reset-at="quotaInfo.global.weekly.reset_at"
          class="mt-3"
        />
      </div>

      <!-- Per-Group Quotas (Collapsible) -->
      <div v-if="hasGroups && expanded" class="space-y-4">
        <div
          v-for="group in quotaInfo.groups"
          :key="group.group_id"
          class="border border-gray-100 dark:border-gray-700 rounded-lg p-3"
        >
          <div class="text-xs font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ group.group_name }}
          </div>

          <!-- Group Daily Quota -->
          <QuotaBar
            v-if="group.daily.limit !== null"
            :label="t('dashboard.quota.daily')"
            :used="group.daily.used"
            :limit="group.daily.limit"
            :remaining="group.daily.remaining"
            :reset-at="group.daily.reset_at"
            size="sm"
          />

          <!-- Group Weekly Quota -->
          <QuotaBar
            v-if="group.weekly.limit !== null"
            :label="t('dashboard.quota.weekly')"
            :used="group.weekly.used"
            :limit="group.weekly.limit"
            :remaining="group.weekly.remaining"
            :reset-at="group.weekly.reset_at"
            size="sm"
            class="mt-2"
          />

          <!-- No limit for this group -->
          <div
            v-if="group.daily.limit === null && group.weekly.limit === null"
            class="text-xs text-gray-400 dark:text-gray-500"
          >
            {{ t('dashboard.quota.noLimit') }}
          </div>
        </div>
      </div>

      <!-- Groups Summary (When collapsed) -->
      <div v-if="hasGroups && !expanded && !hasGlobalQuota" class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('dashboard.quota.groupsCount', { count: quotaInfo.total_groups }) }}
      </div>

      <!-- Backward compatible display (when no global quota but has groups) -->
      <template v-if="!hasGlobalQuota && !hasGroups">
        <!-- Daily Quota -->
        <QuotaBar
          v-if="quotaInfo.daily.limit !== null"
          :label="t('dashboard.quota.daily')"
          :used="quotaInfo.daily.used"
          :limit="quotaInfo.daily.limit"
          :remaining="quotaInfo.daily.remaining"
          :reset-at="quotaInfo.daily.reset_at"
        />

        <!-- Weekly Quota -->
        <QuotaBar
          v-if="quotaInfo.weekly.limit !== null"
          :label="t('dashboard.quota.weekly')"
          :used="quotaInfo.weekly.used"
          :limit="quotaInfo.weekly.limit"
          :remaining="quotaInfo.weekly.remaining"
          :reset-at="quotaInfo.weekly.reset_at"
          class="mt-3"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BalanceQuotaInfoV2 } from '@/api/usage'
import QuotaBar from './QuotaBar.vue'

const props = defineProps<{
  quotaInfo: BalanceQuotaInfoV2 | null
}>()

const { t } = useI18n()
const expanded = ref(false)

const hasQuota = computed(() => {
  const q = props.quotaInfo
  if (!q) return false
  // Check global quota
  if (q.global.daily.limit != null || q.global.weekly.limit != null) return true
  // Check if any group has quota
  if (q.groups?.some(g => g.daily.limit != null || g.weekly.limit != null)) return true
  // Check backward compatible fields
  return q.daily.limit != null || q.weekly.limit != null
})

const hasGlobalQuota = computed(() =>
  props.quotaInfo?.global.daily.limit != null || props.quotaInfo?.global.weekly.limit != null
)

const hasGroups = computed(() =>
  props.quotaInfo?.groups?.some(g => g.daily.limit != null || g.weekly.limit != null) ?? false
)

const toggleExpanded = () => {
  expanded.value = !expanded.value
}

const getSourceBadgeClass = (source: string) => {
  switch (source) {
    case 'user':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    case 'global':
      return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
    case 'group':
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
  }
}

const getSourceLabel = (source: string) => {
  switch (source) {
    case 'user':
      return t('dashboard.quota.userOverride')
    case 'global':
      return t('dashboard.quota.globalLimit')
    case 'group':
      return t('dashboard.quota.groupDefault')
    default:
      return ''
  }
}
</script>
