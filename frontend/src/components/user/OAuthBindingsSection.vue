<template>
  <div class="space-y-4">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
      {{ t('settings.oauth.title') }}
    </h3>
    <p class="text-sm text-gray-500 dark:text-dark-400">
      {{ t('settings.oauth.description') }}
    </p>

    <!-- Loading -->
    <div v-if="loading" class="flex justify-center py-8">
      <div class="loading loading-spinner loading-lg text-primary"></div>
    </div>

    <!-- Providers List -->
    <div v-else class="space-y-3">
      <!-- Bound Providers -->
      <div
        v-for="binding in bindings"
        :key="binding.provider"
        class="flex items-center justify-between p-4 rounded-lg border border-gray-200 dark:border-dark-700 bg-white dark:bg-dark-800"
      >
        <div class="flex items-center gap-3">
          <component :is="getProviderIcon(binding.provider)" class="w-8 h-8" />
          <div>
            <div class="font-medium text-gray-900 dark:text-white">
              {{ getProviderDisplayName(binding.provider) }}
            </div>
            <div class="text-sm text-gray-500 dark:text-dark-400">
              {{ binding.provider_email || binding.provider_username || t('settings.oauth.bound') }}
            </div>
          </div>
        </div>
        <button
          class="btn btn-sm btn-ghost text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
          :disabled="unbinding === binding.provider"
          @click="handleUnbind(binding.provider)"
        >
          <span v-if="unbinding === binding.provider" class="loading loading-spinner loading-xs"></span>
          <span v-else>{{ t('settings.oauth.unbind') }}</span>
        </button>
      </div>

      <!-- Available Providers (not bound) -->
      <div
        v-for="provider in unboundProviders"
        :key="provider.name"
        class="flex items-center justify-between p-4 rounded-lg border border-dashed border-gray-300 dark:border-dark-600 bg-gray-50 dark:bg-dark-900"
      >
        <div class="flex items-center gap-3">
          <component :is="getProviderIcon(provider.name)" class="w-8 h-8 opacity-50" />
          <div>
            <div class="font-medium text-gray-700 dark:text-dark-300">
              {{ provider.display_name }}
            </div>
            <div class="text-sm text-gray-400 dark:text-dark-500">
              {{ t('settings.oauth.notBound') }}
            </div>
          </div>
        </div>
        <button
          class="btn btn-sm btn-outline btn-primary"
          :disabled="binding === provider.name"
          @click="handleBind(provider.name)"
        >
          <span v-if="binding === provider.name" class="loading loading-spinner loading-xs"></span>
          <span v-else>{{ t('settings.oauth.bind') }}</span>
        </button>
      </div>
    </div>

    <!-- Error Message -->
    <div v-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import {
  getSocialProviders,
  getUserBindings,
  startSocialBind,
  unbindSocialAccount,
  type SocialProvider,
  type UserOAuthBinding
} from '@/api/social'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const error = ref('')
const binding = ref<string | null>(null)
const unbinding = ref<string | null>(null)

const providers = ref<SocialProvider[]>([])
const bindings = ref<UserOAuthBinding[]>([])

// 未绑定的提供商
const unboundProviders = computed(() => {
  const boundNames = new Set(bindings.value.map(b => b.provider))
  return providers.value.filter(p => !boundNames.has(p.name))
})

// Provider icons
const GoogleIcon = {
  render() {
    return h('svg', { viewBox: '0 0 24 24', fill: 'currentColor', class: 'w-full h-full' }, [
      h('path', { d: 'M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z', fill: '#4285F4' }),
      h('path', { d: 'M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z', fill: '#34A853' }),
      h('path', { d: 'M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z', fill: '#FBBC05' }),
      h('path', { d: 'M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z', fill: '#EA4335' })
    ])
  }
}

const GitHubIcon = {
  render() {
    return h('svg', { viewBox: '0 0 24 24', fill: 'currentColor', class: 'w-full h-full' }, [
      h('path', { d: 'M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z' })
    ])
  }
}

function getProviderIcon(name: string) {
  switch (name) {
    case 'google':
      return GoogleIcon
    case 'github':
      return GitHubIcon
    default:
      return null
  }
}

function getProviderDisplayName(name: string): string {
  const provider = providers.value.find(p => p.name === name)
  if (provider) return provider.display_name
  return name.charAt(0).toUpperCase() + name.slice(1)
}

async function loadData() {
  try {
    loading.value = true
    error.value = ''
    const [providersData, bindingsData] = await Promise.all([
      getSocialProviders(),
      getUserBindings()
    ])
    providers.value = providersData
    bindings.value = bindingsData
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('settings.oauth.loadError')
  } finally {
    loading.value = false
  }
}

async function handleBind(provider: string) {
  try {
    binding.value = provider
    const result = await startSocialBind(provider)

    // 存储 session
    localStorage.setItem('social_oauth_bind_session', JSON.stringify({
      provider,
      session_id: result.session_id,
      timestamp: Date.now()
    }))

    // 跳转到授权页面
    window.location.href = result.auth_url
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : t('settings.oauth.bindError'))
  } finally {
    binding.value = null
  }
}

async function handleUnbind(provider: string) {
  if (!confirm(t('settings.oauth.confirmUnbind'))) return

  try {
    unbinding.value = provider
    await unbindSocialAccount(provider)
    appStore.showSuccess(t('settings.oauth.unbindSuccess'))
    await loadData()
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : t('settings.oauth.unbindError'))
  } finally {
    unbinding.value = null
  }
}

onMounted(() => {
  loadData()
})
</script>
