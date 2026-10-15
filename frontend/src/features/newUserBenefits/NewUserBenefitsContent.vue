<template>
  <div class="text-gray-900 dark:text-gray-100">
    <p class="text-sm font-semibold tracking-wide text-amber-600 dark:text-amber-300">{{ t('newUserBenefits.kicker') }}</p>
    <h1 class="mt-2 text-2xl font-bold sm:text-3xl">{{ t('newUserBenefits.title') }}</h1>
    <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">{{ t('newUserBenefits.description') }}</p>

    <div class="mt-6 grid gap-4 sm:grid-cols-2">
      <article class="rounded-2xl border border-green-200 bg-green-50 p-5 dark:border-green-900 dark:bg-green-950/30">
        <span class="text-3xl" aria-hidden="true">💬</span>
        <h2 class="mt-3 text-xl font-bold">{{ t('newUserBenefits.wechatTitle') }}</h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('newUserBenefits.wechatDescription') }}</p>
        <p class="mt-4 select-all rounded-lg bg-white px-3 py-2 font-mono font-semibold dark:bg-dark-800">lianjieai_top</p>
        <button type="button" class="mt-4 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white hover:bg-green-700" @click="copyWechat">
          {{ copied ? t('newUserBenefits.copied') : t('newUserBenefits.copyWechat') }}
        </button>
      </article>

      <article class="rounded-2xl border border-amber-200 bg-amber-50 p-5 dark:border-amber-900 dark:bg-amber-950/30">
        <span class="text-3xl" aria-hidden="true">🎁</span>
        <h2 class="mt-3 text-xl font-bold">{{ t('newUserBenefits.rechargeTitle') }}</h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('newUserBenefits.rechargeDescription') }}</p>
        <RouterLink to="/purchase" class="mt-4 inline-flex rounded-lg bg-amber-500 px-4 py-2 text-sm font-semibold text-white hover:bg-amber-600" @click="emit('navigate')">
          {{ t('newUserBenefits.rechargeAction') }}
        </RouterLink>
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const copied = ref(false)
const emit = defineEmits<{ navigate: [] }>()

async function copyWechat(): Promise<void> {
  try {
    await navigator.clipboard.writeText('lianjieai_top')
    copied.value = true
  } catch {
    // The visible ID remains selectable in browsers without clipboard access.
  }
}
</script>
