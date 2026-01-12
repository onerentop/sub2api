<template>
  <Transition name="slide-down">
    <div v-if="showBar && visibleAnnouncements.length > 0" class="announcement-bar" :class="currentTypeClass">
      <div class="announcement-container">
        <!-- Carousel Content -->
        <Transition name="fade" mode="out-in">
          <div :key="currentIndex" class="announcement-content">
            <!-- Type Icon -->
            <Icon :name="currentIconName" class="announcement-icon" />

            <!-- Content -->
            <div class="announcement-text">
              <span v-if="currentAnnouncement?.title" class="announcement-title">
                {{ currentAnnouncement.title }}:
              </span>
              <span v-html="currentAnnouncement?.content" class="announcement-body"></span>
            </div>

            <!-- Carousel Indicators -->
            <div v-if="visibleAnnouncements.length > 1" class="carousel-indicators">
              <button
                v-for="(_, index) in visibleAnnouncements"
                :key="index"
                class="indicator-dot"
                :class="{ active: index === currentIndex }"
                @click="goToAnnouncement(index)"
                :aria-label="`Go to announcement ${index + 1}`"
              />
            </div>
          </div>
        </Transition>

        <!-- Close Button -->
        <button
          @click="closeBar"
          class="close-button"
          :aria-label="t('announcement.close')"
        >
          <Icon name="x" size="sm" />
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getActiveAnnouncements } from '@/api/announcements'
import type { Announcement } from '@/types'

const { t } = useI18n()

// State
const announcements = ref<Announcement[]>([])
const settings = ref({ enabled: true, interval: 5000 })
const currentIndex = ref(0)
const showBar = ref(true)
const isLoading = ref(false)
let carouselTimer: ReturnType<typeof setInterval> | null = null

// Session storage key for closed announcements
const CLOSED_KEY = 'announcement_bar_closed'
const CLOSED_IDS_KEY = 'announcement_closed_ids'

// Filter out closed announcements
const visibleAnnouncements = computed(() => {
  const closedIds = getClosedIds()
  return announcements.value.filter((a) => !closedIds.includes(a.id))
})

const currentAnnouncement = computed(() => {
  return visibleAnnouncements.value[currentIndex.value] || null
})

const currentTypeClass = computed(() => {
  const type = currentAnnouncement.value?.type || 'info'
  return `announcement-${type}`
})

const currentIconName = computed((): 'infoCircle' | 'checkCircle' | 'exclamationTriangle' | 'exclamationCircle' => {
  const type = currentAnnouncement.value?.type || 'info'
  const icons = {
    info: 'infoCircle',
    success: 'checkCircle',
    warning: 'exclamationTriangle',
    error: 'exclamationCircle'
  } as const
  return icons[type]
})

// Functions
function getClosedIds(): number[] {
  try {
    const stored = sessionStorage.getItem(CLOSED_IDS_KEY)
    return stored ? JSON.parse(stored) : []
  } catch {
    return []
  }
}

function setClosedIds(ids: number[]) {
  sessionStorage.setItem(CLOSED_IDS_KEY, JSON.stringify(ids))
}

function closeBar() {
  // Add current announcement to closed list
  if (currentAnnouncement.value) {
    const closedIds = getClosedIds()
    closedIds.push(currentAnnouncement.value.id)
    setClosedIds(closedIds)
  }

  // If there are more announcements, move to next
  if (visibleAnnouncements.value.length > 1) {
    // Just trigger reactivity - computed will filter out the closed one
    currentIndex.value = 0
  } else {
    // No more announcements, hide the bar
    showBar.value = false
    sessionStorage.setItem(CLOSED_KEY, 'true')
  }
}

function goToAnnouncement(index: number) {
  currentIndex.value = index
  resetCarouselTimer()
}

function nextAnnouncement() {
  if (visibleAnnouncements.value.length > 1) {
    currentIndex.value = (currentIndex.value + 1) % visibleAnnouncements.value.length
  }
}

function startCarousel() {
  if (carouselTimer) {
    clearInterval(carouselTimer)
  }
  if (visibleAnnouncements.value.length > 1 && settings.value.interval > 0) {
    carouselTimer = setInterval(nextAnnouncement, settings.value.interval)
  }
}

function resetCarouselTimer() {
  startCarousel()
}

async function loadAnnouncements() {
  // Check if bar was closed for this session
  if (sessionStorage.getItem(CLOSED_KEY) === 'true') {
    showBar.value = false
    return
  }

  isLoading.value = true
  try {
    const response = await getActiveAnnouncements()
    announcements.value = response.announcements || []
    settings.value = response.settings || { enabled: true, interval: 5000 }

    if (!settings.value.enabled) {
      showBar.value = false
      return
    }

    // Ensure currentIndex is valid
    if (currentIndex.value >= visibleAnnouncements.value.length) {
      currentIndex.value = 0
    }

    startCarousel()
  } catch (error) {
    console.error('Failed to load announcements:', error)
    showBar.value = false
  } finally {
    isLoading.value = false
  }
}

// Watch for changes in visible announcements
watch(
  () => visibleAnnouncements.value.length,
  (newLength) => {
    if (newLength === 0) {
      showBar.value = false
    } else if (currentIndex.value >= newLength) {
      currentIndex.value = 0
    }
    startCarousel()
  }
)

onMounted(() => {
  loadAnnouncements()
})

onUnmounted(() => {
  if (carouselTimer) {
    clearInterval(carouselTimer)
  }
})
</script>

<style scoped>
.announcement-bar {
  @apply w-full py-2.5 px-4 text-sm;
}

.announcement-info {
  @apply bg-blue-50 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200;
}

.announcement-success {
  @apply bg-green-50 text-green-800 dark:bg-green-900/30 dark:text-green-200;
}

.announcement-warning {
  @apply bg-yellow-50 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-200;
}

.announcement-error {
  @apply bg-red-50 text-red-800 dark:bg-red-900/30 dark:text-red-200;
}

.announcement-container {
  @apply max-w-7xl mx-auto flex items-center justify-between gap-4;
}

.announcement-content {
  @apply flex items-center gap-3 flex-1 min-w-0;
}

.announcement-icon {
  @apply h-5 w-5 flex-shrink-0;
}

.announcement-text {
  @apply flex-1 min-w-0 truncate;
}

.announcement-title {
  @apply font-medium mr-1;
}

.announcement-body {
  @apply inline;
}

.announcement-body :deep(a) {
  @apply underline hover:no-underline;
}

.carousel-indicators {
  @apply flex items-center gap-1.5 ml-4;
}

.indicator-dot {
  @apply w-1.5 h-1.5 rounded-full bg-current opacity-40 transition-opacity;
}

.indicator-dot.active {
  @apply opacity-100;
}

.indicator-dot:hover {
  @apply opacity-70;
}

.close-button {
  @apply p-1 rounded-full hover:bg-black/10 dark:hover:bg-white/10 transition-colors flex-shrink-0;
}

/* Animations */
.slide-down-enter-active,
.slide-down-leave-active {
  @apply transition-all duration-300 ease-out;
}

.slide-down-enter-from,
.slide-down-leave-to {
  @apply opacity-0 -translate-y-full;
}

.fade-enter-active,
.fade-leave-active {
  @apply transition-opacity duration-300;
}

.fade-enter-from,
.fade-leave-to {
  @apply opacity-0;
}
</style>
