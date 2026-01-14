<template>
  <div :class="size === 'sm' ? 'space-y-0.5' : 'space-y-1'">
    <div class="flex items-center justify-between">
      <span :class="[
        'text-gray-500 dark:text-gray-400',
        size === 'sm' ? 'text-[10px]' : 'text-xs'
      ]">{{ label }}</span>
      <span :class="[
        'text-gray-600 dark:text-gray-300',
        size === 'sm' ? 'text-[10px]' : 'text-xs'
      ]">
        ${{ formatCost(used) }} / ${{ formatCost(limit) }}
      </span>
    </div>
    <div :class="[
      'w-full bg-gray-200 rounded-full dark:bg-gray-700',
      size === 'sm' ? 'h-1.5' : 'h-2'
    ]">
      <div
        :class="[
          'rounded-full transition-all duration-300',
          size === 'sm' ? 'h-1.5' : 'h-2',
          progressClass
        ]"
        :style="{ width: `${remainingPercentage}%` }"
      ></div>
    </div>
    <div class="flex items-center justify-between">
      <span :class="[
        remaining !== null && remaining <= 0 ? 'text-red-500' : 'text-gray-500 dark:text-gray-400',
        size === 'sm' ? 'text-[10px]' : 'text-xs'
      ]">
        {{ remaining !== null ? `${t('dashboard.quota.remaining')}: $${formatCost(remaining)}` : '' }}
      </span>
      <span :class="[
        'text-gray-400 dark:text-gray-500',
        size === 'sm' ? 'text-[10px]' : 'text-xs'
      ]">
        {{ t('dashboard.quota.resetsAt') }}: {{ formatResetTime(resetAt) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  label: string
  used: number
  limit: number | null
  remaining: number | null
  resetAt: string
  size?: 'default' | 'sm'
}>(), {
  size: 'default'
})

const { t } = useI18n()

const usedPercentage = computed(() => {
  if (props.limit === null || props.limit === 0) return 0
  return (props.used / props.limit) * 100
})

const remainingPercentage = computed(() => {
  return Math.max(0, 100 - usedPercentage.value)
})

const progressClass = computed(() => {
  const used = usedPercentage.value
  if (used >= 100) return 'bg-red-500'
  if (used >= 80) return 'bg-orange-500'
  if (used >= 60) return 'bg-yellow-500'
  return 'bg-green-500'
})

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
