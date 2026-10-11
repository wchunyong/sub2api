<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-7 text-gray-900 dark:text-gray-100">
      <header class="border-b border-gray-200 pb-8 dark:border-dark-700">
        <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ t('tenYuanActivity.kicker') }}</p>
        <h1 class="mt-3 text-4xl font-bold">{{ t('tenYuanActivity.title') }}</h1>
        <p class="mt-3 text-base leading-7 text-gray-600 dark:text-gray-300">{{ t('tenYuanActivity.description') }}</p>
        <div v-if="state?.starts_at && state.ends_at" class="mt-5 flex flex-wrap gap-3 text-sm text-gray-500 dark:text-gray-400">
          <span>{{ t('tenYuanActivity.start') }} {{ formatBeijing(state.starts_at) }}</span>
          <span>—</span>
          <span>{{ t('tenYuanActivity.end') }} {{ formatBeijing(state.ends_at) }}</span>
        </div>
      </header>

      <div v-if="loading" class="card p-10 text-center">{{ t('common.loading') }}</div>
      <div v-else-if="error" role="alert" class="card border-red-200 p-6 text-red-700">{{ error }}</div>
      <div v-else-if="!state?.enabled" class="card p-8 text-center text-gray-500">{{ state?.status === 'closed' ? t(state.claimed ? 'tenYuanActivity.closedClaimed' : 'tenYuanActivity.closed') : state?.status === 'ended' ? t('tenYuanActivity.ended') : t('tenYuanActivity.unavailable') }}</div>
      <template v-else>
        <section class="rounded-2xl border border-red-200 bg-gradient-to-br from-red-50 via-white to-amber-50 p-6 shadow-sm dark:border-red-900/60 dark:from-red-950/30 dark:via-dark-800 dark:to-amber-950/20 sm:p-9">
          <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ liveStatus === 'scheduled' ? t('tenYuanActivity.startsIn') : t('tenYuanActivity.endsIn') }}</p>
          <p data-test="activity-countdown" class="mt-3 font-mono text-4xl font-bold tabular-nums sm:text-5xl">{{ countdownText }}</p>
          <p class="mt-5 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ state.claimed ? t('tenYuanActivity.claimed') : liveStatus === 'scheduled' ? t('tenYuanActivity.scheduled') : t('tenYuanActivity.active') }}</p>
          <RouterLink v-if="liveStatus === 'active' && !state.claimed" to="/purchase" class="btn btn-primary mt-6 inline-flex">{{ t('tenYuanActivity.recharge') }}</RouterLink>
        </section>

        <section class="grid gap-5 md:grid-cols-2">
          <article class="card p-6">
            <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ t('tenYuanActivity.benefitLabel') }}</p>
            <h2 class="mt-3 text-2xl font-bold">{{ state.daily_limit_usd == null ? t('tenYuanActivity.configuredBenefit') : `${state.daily_limit_usd} USD / ${t('tenYuanActivity.day')}` }}</h2>
            <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ state.daily_limit_usd == null ? t('tenYuanActivity.benefitVaries') : t('tenYuanActivity.benefit', { amount: state.daily_limit_usd }) }}</p>
          </article>
          <article class="card p-6">
            <p class="text-sm font-semibold text-amber-700 dark:text-amber-300">{{ t('tenYuanActivity.unlockLabel') }}</p>
            <h2 class="mt-3 text-2xl font-bold">{{ t('tenYuanActivity.threshold', { amount: state.min_recharge }) }}</h2>
            <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ t('tenYuanActivity.once') }}</p>
          </article>
        </section>

        <section class="card p-6 text-sm leading-7 text-gray-600 dark:text-gray-300">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('tenYuanActivity.details') }}</h2>
          <p class="mt-3">{{ t('tenYuanActivity.fixedEnd') }}</p>
          <p class="mt-2">{{ t('tenYuanActivity.partialDays') }}</p>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getTenYuanActivity, type TenYuanActivityState } from '@/api/activities'
import { countdownWithServerSkew, formatCountdown } from '@/features/activity/countdown'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const state = ref<TenYuanActivityState | null>(null)
const receivedAt = ref(Date.now())
const clock = ref(Date.now())
const loading = ref(true)
const error = ref('')
let clockTimer: ReturnType<typeof setInterval> | undefined
let refreshTimer: ReturnType<typeof setInterval> | undefined
let loadingRequest = false

function formatBeijing(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(new Date(value))
}

const liveStatus = computed(() => {
  const current = state.value
  if (!current?.enabled || !current.starts_at || !current.ends_at) return current?.status ?? 'off'
  if (countdownWithServerSkew(current.starts_at, current.server_now, receivedAt.value, clock.value) > 0) return 'scheduled'
  if (countdownWithServerSkew(current.ends_at, current.server_now, receivedAt.value, clock.value) > 0) return 'active'
  return 'ended'
})
const countdownSeconds = computed(() => {
  const current = state.value
  if (!current?.enabled || !current.starts_at || !current.ends_at) return 0
  const target = liveStatus.value === 'scheduled' ? current.starts_at : current.ends_at
  return countdownWithServerSkew(target, current.server_now, receivedAt.value, clock.value)
})
const countdownText = computed(() => formatCountdown(countdownSeconds.value))

async function load() {
  if (loadingRequest) return
  loadingRequest = true
  try {
    const requestStartedAt = Date.now()
    state.value = await getTenYuanActivity()
    const requestFinishedAt = Date.now()
    receivedAt.value = (requestStartedAt + requestFinishedAt) / 2
    clock.value = requestFinishedAt
    error.value = ''
  } catch (cause) {
    error.value = extractApiErrorMessage(cause, t('common.unknownError'))
  } finally { loading.value = false; loadingRequest = false }
}

function tick() {
  clock.value = Date.now()
  if (state.value?.enabled && state.value.status !== liveStatus.value) void load()
}
function refreshOnFocus() { if (!document.hidden) void load() }

onMounted(() => {
  void load()
  clockTimer = setInterval(tick, 1000)
  refreshTimer = setInterval(() => { void load() }, 30000)
  window.addEventListener('focus', refreshOnFocus)
  document.addEventListener('visibilitychange', refreshOnFocus)
})
onBeforeUnmount(() => {
  if (clockTimer) clearInterval(clockTimer)
  if (refreshTimer) clearInterval(refreshTimer)
  window.removeEventListener('focus', refreshOnFocus)
  document.removeEventListener('visibilitychange', refreshOnFocus)
})
</script>
