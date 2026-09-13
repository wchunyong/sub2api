<template>
  <Transition name="announcement-banner-slide">
    <div
      ref="bannerEl"
      v-if="banner"
      class="relative z-[50] bg-gradient-to-r from-violet-600 via-rose-500 to-amber-400 px-4 py-3 text-white shadow-lg"
      data-testid="announcement-banner"
      :class="{ 'cursor-pointer': canClickWholeBanner }"
      role="region"
      :aria-label="banner.title"
      @click="handleBannerClick"
    >
      <div class="mx-auto flex max-w-7xl items-center justify-center gap-3 pr-9 text-center">
        <div
          class="announcement-banner-content min-w-0 text-sm font-medium leading-6 sm:text-base"
          v-html="renderedContent"
        ></div>

        <button
          v-if="showButton"
          type="button"
          class="shrink-0 rounded-md bg-white px-3 py-1.5 text-sm font-medium text-blue-700 shadow-sm transition hover:bg-blue-50 focus:outline-none focus:ring-2 focus:ring-white/70"
          @click.stop="handleButtonClick"
        >
          {{ buttonText }}
        </button>
      </div>

      <button
        type="button"
        data-testid="announcement-banner-dismiss"
        class="absolute right-3 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-white/90 transition hover:bg-white/15 hover:text-white focus:outline-none focus:ring-2 focus:ring-white/70"
        :aria-label="t('common.close')"
        @click.stop="dismiss"
      >
        <Icon name="x" size="sm" />
      </button>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const authStore = useAuthStore()
const { announcements } = storeToRefs(announcementStore)
const locallyDismissedIds = ref(new Set<number>())
const bannerEl = ref<HTMLElement | null>(null)
let resizeObserver: ResizeObserver | null = null

const banner = computed(() =>
  announcements.value.find((item) =>
    item.notify_mode === 'banner' &&
    !item.read_at &&
    !locallyDismissedIds.value.has(item.id)
  ) ?? null
)

const config = computed(() => banner.value?.banner_config ?? {})
const canClickWholeBanner = computed(() =>
  Boolean(config.value.whole_banner_click_enabled && safeURL(config.value.click_url))
)
const showButton = computed(() =>
  Boolean(config.value.button_enabled && buttonText.value && safeURL(config.value.button_url))
)
const buttonText = computed(() => (config.value.button_text ?? '').trim())

const renderedContent = computed(() => {
  const content = banner.value?.content || banner.value?.title || ''
  const html = marked.parseInline(content) as string
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['a', 'strong', 'em', 'b', 'i', 'u', 'span', 'code'],
    ALLOWED_ATTR: ['href', 'target', 'rel'],
  })
})

function safeURL(raw?: string): string {
  const trimmed = (raw ?? '').trim()
  if (!trimmed) return ''

  try {
    const parsed = new URL(trimmed, window.location.origin)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return ''
    }
    return parsed.toString()
  } catch {
    return ''
  }
}

function openURL(raw?: string) {
  const url = safeURL(raw)
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}

async function dismiss() {
  if (!banner.value) return
  if (!authStore.isAuthenticated) {
    locallyDismissedIds.value = new Set(locallyDismissedIds.value).add(banner.value.id)
    return
  }
  await announcementStore.markAsRead(banner.value.id)
}

function clearBannerOffset() {
  resizeObserver?.disconnect()
  resizeObserver = null
  document.documentElement.classList.remove('has-announcement-banner')
  document.documentElement.style.removeProperty('--announcement-banner-height')
}

function syncBannerOffset() {
  const height = bannerEl.value?.getBoundingClientRect().height || bannerEl.value?.offsetHeight || 48
  document.documentElement.classList.add('has-announcement-banner')
  document.documentElement.style.setProperty('--announcement-banner-height', `${height}px`)
}

async function bindBannerOffset() {
  await nextTick()
  if (!banner.value || !bannerEl.value) {
    clearBannerOffset()
    return
  }

  syncBannerOffset()
  resizeObserver?.disconnect()
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(syncBannerOffset)
    resizeObserver.observe(bannerEl.value)
  }
}

watch(banner, () => {
  void bindBannerOffset()
}, { immediate: true })

onBeforeUnmount(clearBannerOffset)

async function handleBannerClick() {
  if (!banner.value || !canClickWholeBanner.value) return
  openURL(config.value.click_url)
  await dismiss()
}

async function handleButtonClick() {
  if (!banner.value || !showButton.value) return
  openURL(config.value.button_url)
  await dismiss()
}
</script>

<style scoped>
.announcement-banner-slide-enter-active,
.announcement-banner-slide-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.announcement-banner-slide-enter-from,
.announcement-banner-slide-leave-to {
  opacity: 0;
  transform: translateY(-100%);
}

:deep(.announcement-banner-content a) {
  color: inherit;
  text-decoration: underline;
  text-underline-offset: 3px;
}

:deep(.announcement-banner-content p) {
  margin: 0;
}
</style>
