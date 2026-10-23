<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-7 text-gray-900 dark:text-gray-100">
      <header class="border-b border-gray-200 pb-8 dark:border-dark-700">
        <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ t('crazyMonday.kicker') }}</p>
        <h1 class="mt-3 text-4xl font-bold">{{ t('crazyMonday.title') }}</h1>
        <p class="mt-3 text-base leading-7 text-gray-600 dark:text-gray-300">{{ t('crazyMonday.description') }}</p>
      </header>
      <section class="rounded-2xl border border-red-200 bg-gradient-to-br from-red-50 via-white to-amber-50 p-6 shadow-sm dark:border-red-900/60 dark:from-red-950/30 dark:via-dark-800 dark:to-amber-950/20 sm:p-9">
        <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ active ? t('crazyMonday.endsIn') : t('crazyMonday.startsIn') }}</p>
        <p class="mt-3 font-mono text-4xl font-bold tabular-nums sm:text-5xl">{{ countdown }}</p>
        <p class="mt-5 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('crazyMonday.schedule') }}</p>
      </section>
      <section class="grid gap-5 md:grid-cols-2">
        <article class="card p-6">
          <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ t('crazyMonday.benefitLabel') }}</p>
          <h2 class="mt-3 text-2xl font-bold">{{ t('crazyMonday.benefit') }}</h2>
          <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ t('crazyMonday.benefitDetails') }}</p>
        </article>
        <article class="card p-6">
          <p class="text-sm font-semibold text-amber-700 dark:text-amber-300">{{ t('crazyMonday.eligibilityLabel') }}</p>
          <h2 class="mt-3 text-2xl font-bold">{{ t('crazyMonday.eligibility') }}</h2>
          <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ t('crazyMonday.eligibilityDetails') }}</p>
          <RouterLink to="/purchase" class="btn btn-primary mt-5 inline-flex">{{ t('crazyMonday.recharge') }}</RouterLink>
        </article>
      </section>
      <section class="card p-6 text-sm leading-7 text-gray-600 dark:text-gray-300">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('crazyMonday.details') }}</h2>
        <p class="mt-3">{{ t('crazyMonday.reason') }}</p>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { formatCountdown } from '@/features/activity/countdown'

const { t } = useI18n()
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined
const window = computed(() => {
  const local = new Date(now.value + 8 * 3600_000)
  const days = (8 - local.getUTCDay()) % 7
  const start = Date.UTC(local.getUTCFullYear(), local.getUTCMonth(), local.getUTCDate() + days) - 8 * 3600_000
  return { start, end: start + 24 * 3600_000 }
})
const active = computed(() => now.value >= window.value.start)
const countdown = computed(() => formatCountdown(Math.max(0, Math.ceil(((active.value ? window.value.end : window.value.start) - now.value) / 1000))))
onMounted(() => { timer = setInterval(() => { now.value = Date.now() }, 1000) })
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>
