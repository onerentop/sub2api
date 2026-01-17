<template>
  <div v-if="providers.length > 0" class="space-y-4">
    <!-- 分隔线 -->
    <div v-if="showDivider" class="flex items-center gap-3">
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
      <span class="text-xs text-gray-500 dark:text-dark-400">
        {{ t('auth.social.orContinueWith') }}
      </span>
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
    </div>

    <!-- 社交登录按钮 -->
    <div class="space-y-2">
      <SocialLoginButton
        v-for="provider in providers"
        :key="provider.name"
        :provider="provider.name"
        :display-name="provider.display_name"
        :disabled="disabled"
        @error="handleError"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getSocialProviders, type SocialProvider } from '@/api/social'
import SocialLoginButton from './SocialLoginButton.vue'

defineProps<{
  disabled?: boolean
  showDivider?: boolean
}>()

const emit = defineEmits<{
  (e: 'error', error: Error): void
}>()

const { t } = useI18n()
const providers = ref<SocialProvider[]>([])
const loading = ref(false)

onMounted(async () => {
  try {
    loading.value = true
    providers.value = await getSocialProviders()
  } catch (error) {
    console.error('Failed to load social providers:', error)
  } finally {
    loading.value = false
  }
})

function handleError(error: Error) {
  emit('error', error)
}
</script>
