<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <header>
        <p class="text-sm font-semibold text-amber-600 dark:text-amber-400">{{ t('admin.activity.kicker') }}</p>
        <h1 class="mt-2 text-3xl font-bold text-gray-900 dark:text-white">{{ t('admin.activity.title') }}</h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.activity.description') }}</p>
      </header>

      <div v-if="error" role="alert" class="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300">{{ error }}</div>
      <div v-if="loading" class="card p-8 text-center text-gray-500">{{ t('common.loading') }}</div>
      <template v-else>
        <section class="card p-6" data-test="programmers-day-settings">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.activity.programmersDayTitle') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.activity.programmersDayRule') }}</p>
            </div>
            <span class="rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">{{ programmersDayStatus }}</span>
          </div>
          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="flex items-center gap-3 text-sm font-medium text-gray-700 dark:text-gray-200">
              <input v-model="programmersDayEnabled" type="checkbox" data-test="programmers-day-enabled" />
              {{ t('admin.activity.programmersDayEnabled') }}
            </label>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('admin.activity.programmersDayPercent') }}
              <input v-model.number="programmersDayPercent" type="number" min="0" max="1000" step="0.01" class="input mt-2 w-full" data-test="programmers-day-percent" />
            </label>
          </div>
          <p v-if="programmersDay" class="mt-4 text-sm text-gray-500 dark:text-gray-400">{{ formatBeijing(programmersDay.starts_at) }} — {{ formatBeijing(programmersDay.ends_at) }}</p>
          <button class="btn btn-primary mt-5" :disabled="savingProgrammersDay || !Number.isFinite(programmersDayPercent) || programmersDayPercent < 0 || programmersDayPercent > 1000" @click="saveProgrammersDay">{{ savingProgrammersDay ? t('common.saving') : t('admin.activity.save') }}</button>
        </section>
        <section class="card p-6">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.activity.campaignName') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.activity.rule') }}</p>
            </div>
            <span class="rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">{{ statusLabel }}</span>
          </div>
          <div v-if="state?.current" class="mt-6 grid gap-4 rounded-xl bg-gray-50 p-4 text-sm dark:bg-dark-700/50 sm:grid-cols-3">
            <div><p class="text-gray-500">{{ t('admin.activity.start') }}</p><p class="mt-1 font-semibold">{{ formatBeijing(state.current.starts_at) }}</p></div>
            <div><p class="text-gray-500">{{ t('admin.activity.end') }}</p><p class="mt-1 font-semibold">{{ formatBeijing(state.current.ends_at) }}</p></div>
            <div><p class="text-gray-500">{{ t('admin.activity.group') }}</p><p class="mt-1 font-semibold">{{ selectedGroup?.name ?? state.current.group_id }}</p><p class="mt-1 text-xs text-gray-500">{{ t('admin.activity.dailyLimit') }}: {{ selectedGroup?.daily_limit_usd == null ? '—' : `$${selectedGroup.daily_limit_usd}` }}</p></div>
          </div>
          <p v-if="state?.current && state.current.status === 'active'" class="mt-4 text-sm text-amber-700 dark:text-amber-300">{{ t('admin.activity.activeLocked') }}</p>
          <p v-if="state?.current && state.current.status === 'closed'" class="mt-4 text-sm text-gray-500">{{ t('admin.activity.closedNote') }}</p>
        </section>

        <section v-if="canSchedule" class="card p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ state?.current?.status === 'scheduled' ? t('admin.activity.edit') : t('admin.activity.schedule') }}</h2>
          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('admin.activity.group') }}
              <select v-model.number="groupId" class="input mt-2 w-full" data-test="activity-group">
                <option :value="0">{{ t('admin.activity.selectGroup') }}</option>
                <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}{{ group.daily_limit_usd != null ? ` · $${group.daily_limit_usd}/${t('admin.activity.day')}` : '' }}</option>
              </select>
              <span v-if="!state?.default_group_id" class="mt-2 block text-xs text-amber-700">{{ t('admin.activity.missingDefault') }}</span>
            </label>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('admin.activity.startBeijing') }}
              <input v-model="startsAtLocal" type="datetime-local" step="1" class="input mt-2 w-full" data-test="activity-start" />
            </label>
          </div>
          <p class="mt-4 text-sm text-gray-500">{{ t('admin.activity.derivedEnd') }}: {{ derivedEnd }}</p>
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.activity.partialDays') }}</p>
          <button class="btn btn-primary mt-5" data-test="ten-yuan-save" :disabled="saving || !groupId || !startsAtLocal" @click="save">{{ saving ? t('common.saving') : t('admin.activity.save') }}</button>
        </section>

        <section v-if="state?.current && (state.current.status === 'scheduled' || state.current.status === 'active')" class="card border-amber-200 p-6 dark:border-amber-900">
          <h2 class="text-lg font-semibold">{{ t('admin.activity.close') }}</h2>
          <p class="mt-2 text-sm text-gray-500">{{ t('admin.activity.closeWarning') }}</p>
          <button class="btn btn-secondary mt-4" :disabled="saving" @click="close">{{ t('admin.activity.close') }}</button>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getAll } from '@/api/admin/groups'
import { closeTenYuanActivity, getTenYuanActivityAdmin, scheduleTenYuanActivity, getProgrammersDayAdmin, updateProgrammersDay, type TenYuanActivityAdminState } from '@/api/admin/activities'
import type { ProgrammersDayState } from '@/api/activities'
import type { AdminGroup } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const state = ref<TenYuanActivityAdminState | null>(null)
const programmersDay = ref<ProgrammersDayState | null>(null)
const programmersDayEnabled = ref(true)
const programmersDayPercent = ref(24)
const savingProgrammersDay = ref(false)
const groups = ref<AdminGroup[]>([])
const groupId = ref(0)
const startsAtLocal = ref('')
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const selectedGroup = computed(() => groups.value.find(group => group.id === state.value?.current?.group_id))
const canSchedule = computed(() => !state.value?.current || state.value.current.status === 'scheduled' || state.value.current.status === 'ended')
const statusLabel = computed(() => t(`admin.activity.status.${state.value?.current?.status ?? 'off'}`))
const programmersDayStatus = computed(() => {
  if (!programmersDay.value?.enabled) return t('admin.activity.status.off')
  if (programmersDay.value.active) return t('admin.activity.status.active')
  return Date.now() >= new Date(programmersDay.value.ends_at).getTime() ? t('admin.activity.status.ended') : t('admin.activity.status.scheduled')
})

function formatBeijing(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(new Date(value))
}

function beijingInput(value: string): string {
  return new Date(value).toLocaleString('sv-SE', { timeZone: 'Asia/Shanghai', hour12: false }).replace(' ', 'T')
}

const derivedEnd = computed(() => {
  if (!startsAtLocal.value) return '—'
  const start = new Date(`${startsAtLocal.value}+08:00`)
  return Number.isNaN(start.getTime()) ? '—' : formatBeijing(new Date(start.getTime() + 168 * 60 * 60 * 1000).toISOString())
})

async function load() {
  error.value = ''
  loading.value = true
  try {
    const [activity, allGroups, festival] = await Promise.all([getTenYuanActivityAdmin(), getAll(), getProgrammersDayAdmin()])
    state.value = activity
    programmersDay.value = festival
    programmersDayEnabled.value = festival.enabled
    programmersDayPercent.value = festival.bonus_percent
    groups.value = allGroups.filter(group => group.subscription_type === 'subscription' && group.status === 'active')
    groupId.value = activity.current?.status === 'scheduled' ? activity.current.group_id : activity.default_group_id ?? 0
    startsAtLocal.value = activity.current?.status === 'scheduled' ? beijingInput(activity.current.starts_at) : beijingInput(new Date(Date.now() + 60 * 60 * 1000).toISOString())
  } catch (cause) {
    error.value = extractApiErrorMessage(cause, t('common.unknownError'))
  } finally {
    loading.value = false
  }
}

async function saveProgrammersDay() {
  savingProgrammersDay.value = true
  error.value = ''
  try {
    programmersDay.value = await updateProgrammersDay(programmersDayEnabled.value, programmersDayPercent.value)
  } catch (cause) {
    error.value = extractApiErrorMessage(cause, t('common.unknownError'))
  } finally { savingProgrammersDay.value = false }
}

async function save() {
  if (!groupId.value || !startsAtLocal.value) return
  saving.value = true
  error.value = ''
  try {
    const secondPreciseStart = startsAtLocal.value.length === 16 ? `${startsAtLocal.value}:00` : startsAtLocal.value.slice(0, 19)
    await scheduleTenYuanActivity(groupId.value, `${secondPreciseStart}+08:00`)
    await load()
  } catch (cause) {
    error.value = extractApiErrorMessage(cause, t('common.unknownError'))
  } finally { saving.value = false }
}

async function close() {
  const runId = state.value?.current?.id
  if (!runId || !window.confirm(t('admin.activity.closeWarning'))) return
  saving.value = true
  error.value = ''
  try {
    await closeTenYuanActivity(runId)
    await load()
  } catch (cause) {
    error.value = extractApiErrorMessage(cause, t('common.unknownError'))
  } finally { saving.value = false }
}

onMounted(() => { void load() })
</script>
