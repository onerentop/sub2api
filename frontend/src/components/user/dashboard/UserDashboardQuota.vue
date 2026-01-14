<template>
  <div v-if="quotaInfo && hasQuota" class="card p-4">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('dashboard.quota.title') }}
      </h3>
      <span
        v-if="quotaInfo.source"
        class="text-xs px-2 py-0.5 rounded-full"
        :class="quotaInfo.source === 'user' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'"
      >
        {{ quotaInfo.source === 'user' ? t('dashboard.quota.userOverride') : t('dashboard.quota.groupDefault') }}
      </span>
    </div>

    <div class="space-y-4">
      <!-- Daily Quota -->
      <div v-if="quotaInfo.daily.limit !== null">
        <div class="flex items-center justify-between mb-1">
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.quota.daily') }}</span>
          <span class="text-xs text-gray-600 dark:text-gray-300">
            ${{ formatCost(quotaInfo.daily.used) }} / ${{ formatCost(quotaInfo.daily.limit) }}
          </span>
        </div>
        <div class="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
          <div
            class="h-2 rounded-full transition-all duration-300"
            :class="getProgressClass(getDailyUsedPercentage)"
            :style="{ width: `${getDailyRemainingPercentage}%` }"
          ></div>
        </div>
        <div class="flex items-center justify-between mt-1">
          <span class="text-xs" :class="quotaInfo.daily.remaining !== null && quotaInfo.daily.remaining <= 0 ? 'text-red-500' : 'text-gray-500 dark:text-gray-400'">
            {{ quotaInfo.daily.remaining !== null ? `${t('dashboard.quota.remaining')}: $${formatCost(quotaInfo.daily.remaining)}` : '' }}
          </span>
          <span class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('dashboard.quota.resetsAt') }}: {{ formatResetTime(quotaInfo.daily.reset_at) }}
          </span>
        </div>
      </div>

      <!-- Weekly Quota -->
      <div v-if="quotaInfo.weekly.limit !== null">
        <div class="flex items-center justify-between mb-1">
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.quota.weekly') }}</span>
          <span class="text-xs text-gray-600 dark:text-gray-300">
            ${{ formatCost(quotaInfo.weekly.used) }} / ${{ formatCost(quotaInfo.weekly.limit) }}
          </span>
        </div>
        <div class="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
          <div
            class="h-2 rounded-full transition-all duration-300"
            :class="getProgressClass(getWeeklyUsedPercentage)"
            :style="{ width: `${getWeeklyRemainingPercentage}%` }"
          ></div>
        </div>
        <div class="flex items-center justify-between mt-1">
          <span class="text-xs" :class="quotaInfo.weekly.remaining !== null && quotaInfo.weekly.remaining <= 0 ? 'text-red-500' : 'text-gray-500 dark:text-gray-400'">
            {{ quotaInfo.weekly.remaining !== null ? `${t('dashboard.quota.remaining')}: $${formatCost(quotaInfo.weekly.remaining)}` : '' }}
          </span>
          <span class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('dashboard.quota.resetsAt') }}: {{ formatResetTime(quotaInfo.weekly.reset_at) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BalanceQuotaInfo } from '@/api/usage'

const props = defineProps<{
  quotaInfo: BalanceQuotaInfo | null
}>()

const { t } = useI18n()

const hasQuota = computed(() => {
  if (!props.quotaInfo) return false
  return props.quotaInfo.daily.limit !== null || props.quotaInfo.weekly.limit !== null
})

// 计算已使用百分比（用于颜色判断）
const getDailyUsedPercentage = computed(() => {
  if (!props.quotaInfo || props.quotaInfo.daily.limit === null || props.quotaInfo.daily.limit === 0) return 0
  return (props.quotaInfo.daily.used / props.quotaInfo.daily.limit) * 100
})

const getWeeklyUsedPercentage = computed(() => {
  if (!props.quotaInfo || props.quotaInfo.weekly.limit === null || props.quotaInfo.weekly.limit === 0) return 0
  return (props.quotaInfo.weekly.used / props.quotaInfo.weekly.limit) * 100
})

// 计算剩余百分比（用于进度条宽度）
const getDailyRemainingPercentage = computed(() => {
  return Math.max(0, 100 - getDailyUsedPercentage.value)
})

const getWeeklyRemainingPercentage = computed(() => {
  return Math.max(0, 100 - getWeeklyUsedPercentage.value)
})

// 根据已使用百分比决定颜色（使用越多颜色越危险）
const getProgressClass = (usedPercentage: number) => {
  if (usedPercentage >= 100) return 'bg-red-500'
  if (usedPercentage >= 80) return 'bg-orange-500'
  if (usedPercentage >= 60) return 'bg-yellow-500'
  return 'bg-green-500'
}

const formatCost = (c: number | undefined | null) => (c ?? 0).toFixed(2)

const formatResetTime = (isoString: string) => {
  if (!isoString) return ''
  const date = new Date(isoString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()

  if (diffMs <= 0) return t('dashboard.quota.resettingSoon')

  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffMins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))

  if (diffHours >= 24) {
    const days = Math.floor(diffHours / 24)
    const hours = diffHours % 24
    return `${days}${t('dashboard.quota.days')} ${hours}${t('dashboard.quota.hours')}`
  }
  if (diffHours > 0) {
    return `${diffHours}${t('dashboard.quota.hours')} ${diffMins}${t('dashboard.quota.minutes')}`
  }
  return `${diffMins}${t('dashboard.quota.minutes')}`
}
</script>
