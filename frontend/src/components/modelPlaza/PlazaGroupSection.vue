<template>
  <section class="space-y-5" :data-group-id="group.id">
    <div v-for="(family, index) in modelFamilies" :key="family.platform" class="space-y-3">
      <div class="flex flex-wrap items-center gap-2.5">
        <span class="flex h-9 w-9 items-center justify-center rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
          <PlatformIcon :platform="family.platform as GroupPlatform" size="sm" />
        </span>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ platformLabel(family.platform) }}</h2>
        <span class="text-xs text-gray-400 dark:text-dark-500">{{ t('modelPlaza.card.modelCount', { count: family.models.length }) }}</span>
        <GroupBadge
          v-if="index === 0"
          :name="group.name"
          :platform="group.platform as GroupPlatform"
          :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
          :rate-multiplier="group.rate_multiplier"
          :user-rate-multiplier="group.user_rate_multiplier ?? null"
          :peak-rate-enabled="group.peak_rate_enabled"
          :peak-start="group.peak_start"
          :peak-end="group.peak_end"
          :peak-rate-multiplier="group.peak_rate_multiplier"
          always-show-rate
        />
        <span
          v-if="index === 0 && group.is_exclusive"
          class="inline-flex items-center gap-1 rounded-md bg-purple-50 px-2 py-0.5 text-xs font-medium text-purple-600 dark:bg-purple-900/20 dark:text-purple-400"
        >
          <Icon name="shield" size="xs" class="h-3 w-3" />
          {{ t('modelPlaza.badges.exclusive') }}
        </span>
        <span
          v-if="index === 0 && group.subscription_type === 'subscription'"
          class="inline-flex items-center rounded-md bg-violet-50 px-2 py-0.5 text-xs font-medium text-violet-600 dark:bg-violet-900/20 dark:text-violet-400"
        >
          {{ t('modelPlaza.badges.subscription') }}
        </span>
        <span class="ml-2 h-px min-w-8 flex-1 bg-gray-200 dark:bg-dark-700"></span>
      </div>
      <p v-if="index === 0 && group.description" class="text-sm text-gray-500 dark:text-dark-400">{{ group.description }}</p>
      <p v-if="index === 0 && peakNote" class="inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
        <Icon name="clock" size="xs" class="h-3 w-3" />
        {{ peakNote }}
      </p>
      <p v-if="index === 0 && longContextNote" class="flex items-center gap-1 text-xs text-gray-500 dark:text-dark-400">
        <Icon name="infoCircle" size="xs" class="h-3 w-3" />
        {{ longContextNote }}
      </p>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <PlazaModelCard
          v-for="model in family.models"
          :key="`${model.platform}:${model.name}`"
          :model="model"
          :rate-multiplier="group.rate_multiplier"
          :user-rate-multiplier="group.user_rate_multiplier ?? null"
          :image-rate-independent="group.image_rate_independent"
          :image-rate-multiplier="group.image_rate_multiplier"
          :video-rate-independent="group.video_rate_independent"
          :video-rate-multiplier="group.video_rate_multiplier"
        />
      </div>
    </div>

    <p v-if="group.models.length === 0" class="rounded-2xl border border-dashed border-gray-200 px-5 py-6 text-center text-sm text-gray-400 dark:border-dark-700 dark:text-dark-500">{{ t('modelPlaza.detail.noModels') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import PlazaModelCard from './PlazaModelCard.vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformLabel } from '@/utils/platformColors'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { useAppStore } from '@/stores/app'

const props = defineProps<{
  group: ModelPlazaGroup
}>()

const { t } = useI18n()
const appStore = useAppStore()
const modelFamilies = computed(() => {
  const families = new Map<string, PlazaModel[]>()
  for (const model of props.group.models) {
    const platform = model.platform || props.group.platform
    if (!families.has(platform)) families.set(platform, [])
    families.get(platform)!.push(model)
  }
  return [...families].map(([platform, models]) => ({ platform, models }))
})

/** 高峰窗口描述(含倍率与服务器时区标注);分组未启用高峰为空串。 */
const peakWindow = computed(() => {
  if (!hasPeakRate(props.group)) return ''
  return formatPeakRateWindow(
    props.group,
    serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)
  )
})

const peakNote = computed(() => {
  if (!peakWindow.value) return ''
  return t('modelPlaza.detail.peakNote', {
    window: peakWindow.value,
    multiplier: props.group.peak_rate_multiplier
  })
})

/**
 * 分组关闭了长上下文阶梯、但组内有模型官方带阶梯时提示:实付列只展示基础档,
 * 官方阶梯仅供参考。字段缺失(旧后端)不提示。
 */
const longContextNote = computed(() => {
  if (props.group.long_context_pricing_enabled !== false) return ''
  const hasOfficialLadder = props.group.models.some(
    (m) => (m.official_pricing?.intervals?.length ?? 0) > 1
  )
  return hasOfficialLadder ? t('modelPlaza.detail.longContextDisabledNote') : ''
})
</script>
