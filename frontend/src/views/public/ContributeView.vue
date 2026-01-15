<template>
  <AuthLayout>
    <div class="space-y-6">
      <!-- Title -->
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('contribute.title') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('contribute.description') }}
        </p>
      </div>

      <!-- Step Indicator -->
      <div class="flex items-center justify-center space-x-4">
        <div
          v-for="(_, index) in steps"
          :key="index"
          class="flex items-center"
        >
          <div
            class="flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors"
            :class="getStepClass(index)"
          >
            <Icon v-if="index < step" name="check" size="sm" />
            <span v-else>{{ index + 1 }}</span>
          </div>
          <div
            v-if="index < steps.length - 1"
            class="ml-2 h-0.5 w-8 bg-gray-200 dark:bg-dark-700"
            :class="{ 'bg-primary-500 dark:bg-primary-500': index < step }"
          />
        </div>
      </div>

      <!-- Step 1: Start OAuth -->
      <div v-if="step === 0" class="space-y-4">
        <div class="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
          <div class="flex">
            <Icon name="infoCircle" size="md" class="text-blue-500" />
            <div class="ml-3">
              <p class="text-sm text-blue-700 dark:text-blue-300">
                {{ t('contribute.step1Info') }}
              </p>
            </div>
          </div>
        </div>

        <button
          @click="handleStartOAuth"
          :disabled="isLoading"
          class="btn btn-primary w-full"
        >
          <Icon v-if="isLoading" name="refresh" size="sm" class="mr-2 animate-spin" />
          <Icon v-else name="externalLink" size="sm" class="mr-2" />
          {{ t('contribute.startAuth') }}
        </button>
      </div>

      <!-- Step 2: Enter Code -->
      <div v-if="step === 1" class="space-y-4">
        <div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-900/20">
          <div class="flex">
            <Icon name="exclamationTriangle" size="md" class="text-amber-500" />
            <div class="ml-3">
              <p class="text-sm text-amber-700 dark:text-amber-300">
                {{ t('contribute.step2Info') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Auth URL Display -->
        <div class="space-y-2">
          <label class="input-label">{{ t('contribute.authUrlLabel') }}</label>
          <div class="flex space-x-2">
            <input
              :value="authUrl"
              readonly
              class="input flex-1 text-xs"
            />
            <button
              @click="copyAuthUrl"
              class="btn btn-secondary"
              :title="t('common.copy')"
            >
              <Icon name="copy" size="sm" />
            </button>
            <a
              :href="authUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-primary"
              :title="t('contribute.openInNewTab')"
            >
              <Icon name="externalLink" size="sm" />
            </a>
          </div>
        </div>

        <!-- Code Input -->
        <div class="space-y-2">
          <label for="callbackUrl" class="input-label">
            {{ t('contribute.callbackUrlLabel') }}
          </label>
          <textarea
            id="callbackUrl"
            v-model="callbackUrl"
            rows="3"
            class="input font-mono text-xs"
            :placeholder="t('contribute.callbackUrlPlaceholder')"
          />
          <p class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('contribute.callbackUrlHint') }}
          </p>
        </div>

        <div class="flex space-x-3">
          <button
            @click="step = 0"
            class="btn btn-secondary flex-1"
          >
            {{ t('common.back') }}
          </button>
          <button
            @click="handleSubmitCode"
            :disabled="isLoading || !callbackUrl.trim()"
            class="btn btn-primary flex-1"
          >
            <Icon v-if="isLoading" name="refresh" size="sm" class="mr-2 animate-spin" />
            {{ t('contribute.submit') }}
          </button>
        </div>
      </div>

      <!-- Step 3: Success -->
      <div v-if="step === 2" class="space-y-4 text-center">
        <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
          <Icon name="check" size="lg" class="text-green-600 dark:text-green-400" />
        </div>

        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('contribute.successTitle') }}
          </h3>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
            {{ result?.message || t('contribute.successMessage') }}
          </p>
          <p v-if="result?.email" class="mt-1 text-sm font-medium text-primary-600 dark:text-primary-400">
            {{ result.email }}
          </p>
        </div>

        <!-- Wake Test Section -->
        <div v-if="result?.session_id" class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <h4 class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('contribute.wakeTest') }}
          </h4>
          <p class="mb-3 text-xs text-gray-500 dark:text-dark-400">
            {{ t('contribute.wakeTestDescription') }}
          </p>

          <!-- Wake Result -->
          <div v-if="wakeResult" class="mb-3 rounded-lg p-3" :class="wakeResult.success ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20'">
            <div class="flex items-start">
              <Icon
                :name="wakeResult.success ? 'check' : 'exclamationCircle'"
                size="sm"
                :class="wakeResult.success ? 'text-green-500' : 'text-red-500'"
              />
              <div class="ml-2 flex-1 text-left">
                <p class="text-sm font-medium" :class="wakeResult.success ? 'text-green-700 dark:text-green-300' : 'text-red-700 dark:text-red-300'">
                  {{ wakeResult.success ? t('contribute.wakeSuccess') : t('contribute.wakeFailed') }}
                </p>
                <p v-if="wakeResult.model" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('contribute.wakeModel') }}: {{ wakeResult.model }}
                </p>
                <p v-if="wakeResult.duration != null" class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('contribute.wakeDuration') }}: {{ wakeResult.duration }}ms
                </p>
                <p v-if="wakeResult.text" class="mt-2 rounded bg-white p-2 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  {{ wakeResult.text }}
                </p>
                <p v-if="!wakeResult.success && wakeResult.message" class="mt-1 text-xs text-red-600 dark:text-red-400">
                  {{ wakeResult.message }}
                </p>
              </div>
            </div>
          </div>

          <button
            @click="handleWakeTest"
            :disabled="isWaking"
            class="btn btn-secondary w-full"
          >
            <Icon v-if="isWaking" name="refresh" size="sm" class="mr-2 animate-spin" />
            <Icon v-else name="play" size="sm" class="mr-2" />
            {{ isWaking ? t('contribute.waking') : t('contribute.wakeNow') }}
          </button>
        </div>

        <button
          @click="resetForm"
          class="btn btn-primary"
        >
          {{ t('contribute.contributeAnother') }}
        </button>
      </div>

      <!-- Error Display -->
      <div v-if="error" class="rounded-lg bg-red-50 p-4 dark:bg-red-900/20">
        <div class="flex">
          <Icon name="exclamationCircle" size="md" class="text-red-500" />
          <div class="ml-3">
            <p class="text-sm text-red-700 dark:text-red-300">
              {{ error }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <template #footer>
      <router-link
        to="/home"
        class="text-gray-500 transition-colors hover:text-primary-600 dark:text-dark-400 dark:hover:text-primary-400"
      >
        {{ t('common.backToHome') }}
      </router-link>
    </template>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { startAntigravityOAuth, completeAntigravityOAuth, wakeAntigravity, type OAuthCompleteResponse, type WakeResponse } from '@/api/public'

const { t } = useI18n()

// State
const step = ref(0)
const isLoading = ref(false)
const error = ref('')
const authUrl = ref('')
const sessionId = ref('')
const state = ref('')
const callbackUrl = ref('')
const result = ref<OAuthCompleteResponse | null>(null)

// Wake test state
const isWaking = ref(false)
const wakeResult = ref<WakeResponse | null>(null)

const steps = [
  { label: 'Start' },
  { label: 'Authorize' },
  { label: 'Complete' }
]

// Methods
function getStepClass(index: number) {
  if (index < step.value) {
    return 'bg-primary-500 text-white'
  } else if (index === step.value) {
    return 'bg-primary-100 text-primary-600 dark:bg-primary-900/50 dark:text-primary-400'
  }
  return 'bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-dark-500'
}

async function handleStartOAuth() {
  isLoading.value = true
  error.value = ''

  try {
    const response = await startAntigravityOAuth()
    authUrl.value = response.auth_url
    sessionId.value = response.session_id
    state.value = response.state
    step.value = 1
  } catch (err: any) {
    console.error('OAuth start error:', err)
    const errMsg = err.response?.data?.message || err.message || t('contribute.errorStarting')
    error.value = errMsg
  } finally {
    isLoading.value = false
  }
}

function copyAuthUrl() {
  navigator.clipboard.writeText(authUrl.value)
}

function extractCodeFromUrl(url: string): string | null {
  try {
    // 尝试解析为 URL
    const urlObj = new URL(url)
    return urlObj.searchParams.get('code')
  } catch {
    // 如果不是有效 URL，检查是否直接是 code
    if (url.length > 20 && !url.includes(' ')) {
      return url.trim()
    }
    return null
  }
}

async function handleSubmitCode() {
  isLoading.value = true
  error.value = ''

  try {
    const code = extractCodeFromUrl(callbackUrl.value.trim())
    if (!code) {
      error.value = t('contribute.invalidCode')
      isLoading.value = false
      return
    }

    const response = await completeAntigravityOAuth({
      session_id: sessionId.value,
      state: state.value,
      code: code
    })

    result.value = response
    step.value = 2
  } catch (err: any) {
    error.value = err.response?.data?.message || err.message || t('contribute.errorSubmitting')
  } finally {
    isLoading.value = false
  }
}

function resetForm() {
  step.value = 0
  authUrl.value = ''
  sessionId.value = ''
  state.value = ''
  callbackUrl.value = ''
  result.value = null
  error.value = ''
  // Reset wake state
  isWaking.value = false
  wakeResult.value = null
}

// Wake test handler
async function handleWakeTest() {
  if (!result.value?.session_id) {
    return
  }

  isWaking.value = true
  wakeResult.value = null

  try {
    const response = await wakeAntigravity({
      session_id: result.value.session_id
    })
    wakeResult.value = response
  } catch (err: any) {
    wakeResult.value = {
      success: false,
      message: err.response?.data?.message || err.message || t('contribute.wakeError'),
      model: ''
    }
  } finally {
    isWaking.value = false
  }
}
</script>
