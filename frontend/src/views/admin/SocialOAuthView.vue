<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('admin.socialOAuth.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.socialOAuth.description') }}
          </p>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div class="loading loading-spinner loading-lg text-primary"></div>
      </div>

      <!-- Providers List -->
      <div v-else class="grid gap-6 md:grid-cols-2">
        <div
          v-for="provider in providers"
          :key="provider.name"
          class="card p-6 space-y-4"
        >
          <!-- Provider Header -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <component :is="getProviderIcon(provider.name)" class="w-10 h-10" />
              <div>
                <h3 class="font-semibold text-gray-900 dark:text-white">
                  {{ provider.display_name }}
                </h3>
                <p class="text-sm text-gray-500 dark:text-dark-400">
                  {{ provider.name }}
                </p>
              </div>
            </div>
            <label class="flex cursor-pointer items-center gap-2">
              <input
                type="checkbox"
                :checked="provider.enabled"
                class="toggle toggle-primary"
                @change="handleToggleEnabled(provider)"
              />
              <span class="text-sm">
                {{ provider.enabled ? t('common.enabled') : t('common.disabled') }}
              </span>
            </label>
          </div>

          <!-- Provider Config -->
          <div class="space-y-3">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-dark-300 mb-1">
                Client ID
              </label>
              <input
                v-model="editingProviders[provider.name].client_id"
                type="text"
                class="input w-full"
                :placeholder="t('admin.socialOAuth.clientIdPlaceholder')"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-dark-300 mb-1">
                Client Secret
              </label>
              <div class="relative">
                <input
                  v-model="editingProviders[provider.name].client_secret"
                  :type="showSecrets[provider.name] ? 'text' : 'password'"
                  class="input w-full pr-10"
                  :placeholder="provider.client_secret_set
                    ? t('admin.socialOAuth.secretSetPlaceholder')
                    : t('admin.socialOAuth.clientSecretPlaceholder')"
                />
                <button
                  type="button"
                  class="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-gray-600"
                  @click="showSecrets[provider.name] = !showSecrets[provider.name]"
                >
                  <Icon :name="showSecrets[provider.name] ? 'eyeOff' : 'eye'" size="md" />
                </button>
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-dark-300 mb-1">
                Redirect URI
              </label>
              <div class="flex gap-2">
                <input
                  :value="getRedirectUri(provider.name)"
                  type="text"
                  class="input w-full bg-gray-50 dark:bg-dark-800"
                  readonly
                />
                <button
                  type="button"
                  class="btn btn-ghost btn-sm"
                  @click="copyRedirectUri(provider.name)"
                >
                  <Icon name="copy" size="md" />
                </button>
              </div>
              <p class="text-xs text-gray-500 dark:text-dark-400 mt-1">
                {{ t('admin.socialOAuth.redirectUriHint') }}
              </p>
            </div>
          </div>

          <!-- Save Button -->
          <div class="flex justify-end pt-2">
            <button
              class="btn btn-primary btn-sm"
              :disabled="saving[provider.name]"
              @click="handleSave(provider.name)"
            >
              <span v-if="saving[provider.name]" class="loading loading-spinner loading-xs"></span>
              <span v-else>{{ t('common.save') }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div
        v-if="!loading && providers.length === 0"
        class="text-center py-12 text-gray-500 dark:text-dark-400"
      >
        {{ t('admin.socialOAuth.noProviders') }}
      </div>

      <!-- Error Message -->
      <div v-if="error" class="alert alert-error">
        <span>{{ error }}</span>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  getProviders,
  updateProvider,
  type SocialOAuthProvider,
  type UpdateProviderRequest
} from '@/api/admin/socialOAuth'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const error = ref('')
const providers = ref<SocialOAuthProvider[]>([])
const editingProviders = reactive<Record<string, { client_id: string; client_secret: string }>>({})
const showSecrets = reactive<Record<string, boolean>>({})
const saving = reactive<Record<string, boolean>>({})

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

function getRedirectUri(providerName: string): string {
  const baseUrl = window.location.origin
  return `${baseUrl}/auth/social/login/callback?provider=${providerName}`
}

async function copyRedirectUri(providerName: string) {
  try {
    await navigator.clipboard.writeText(getRedirectUri(providerName))
    appStore.showSuccess(t('common.copied'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

async function loadProviders() {
  try {
    loading.value = true
    error.value = ''
    providers.value = await getProviders()

    // Initialize editing state for each provider
    providers.value.forEach(provider => {
      editingProviders[provider.name] = {
        client_id: provider.client_id || '',
        client_secret: ''
      }
      showSecrets[provider.name] = false
      saving[provider.name] = false
    })
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('admin.socialOAuth.loadError')
  } finally {
    loading.value = false
  }
}

async function handleToggleEnabled(provider: SocialOAuthProvider) {
  try {
    saving[provider.name] = true
    await updateProvider(provider.name, {
      enabled: !provider.enabled
    })
    provider.enabled = !provider.enabled
    appStore.showSuccess(t('admin.socialOAuth.updateSuccess'))
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : t('admin.socialOAuth.updateError'))
  } finally {
    saving[provider.name] = false
  }
}

async function handleSave(providerName: string) {
  try {
    saving[providerName] = true

    const data: UpdateProviderRequest = {}
    const editing = editingProviders[providerName]

    if (editing.client_id) {
      data.client_id = editing.client_id
    }
    if (editing.client_secret) {
      data.client_secret = editing.client_secret
    }

    const updated = await updateProvider(providerName, data)

    // Update provider in list
    const index = providers.value.findIndex(p => p.name === providerName)
    if (index !== -1) {
      providers.value[index] = updated
      editingProviders[providerName].client_id = updated.client_id
      editingProviders[providerName].client_secret = ''
    }

    appStore.showSuccess(t('admin.socialOAuth.updateSuccess'))
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : t('admin.socialOAuth.updateError'))
  } finally {
    saving[providerName] = false
  }
}

onMounted(() => {
  loadProviders()
})
</script>
