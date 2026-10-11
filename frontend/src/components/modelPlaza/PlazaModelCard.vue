<template>
  <article class="flex h-full flex-col rounded-2xl border border-gray-200/80 bg-white p-5 shadow-sm transition-shadow hover:shadow-card dark:border-dark-700 dark:bg-dark-800/70">
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <h3 class="break-all text-base font-semibold leading-6 text-gray-900 dark:text-white">{{ model.name }}</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ unitLabel }}</p>
      </div>
      <span class="shrink-0 rounded-lg bg-primary-50 px-2.5 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">
        {{ rateLabel }}
      </span>
    </div>

    <div class="mt-4 grid grid-cols-2 gap-x-3 gap-y-5 border-t border-gray-100 pt-4 sm:grid-cols-3 dark:border-dark-700">
      <div v-for="price in prices" :key="price.key" class="min-w-0">
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ price.label }}</p>
        <p class="mt-1 min-h-4 truncate font-mono text-xs text-gray-400 line-through dark:text-dark-500" :title="price.base">
          {{ price.base === '-' ? '\u00a0' : t('modelPlaza.card.originalPrice', { price: price.base }) }}
        </p>
        <p class="mt-1 break-all font-mono text-xl font-bold tracking-tight text-primary-700 dark:text-primary-300" :data-price="price.key">
          {{ price.paid }}
        </p>
        <p class="mt-0.5 text-[11px] text-gray-400 dark:text-dark-500">{{ t(planned ? 'modelPlaza.card.plannedPrice' : 'modelPlaza.card.currentPrice') }}</p>
      </div>
    </div>

    <p v-if="hasComplexPricing" class="mt-auto pt-4 text-xs text-gray-500 dark:text-dark-400">
      {{ t('modelPlaza.card.detailHint') }}
    </p>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PlazaModel } from '@/api/modelPlaza'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO } from '@/constants/channel'
import { formatScaled, resolveIntervalPrices } from '@/utils/pricing'

const props = defineProps<{
  model: PlazaModel
  rateMultiplier: number
  userRateMultiplier?: number | null
  imageRateIndependent?: boolean
  imageRateMultiplier?: number | null
  videoRateIndependent?: boolean
  videoRateMultiplier?: number | null
  planned?: boolean
}>()

const { t } = useI18n()
const mode = computed(() => props.model.pricing?.billing_mode || BILLING_MODE_TOKEN)
const effectiveRate = computed(() => props.userRateMultiplier ?? props.rateMultiplier)
const appliedRate = computed(() => {
  if (mode.value === BILLING_MODE_IMAGE && props.imageRateIndependent) return props.imageRateMultiplier ?? 1
  if (mode.value === BILLING_MODE_VIDEO && props.videoRateIndependent) return Math.max(0, props.videoRateMultiplier ?? 1)
  return effectiveRate.value
})
const rateLabel = computed(() => {
  const rate = appliedRate.value
  return rate > 0 && rate < 1
    ? t('modelPlaza.card.discount', { value: Number((rate * 10).toPrecision(8)) })
    : t('modelPlaza.card.multiplier', { value: rate })
})
const unitLabel = computed(() => mode.value === BILLING_MODE_TOKEN
  ? t('modelPlaza.card.tokenUnit')
  : mode.value === BILLING_MODE_IMAGE
    ? t('modelPlaza.table.perImage')
    : t('modelPlaza.table.perRequest'))
const scale = computed(() => mode.value === BILLING_MODE_TOKEN ? 1_000_000 : 1)
const firstTier = computed(() => [...(props.model.pricing?.intervals ?? [])].sort((a, b) => a.min_tokens - b.min_tokens)[0])
const resolvedTier = computed(() => firstTier.value && props.model.pricing
  ? resolveIntervalPrices(firstTier.value, props.model.pricing)
  : null)
function paid(value: number | null | undefined): string {
  return value == null ? '-' : formatScaled(value * appliedRate.value, scale.value, 2)
}

function base(value: number | null | undefined): string {
  return value == null ? '-' : formatScaled(value, scale.value, 2)
}

const prices = computed(() => mode.value === BILLING_MODE_TOKEN
  ? [
      { key: 'input', label: t('modelPlaza.table.input'), paid: paid(resolvedTier.value?.input_price ?? props.model.pricing?.input_price), base: base(resolvedTier.value?.input_price ?? props.model.pricing?.input_price) },
      { key: 'output', label: t('modelPlaza.table.output'), paid: paid(resolvedTier.value?.output_price ?? props.model.pricing?.output_price), base: base(resolvedTier.value?.output_price ?? props.model.pricing?.output_price) },
      { key: 'cache-read', label: t('modelPlaza.card.cacheRead'), paid: paid(resolvedTier.value?.cache_read_price ?? props.model.pricing?.cache_read_price), base: base(resolvedTier.value?.cache_read_price ?? props.model.pricing?.cache_read_price) },
      ...(resolvedTier.value?.cache_write_price != null || props.model.pricing?.cache_write_price != null
        ? [{ key: 'cache-write', label: t('modelPlaza.card.cacheWrite'), paid: paid(resolvedTier.value?.cache_write_price ?? props.model.pricing?.cache_write_price), base: base(resolvedTier.value?.cache_write_price ?? props.model.pricing?.cache_write_price) }]
        : [])
    ]
  : [{ key: 'request', label: mode.value === BILLING_MODE_IMAGE ? t('modelPlaza.table.perImage') : t('modelPlaza.table.perRequest'), paid: paid(firstTier.value?.per_request_price ?? props.model.pricing?.per_request_price), base: base(firstTier.value?.per_request_price ?? props.model.pricing?.per_request_price) }])

const hasComplexPricing = computed(() =>
  Boolean((props.model.pricing?.intervals?.length ?? 0) > 1 || (props.model.official_pricing?.intervals?.length ?? 0) > 1 ||
    props.model.time_pricing?.periods?.length || props.model.pricing?.reasoning_effort_multipliers)
)
</script>
