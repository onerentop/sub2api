<template>
  <div class="w-full">
    <label v-if="label" class="input-label mb-1.5 block">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>

    <!-- Tags Display Area -->
    <div
      class="min-h-[42px] w-full rounded-lg border border-gray-200 bg-white p-2 transition-all duration-200 focus-within:border-primary-500 focus-within:ring-2 focus-within:ring-primary-500/20 dark:border-dark-600 dark:bg-dark-800"
      :class="[disabled ? 'cursor-not-allowed bg-gray-100 opacity-60 dark:bg-dark-900' : '']"
    >
      <!-- Selected Tags -->
      <div class="flex flex-wrap gap-1.5">
        <span
          v-for="(tag, index) in tags"
          :key="tag"
          class="inline-flex items-center gap-1 rounded-md bg-primary-100 px-2 py-1 text-sm font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
        >
          {{ tag }}
          <button
            v-if="!disabled"
            type="button"
            class="ml-0.5 rounded-sm p-0.5 hover:bg-primary-200 dark:hover:bg-primary-800"
            @click="removeTag(index)"
          >
            <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </span>

        <!-- Input Field -->
        <input
          v-if="!disabled"
          ref="inputRef"
          v-model="inputValue"
          type="text"
          class="min-w-[120px] flex-1 border-0 bg-transparent p-1 text-sm outline-none placeholder:text-gray-400 dark:placeholder:text-dark-400"
          :placeholder="tags.length === 0 ? placeholder : ''"
          @keydown.enter.prevent="addCustomTag"
          @keydown.backspace="handleBackspace"
          @focus="showSuggestions = true"
          @blur="handleBlur"
        />
      </div>
    </div>

    <!-- Suggestions Dropdown -->
    <div
      v-if="showSuggestions && filteredSuggestions.length > 0 && !disabled"
      class="relative z-10"
    >
      <div
        class="absolute left-0 right-0 mt-1 max-h-48 overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800"
      >
        <button
          v-for="suggestion in filteredSuggestions"
          :key="suggestion"
          type="button"
          class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-dark-700"
          @mousedown.prevent="addTag(suggestion)"
        >
          <svg
            class="h-4 w-4 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          {{ suggestion }}
        </button>
      </div>
    </div>

    <!-- Hint Text -->
    <p v-if="hint" class="input-hint mt-1.5">
      {{ hint }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface Props {
  modelValue: string[] | null | undefined
  suggestions?: string[]
  label?: string
  placeholder?: string
  hint?: string
  disabled?: boolean
  required?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  suggestions: () => [],
  placeholder: '',
  disabled: false,
  required: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string[]): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)
const inputValue = ref('')
const showSuggestions = ref(false)

// Safely get tags array (handle null/undefined)
const tags = computed(() => props.modelValue ?? [])

// Filter suggestions to exclude already selected tags
const filteredSuggestions = computed(() => {
  const query = inputValue.value.toLowerCase().trim()
  return props.suggestions.filter(
    (s) => !tags.value.includes(s) && (query === '' || s.toLowerCase().includes(query))
  )
})

const addTag = (tag: string) => {
  const normalizedTag = tag.toLowerCase().trim()
  if (normalizedTag && !tags.value.includes(normalizedTag)) {
    emit('update:modelValue', [...tags.value, normalizedTag])
  }
  inputValue.value = ''
}

const addCustomTag = () => {
  const value = inputValue.value.trim()
  if (value) {
    addTag(value)
  }
}

const removeTag = (index: number) => {
  const newValue = [...tags.value]
  newValue.splice(index, 1)
  emit('update:modelValue', newValue)
}

const handleBackspace = () => {
  if (inputValue.value === '' && tags.value.length > 0) {
    removeTag(tags.value.length - 1)
  }
}

const handleBlur = () => {
  // Delay hiding to allow click on suggestions
  setTimeout(() => {
    showSuggestions.value = false
  }, 150)
}

// Expose focus method
defineExpose({
  focus: () => inputRef.value?.focus()
})
</script>
