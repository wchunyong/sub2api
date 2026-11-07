<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-[100] flex items-center justify-center overflow-y-auto bg-black/65 p-4" role="presentation" @click.self="emit('close')">
      <section role="dialog" aria-modal="true" :aria-label="t('newUserGuide.title')" class="relative w-full max-w-2xl rounded-3xl bg-white p-6 shadow-2xl dark:bg-dark-800 sm:p-8">
        <button type="button" data-test="close-guide" :aria-label="t('newUserGuide.close')" class="absolute right-4 top-4 rounded-full p-2 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700" @click="emit('close')">
          <Icon name="x" size="sm" />
        </button>

        <header>
          <p class="text-sm font-semibold tracking-wide text-orange-600 dark:text-orange-400">{{ t('newUserGuide.kicker') }}</p>
          <h1 class="mt-2 text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">{{ t('newUserGuide.title') }}</h1>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('newUserGuide.subtitle') }}</p>
        </header>

        <div class="mt-6 grid gap-4 sm:grid-cols-2">
          <article class="flex flex-col rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-dark-600 dark:bg-dark-700/50" data-test="guide-video-card">
            <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-orange-100 text-orange-600 dark:bg-orange-900/40 dark:text-orange-300">
              <Icon name="play" size="sm" />
            </span>
            <h2 class="mt-3 text-lg font-bold text-gray-900 dark:text-white">{{ t('newUserGuide.videoTitle') }}</h2>
            <p class="mt-2 flex-1 text-sm text-gray-600 dark:text-gray-300">{{ t('newUserGuide.videoDescription') }}</p>
            <a :href="videoUrl" target="_blank" rel="noopener noreferrer" data-test="guide-video-link" class="mt-4 inline-flex w-fit rounded-lg bg-orange-500 px-4 py-2 text-sm font-semibold text-white hover:bg-orange-600">
              {{ t('newUserGuide.watchVideo') }}
            </a>
          </article>

          <article class="flex flex-col rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-dark-600 dark:bg-dark-700/50" data-test="guide-doc-card">
            <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300">
              <Icon name="document" size="sm" />
            </span>
            <h2 class="mt-3 text-lg font-bold text-gray-900 dark:text-white">{{ t('newUserGuide.docTitle') }}</h2>
            <p class="mt-2 flex-1 text-sm text-gray-600 dark:text-gray-300">{{ t('newUserGuide.docDescription') }}</p>
            <a :href="pdfUrl" target="_blank" rel="noopener noreferrer" data-test="guide-doc-link" class="mt-4 inline-flex w-fit rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700">
              {{ t('newUserGuide.readDoc') }}
            </a>
          </article>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-green-200 bg-green-50 px-5 py-4 dark:border-green-900 dark:bg-green-950/30">
          <div class="text-sm text-gray-700 dark:text-gray-200">
            <p class="font-semibold">{{ t('newUserGuide.helpTitle') }}</p>
            <p class="mt-1">
              {{ t('newUserGuide.wechatLabel') }}
              <span class="ml-1 select-all rounded bg-white px-2 py-0.5 font-mono font-semibold dark:bg-dark-800">lianjieai_top</span>
            </p>
          </div>
          <button type="button" data-test="copy-wechat" class="rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white hover:bg-green-700" @click="copyWechat">
            {{ copied ? t('newUserGuide.copied') : t('newUserGuide.copyWechat') }}
          </button>
        </div>

        <p class="mt-5 text-center text-xs text-gray-500 dark:text-gray-400">{{ t('newUserGuide.footer') }}</p>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{ videoUrl: string; pdfUrl: string }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const copied = ref(false)

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

async function copyWechat(): Promise<void> {
  try {
    await navigator.clipboard.writeText('lianjieai_top')
    copied.value = true
  } catch {
    // The visible ID remains selectable when clipboard access is unavailable.
  }
}
</script>
