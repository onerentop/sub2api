<template>
  <BaseDialog :show="show" :title="t('admin.accounts.importAccounts')" size="lg" @close="$emit('close')">
    <div class="space-y-6">
      <!-- File Upload Area -->
      <div
        class="border-2 border-dashed rounded-lg p-8 text-center transition-colors"
        :class="[
          isDragging
            ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
            : 'border-gray-300 dark:border-dark-600 hover:border-gray-400 dark:hover:border-dark-500'
        ]"
        @dragover.prevent="isDragging = true"
        @dragleave.prevent="isDragging = false"
        @drop.prevent="handleFileDrop"
      >
        <input type="file" ref="fileInput" accept=".json,.zip" multiple class="hidden" @change="handleFileSelect" />
        <div v-if="!previewData" class="space-y-2">
          <Icon name="upload" size="xl" class="mx-auto text-gray-400" />
          <p class="text-gray-600 dark:text-gray-300">{{ t('admin.accounts.dropFileHere') }}</p>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.accounts.supportedFormats') }}</p>
          <button @click="fileInput?.click()" class="btn btn-secondary">
            {{ t('admin.accounts.selectFile') }}
          </button>
        </div>
        <div v-else class="space-y-2">
          <Icon name="checkCircle" size="xl" class="mx-auto text-green-500" />
          <p class="text-gray-900 dark:text-white font-medium">{{ fileName }}</p>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.accountsToImport', { count: previewData.length }) }}
          </p>
          <button @click="clearFile" class="btn btn-secondary btn-sm">
            {{ t('common.clear') }}
          </button>
        </div>
      </div>

      <!-- Preview Table -->
      <div v-if="previewData && previewData.length > 0" class="border rounded-lg overflow-hidden dark:border-dark-600">
        <div class="max-h-64 overflow-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-600">
            <thead class="bg-gray-50 dark:bg-dark-700 sticky top-0">
              <tr>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.type') }}</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.email') }}</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.accounts.columns.tokenStatus') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
              <tr v-for="(item, index) in previewData.slice(0, 20)" :key="index" class="hover:bg-gray-50 dark:hover:bg-dark-700">
                <td class="px-4 py-2 text-sm text-gray-900 dark:text-white">{{ item.type }}</td>
                <td class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300">{{ item.email || '-' }}</td>
                <td class="px-4 py-2">
                  <span
                    class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
                    :class="[
                      item.access_token || item.refresh_token
                        ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                        : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                    ]"
                  >
                    {{ item.access_token || item.refresh_token ? t('admin.accounts.hasToken') : t('admin.accounts.noToken') }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="previewData.length > 20" class="px-4 py-2 text-sm text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-dark-700 border-t dark:border-dark-600">
          {{ t('admin.accounts.andMoreAccounts', { count: previewData.length - 20 }) }}
        </div>
      </div>

      <!-- Options -->
      <div v-if="previewData && previewData.length > 0" class="space-y-4">
        <div class="flex items-center gap-3">
          <input type="checkbox" v-model="skipExisting" id="skipExisting" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <label for="skipExisting" class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.skipExisting') }}</label>
        </div>
      </div>

      <!-- Processing Indicator -->
      <div v-if="processing" class="flex items-center justify-center gap-2 py-4">
        <Icon name="refresh" size="md" class="animate-spin text-primary-500" />
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.processingFiles') }}</span>
      </div>

      <!-- Error Message -->
      <div v-if="parseError" class="rounded-md bg-red-50 dark:bg-red-900/20 p-4">
        <div class="flex">
          <Icon name="exclamationCircle" size="md" class="text-red-400" />
          <div class="ml-3">
            <p class="text-sm text-red-700 dark:text-red-300">{{ parseError }}</p>
          </div>
        </div>
      </div>

      <!-- Import Results -->
      <div v-if="importResult" class="space-y-3">
        <div class="rounded-md bg-gray-50 dark:bg-dark-700 p-4">
          <div class="grid grid-cols-4 gap-4 text-center">
            <div>
              <p class="text-2xl font-bold text-green-600 dark:text-green-400">{{ importResult.created }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.created') }}</p>
            </div>
            <div>
              <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ importResult.updated }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.updated') }}</p>
            </div>
            <div>
              <p class="text-2xl font-bold text-gray-600 dark:text-gray-400">{{ importResult.skipped }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.skipped') }}</p>
            </div>
            <div>
              <p class="text-2xl font-bold text-red-600 dark:text-red-400">{{ importResult.failed }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.failed') }}</p>
            </div>
          </div>
        </div>

        <!-- Failed Items Details -->
        <div v-if="failedItems.length > 0" class="border rounded-lg overflow-hidden dark:border-dark-600">
          <div class="bg-red-50 dark:bg-red-900/20 px-4 py-2 border-b dark:border-dark-600">
            <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ t('admin.accounts.failedItems') }}</p>
          </div>
          <div class="max-h-32 overflow-auto">
            <ul class="divide-y divide-gray-200 dark:divide-dark-600">
              <li v-for="(item, index) in failedItems" :key="index" class="px-4 py-2 text-sm">
                <span class="text-gray-900 dark:text-white">{{ item.email || item.type }}</span>
                <span class="text-red-600 dark:text-red-400 ml-2">{{ item.error }}</span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" class="btn btn-secondary">{{ t('common.close') }}</button>
        <button
          v-if="previewData && !importResult"
          @click="handleImport"
          :disabled="importing || previewData.length === 0"
          class="btn btn-primary"
        >
          <Icon v-if="importing" name="refresh" size="md" class="animate-spin mr-2" />
          {{ importing ? t('admin.accounts.importing') : t('admin.accounts.importAccounts') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { CLIProxyAuth, ImportResult } from '@/api/admin/accounts'
import JSZip from 'jszip'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  imported: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const fileInput = ref<HTMLInputElement | null>(null)
const isDragging = ref(false)
const fileName = ref('')
const previewData = ref<CLIProxyAuth[] | null>(null)
const parseError = ref('')
const skipExisting = ref(true)
const importing = ref(false)
const processing = ref(false)
const importResult = ref<{
  created: number
  updated: number
  skipped: number
  failed: number
  results: ImportResult[]
} | null>(null)

const failedItems = computed(() => {
  if (!importResult.value) return []
  return importResult.value.results.filter(r => r.action === 'failed')
})

watch(() => props.show, (newVal) => {
  if (newVal) {
    clearFile()
    importResult.value = null
  }
})

const clearFile = () => {
  fileName.value = ''
  previewData.value = null
  parseError.value = ''
  processing.value = false
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

const handleFileDrop = (e: DragEvent) => {
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    processFiles(Array.from(files))
  }
}

const handleFileSelect = (e: Event) => {
  const target = e.target as HTMLInputElement
  const files = target.files
  if (files && files.length > 0) {
    processFiles(Array.from(files))
  }
}

/**
 * Process multiple files - supports JSON files and ZIP archives
 */
const processFiles = async (files: File[]) => {
  parseError.value = ''
  importResult.value = null
  processing.value = true

  try {
    const allAccounts: CLIProxyAuth[] = []
    const fileNames: string[] = []

    for (const file of files) {
      if (file.name.endsWith('.zip')) {
        // Process ZIP file - extract and parse all JSON files
        const accounts = await processZipFile(file)
        allAccounts.push(...accounts)
        fileNames.push(`${file.name} (${accounts.length} accounts)`)
      } else if (file.name.endsWith('.json')) {
        // Process single JSON file
        const accounts = await processJsonFile(file)
        allAccounts.push(...accounts)
        fileNames.push(file.name)
      } else {
        // Skip unsupported file
        console.warn(`Skipping unsupported file: ${file.name}`)
      }
    }

    if (allAccounts.length === 0) {
      parseError.value = t('admin.accounts.noValidAccounts')
      return
    }

    fileName.value = fileNames.length === 1 ? fileNames[0] : `${fileNames.length} files`
    previewData.value = allAccounts
  } catch (error) {
    console.error('Failed to process files:', error)
    parseError.value = t('admin.accounts.invalidFileFormat')
  } finally {
    processing.value = false
  }
}

/**
 * Process a single JSON file
 * Supports both single object (CLIProxyAPI format) and array format
 */
const processJsonFile = async (file: File): Promise<CLIProxyAuth[]> => {
  const text = await file.text()
  const data = JSON.parse(text)

  // CLIProxyAPI format: single object per file
  if (!Array.isArray(data)) {
    if (data.type && (data.access_token || data.refresh_token)) {
      return [data as CLIProxyAuth]
    }
    throw new Error('Invalid JSON format')
  }

  // Array format (legacy)
  return data as CLIProxyAuth[]
}

/**
 * Process a ZIP file - extract and parse all JSON files within
 */
const processZipFile = async (file: File): Promise<CLIProxyAuth[]> => {
  const zip = await JSZip.loadAsync(file)
  const accounts: CLIProxyAuth[] = []

  const jsonFiles = Object.keys(zip.files).filter(name => name.endsWith('.json') && !zip.files[name].dir)

  for (const jsonFileName of jsonFiles) {
    try {
      const content = await zip.files[jsonFileName].async('string')
      const data = JSON.parse(content)

      // CLIProxyAPI format: single object per file
      if (!Array.isArray(data)) {
        if (data.type && (data.access_token || data.refresh_token)) {
          accounts.push(data as CLIProxyAuth)
        }
      } else {
        // Array format (less common in ZIP)
        accounts.push(...(data as CLIProxyAuth[]))
      }
    } catch (error) {
      console.warn(`Failed to parse JSON file in ZIP: ${jsonFileName}`, error)
    }
  }

  return accounts
}

const handleImport = async () => {
  if (!previewData.value || previewData.value.length === 0) return

  importing.value = true
  try {
    const result = await adminAPI.accounts.importAccounts({
      accounts: previewData.value,
      skip_existing: skipExisting.value
    })
    importResult.value = result

    if (result.created > 0 || result.updated > 0) {
      appStore.showSuccess(t('admin.accounts.importSuccess', {
        created: result.created,
        updated: result.updated,
        skipped: result.skipped
      }))
      emit('imported')
    }
  } catch (error: any) {
    appStore.showError(error.message || t('admin.accounts.importFailed'))
  } finally {
    importing.value = false
  }
}
</script>
