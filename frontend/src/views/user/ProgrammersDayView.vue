<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-7 text-gray-900 dark:text-gray-100">
      <header class="border-b border-gray-200 pb-8 dark:border-dark-700">
        <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ t('programmersDay.kicker') }}</p>
        <h1 class="mt-3 text-4xl font-bold">{{ t('programmersDay.title') }}</h1>
        <p class="mt-3 text-base leading-7 text-gray-600 dark:text-gray-300">{{ t('programmersDay.description', { percent: state?.bonus_percent ?? 24 }) }}</p>
        <p v-if="state" class="mt-5 text-sm text-gray-500 dark:text-gray-400">{{ formatBeijing(state.starts_at) }} — {{ formatBeijing(state.ends_at) }}</p>
      </header>
      <div v-if="loading" class="card p-10 text-center">{{ t('common.loading') }}</div>
      <div v-else-if="error" role="alert" class="card border-red-200 p-6 text-red-700">{{ error }}</div>
      <template v-else>
        <section class="rounded-2xl border border-red-200 bg-gradient-to-br from-red-50 via-white to-amber-50 p-6 shadow-sm dark:border-red-900/60 dark:from-red-950/30 dark:via-dark-800 dark:to-amber-950/20 sm:p-9">
          <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ !state?.enabled ? t('programmersDay.disabled') : liveActive ? t('programmersDay.endsIn') : upcoming ? t('programmersDay.startsIn') : t('programmersDay.ended') }}</p>
          <p v-if="liveActive || upcoming" class="mt-3 font-mono text-4xl font-bold tabular-nums sm:text-5xl">{{ countdown }}</p>
          <p class="mt-5 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ !state?.enabled ? t('programmersDay.disabled') : liveActive ? t('programmersDay.rule', { percent: state.bonus_percent }) : upcoming ? t('programmersDay.scheduled', { percent: state.bonus_percent }) : t('programmersDay.ended') }}</p>
          <RouterLink v-if="liveActive" to="/purchase" class="btn btn-primary mt-6 inline-flex">{{ t('programmersDay.recharge') }}</RouterLink>
          <div v-else-if="upcoming" class="mt-6 flex flex-wrap items-center gap-3">
            <button type="button" class="btn btn-primary cursor-not-allowed opacity-60" disabled>{{ t('programmersDay.recharge') }}</button>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('programmersDay.rechargeUpcoming') }}</span>
          </div>
        </section>
        <section class="grid gap-5 md:grid-cols-2">
          <article class="card p-6">
            <p class="text-sm font-semibold text-red-600 dark:text-red-300">{{ t('programmersDay.benefitLabel') }}</p>
            <h2 class="mt-3 text-2xl font-bold">{{ t('programmersDay.example', { percent: state?.bonus_percent ?? 24, total: 100 + (state?.bonus_percent ?? 24) }) }}</h2>
            <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ t('programmersDay.benefitDetails') }}</p>
          </article>
          <article class="card p-6">
            <p class="text-sm font-semibold text-amber-700 dark:text-amber-300">{{ t('programmersDay.timeLabel') }}</p>
            <h2 class="mt-3 text-2xl font-bold">{{ t('programmersDay.duration') }}</h2>
            <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ t('programmersDay.timeDetails') }}</p>
          </article>
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
import { getProgrammersDay, type ProgrammersDayState } from '@/api/activities'
import { formatCountdown } from '@/features/activity/countdown'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const state = ref<ProgrammersDayState | null>(null)
const now = ref(Date.now())
const clockOffset = ref(0)
const loading = ref(true)
const error = ref('')
let tickTimer: ReturnType<typeof setInterval> | undefined
let refreshTimer: ReturnType<typeof setInterval> | undefined
const serverTime = computed(() => now.value + clockOffset.value)
const upcoming = computed(() => state.value?.enabled === true && serverTime.value < new Date(state.value.starts_at).getTime())
const liveActive = computed(() => state.value?.enabled === true && serverTime.value >= new Date(state.value.starts_at).getTime() && serverTime.value < new Date(state.value.ends_at).getTime())
const countdown = computed(() => formatCountdown(Math.max(0, Math.ceil(((liveActive.value ? new Date(state.value?.ends_at ?? 0).getTime() : new Date(state.value?.starts_at ?? 0).getTime()) - serverTime.value) / 1000))))
function formatBeijing(value: string) {
  return new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}
async function load() {
  try { state.value = await getProgrammersDay(); clockOffset.value = new Date(state.value.server_now).getTime() - Date.now(); error.value = '' }
  catch (cause) { error.value = extractApiErrorMessage(cause, t('common.unknownError')) }
  finally { loading.value = false }
}
onMounted(() => {
  void load()
  tickTimer = setInterval(() => { now.value = Date.now() }, 1000)
  refreshTimer = setInterval(() => { void load() }, 30000)
})
onBeforeUnmount(() => { if (tickTimer) clearInterval(tickTimer); if (refreshTimer) clearInterval(refreshTimer) })
</script>
