<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-dark-900">
    <div class="max-w-md w-full p-6">
      <div v-if="loading" class="text-center">
        <div class="loading loading-spinner loading-lg text-primary mb-4"></div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ t('auth.social.processing') }}
        </h2>
        <p class="text-gray-500 dark:text-dark-400 mt-2">
          {{ t('auth.social.pleaseWait') }}
        </p>
      </div>

      <div v-else-if="error" class="text-center">
        <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-100 dark:bg-red-900/20 flex items-center justify-center">
          <svg class="w-8 h-8 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ isBind ? t('settings.oauth.bindError') : t('auth.social.loginFailed') }}
        </h2>
        <p class="text-gray-500 dark:text-dark-400 mt-2">
          {{ errorMessage }}
        </p>
        <button class="btn btn-primary mt-4" @click="goBack">
          {{ isBind ? t('common.back') : t('auth.social.backToLogin') }}
        </button>
      </div>

      <div v-else-if="success" class="text-center">
        <div class="w-16 h-16 mx-auto mb-4 rounded-full bg-green-100 dark:bg-green-900/20 flex items-center justify-center">
          <svg class="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ isBind ? t('settings.oauth.bindSuccess') : t('auth.social.loginSuccess') }}
        </h2>
        <p class="text-gray-500 dark:text-dark-400 mt-2">
          {{ t('auth.social.redirecting') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { handleSocialLoginCallback, handleSocialBindCallback } from '@/api/social'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const error = ref(false)
const success = ref(false)
const errorMessage = ref('')

// 通过 localStorage session 类型判断是登录还是绑定
// 绑定和登录现在使用相同的回调 URL，避免 OAuth 提供商需要额外配置
const isBind = ref(false)

onMounted(async () => {
  try {
    // 从 URL 获取参数
    const code = route.query.code as string
    const state = route.query.state as string
    const provider = route.query.provider as string

    if (!code || !state) {
      throw new Error(t('auth.social.missingParams'))
    }

    // 优先检查绑定 session，因为绑定和登录现在使用相同的回调 URL
    const bindSessionData = localStorage.getItem('social_oauth_bind_session')
    const loginSessionData = localStorage.getItem('social_oauth_session')

    let sessionData: string | null = null
    let sessionKey: string

    if (bindSessionData) {
      // 存在绑定 session，这是绑定操作
      sessionData = bindSessionData
      sessionKey = 'social_oauth_bind_session'
      isBind.value = true
    } else if (loginSessionData) {
      // 存在登录 session，这是登录操作
      sessionData = loginSessionData
      sessionKey = 'social_oauth_session'
      isBind.value = false
    } else {
      throw new Error(t('auth.social.sessionExpired'))
    }

    const session = JSON.parse(sessionData)

    // 检查 session 是否过期（10分钟）
    if (Date.now() - session.timestamp > 10 * 60 * 1000) {
      localStorage.removeItem(sessionKey)
      throw new Error(t('auth.social.sessionExpired'))
    }

    // 验证 provider
    const sessionProvider = provider || session.provider
    if (!sessionProvider) {
      throw new Error(t('auth.social.invalidProvider'))
    }

    if (isBind.value) {
      // 处理绑定回调
      await handleSocialBindCallback(
        sessionProvider,
        code,
        state,
        session.session_id
      )

      // 清除 session
      localStorage.removeItem(sessionKey)

      success.value = true
      loading.value = false

      // 跳转回个人设置页面
      setTimeout(() => {
        router.push('/profile')
      }, 1000)
    } else {
      // 处理登录回调
      const result = await handleSocialLoginCallback(
        sessionProvider,
        code,
        state,
        session.session_id
      )

      // 清除 session
      localStorage.removeItem(sessionKey)

      // 保存 token 并获取用户信息
      await authStore.setToken(result.token)

      success.value = true
      loading.value = false

      // 跳转到目标页面
      setTimeout(() => {
        router.push(result.redirect_to || '/dashboard')
      }, 1000)
    }
  } catch (err) {
    loading.value = false
    error.value = true
    errorMessage.value = err instanceof Error ? err.message : t('auth.social.unknownError')
    // 清除所有可能的 session
    localStorage.removeItem('social_oauth_session')
    localStorage.removeItem('social_oauth_bind_session')
  }
})

function goBack() {
  if (isBind.value) {
    router.push('/profile')
  } else {
    router.push('/login')
  }
}
</script>
