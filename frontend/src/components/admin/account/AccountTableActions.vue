<template>
  <div class="flex flex-wrap items-center gap-3">
    <slot name="before"></slot>
    <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary">
      <Icon name="refresh" size="md" :class="[loading ? 'animate-spin' : '']" />
    </button>
    <slot name="after"></slot>
    <button @click="$emit('sync')" class="btn btn-secondary">{{ t('admin.accounts.syncFromCrs') }}</button>
    <button @click="$emit('export')" :disabled="exporting" class="btn btn-secondary">
      <Icon name="download" size="md" class="mr-1" />
      {{ t('admin.accounts.exportAccounts') }}
    </button>
    <button @click="$emit('import')" class="btn btn-secondary">
      <Icon name="upload" size="md" class="mr-1" />
      {{ t('admin.accounts.importAccounts') }}
    </button>
    <button @click="$emit('create')" class="btn btn-primary">{{ t('admin.accounts.createAccount') }}</button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps(['loading', 'exporting'])
defineEmits(['refresh', 'sync', 'create', 'export', 'import'])

const { t } = useI18n()
</script>
