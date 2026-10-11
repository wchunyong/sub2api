<template>
  <div class="space-y-7">
    <section class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-primary-950 via-primary-900 to-primary-700 px-6 py-8 text-white shadow-lg shadow-primary-900/10 sm:px-9 sm:py-10 lg:px-12">
      <div class="absolute -right-16 -top-24 h-72 w-72 rounded-full bg-white/10 blur-3xl" aria-hidden="true"></div>
      <div class="relative grid items-center gap-8 lg:grid-cols-[minmax(260px,0.8fr)_minmax(0,1.2fr)]">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-primary-100/80">{{ t('modelPlaza.hero.eyebrow') }}</p>
          <h1 class="mt-4 text-3xl font-bold tracking-tight sm:text-4xl">{{ t('modelPlaza.hero.title') }}</h1>
          <p class="mt-3 max-w-xl text-sm leading-6 text-primary-50/85 sm:text-base">{{ t('modelPlaza.hero.subtitle') }}</p>
          <div v-if="!loading && response" class="mt-7 flex flex-wrap gap-6 border-t border-white/20 pt-5">
            <div><span data-testid="model-count" class="block text-2xl font-semibold">{{ totalModels }}</span><span class="text-xs text-primary-100/75">{{ t('modelPlaza.hero.models') }}</span></div>
            <div><span class="block text-2xl font-semibold">{{ visibleGroups.length }}</span><span class="text-xs text-primary-100/75">{{ t('modelPlaza.hero.groups') }}</span></div>
            <div><span class="block text-2xl font-semibold">USD</span><span class="text-xs text-primary-100/75">{{ t('modelPlaza.hero.unit') }}</span></div>
          </div>
        </div>
        <div class="rounded-3xl border border-white/25 bg-white/10 p-6 backdrop-blur-sm sm:p-8 lg:order-first">
          <p class="text-sm font-medium text-primary-100/85">{{ hasDomesticPromotion ? t('modelPlaza.hero.promoLabel') : t('modelPlaza.hero.plannedPromoLabel') }}</p>
          <p class="mt-2 text-5xl font-bold tracking-tight text-white sm:text-6xl">{{ t('modelPlaza.hero.discount') }}</p>
          <p class="mt-3 text-sm text-primary-50/80">{{ hasDomesticPromotion ? t('modelPlaza.hero.promoNote', { count: promoGroups.length }) : t('modelPlaza.hero.plannedPromoNote') }}</p>
        </div>
      </div>
    </section>

    <!-- 未登录提示 -->
    <p
      v-if="!isAuthenticated"
      class="flex items-center gap-1.5 text-xs text-gray-400 dark:text-dark-500"
    >
      <Icon name="infoCircle" size="xs" class="h-3.5 w-3.5" />
      {{ t('modelPlaza.anonymousHint') }}
    </p>

    <!-- 加载/错误/空 -->
    <div v-if="loading" class="flex min-h-[240px] items-center justify-center">
      <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-600/25 border-t-primary-600 dark:border-primary-400/25 dark:border-t-primary-400"></div>
    </div>
    <div
      v-else-if="error"
      class="rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
    >
      {{ t('modelPlaza.loadFailed') }}
    </div>
    <template v-else>
      <!-- 筛选区:平台 → 分组 → 倍率 -->
      <div class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800/60">
        <PlazaFilterBar
          :platforms="platforms"
          :groups="groupOptions"
          :rates="rates"
          :platform="selectedPlatform"
          :group-id="selectedGroupId"
          :rate="selectedRate"
          :search="searchQuery"
          @update:platform="selectedPlatform = $event"
          @update:group-id="selectedGroupId = $event"
          @update:rate="selectedRate = $event"
          @update:search="searchQuery = $event"
        />
      </div>

      <!-- 国产模型规划价格与已接入分组共用模型广场列表。 -->
      <div v-if="filteredGroups.length > 0 || filteredDomesticFamilies.length > 0" class="space-y-10">
        <PlazaDomesticCatalog v-if="filteredDomesticFamilies.length > 0" :families="filteredDomesticFamilies" />
        <PlazaGroupSection v-for="g in filteredGroups" :key="g.id" :group="g" />
      </div>
      <div
        v-else
        class="rounded-2xl border border-dashed border-gray-300 px-5 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
      >
        {{ searchActive ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlazaFilterBar from './PlazaFilterBar.vue'
import PlazaGroupSection from './PlazaGroupSection.vue'
import PlazaDomesticCatalog from './PlazaDomesticCatalog.vue'
import { domesticCatalogFamilies } from './domesticCatalog'
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'
import { getPlatformSpec } from '@/constants/platformCatalog'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
  error?: boolean
  /** 后台内嵌形态(AppLayout 内)。 */
  embedded?: boolean
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)
const hiddenDeepSeekModels = new Set(['deepseek-v4-flash', 'deepseek-v4-pro'])
const visibleGroups = computed(() => (props.response?.groups ?? [])
  .filter(g => !(g.platform === 'zhipu' && (g.name === '智谱-满血' || Math.abs(g.rate_multiplier - 0.6) < 0.000001)))
  .map(g => ({ ...g, models: g.models.filter(m => !(m.platform === 'deepseek' && hiddenDeepSeekModels.has(m.name.toLowerCase()))) }))
  .filter(g => g.models.length > 0))
const domesticGroups = computed(() => visibleGroups.value.filter(g =>
  g.models.some(m => getPlatformSpec(m.platform)?.cn_provider)
))
const promoGroups = computed(() => domesticGroups.value.filter(g =>
  Math.abs((g.user_rate_multiplier ?? g.rate_multiplier) - 0.3) < 0.000001
))
const hasDomesticPromotion = computed(() => domesticGroups.value.length > 0 &&
  promoGroups.value.length === domesticGroups.value.length)

const selectedPlatform = ref<string>('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref('')

const searchActive = computed(() => searchQuery.value.trim() !== '')
const filteredDomesticFamilies = computed(() => {
  if (selectedRate.value !== 'all' && selectedRate.value !== 0.3) return []
  const q = searchQuery.value.trim().toLowerCase()
  return domesticCatalogFamilies
    .map((family, index) => ({ family, id: -(index + 1) }))
    .filter(({ family, id }) =>
      (selectedPlatform.value === 'all' || selectedPlatform.value === family.platform) &&
      (selectedGroupId.value === 'all' || selectedGroupId.value === id)
    )
    .map(({ family }) => ({
      ...family,
      models: family.models.filter(model =>
        (!q || model.name.toLowerCase().includes(q))
      )
    }))
    .filter(family => family.models.length > 0)
})

/** 生效倍率 = 用户专属倍率 ?? 分组默认倍率。 */
function effectiveRate(g: ModelPlazaGroup): number {
  return g.user_rate_multiplier ?? g.rate_multiplier
}

const platforms = computed(() =>
  [...new Set([
    ...visibleGroups.value.map((g) => g.platform).filter(Boolean),
    ...domesticCatalogFamilies.map((family) => family.platform)
  ])].sort()
)

const groupOptions = computed(() =>
  [
    ...domesticCatalogFamilies.map((family, index) => ({
      id: -(index + 1),
      name: t('modelPlaza.domestic.groupName', { name: family.label }),
      platform: family.platform,
      rate: 0.3
    })),
    ...visibleGroups.value.map((g) => ({
      id: g.id,
      name: g.name,
      platform: g.platform,
      rate: effectiveRate(g)
    }))
  ]
)

/** 全量生效倍率;当前组合下不可用的项由 FilterBar 置灰而非隐藏。 */
const rates = computed(() =>
  [...new Set([0.3, ...visibleGroups.value.map(effectiveRate)])].sort((a, b) => a - b)
)

/** 数据刷新后选中的倍率可能不复存在,重置为全部。 */
watch(rates, (list) => {
  if (selectedRate.value !== 'all' && !list.includes(selectedRate.value)) {
    selectedRate.value = 'all'
  }
})

const filteredGroups = computed(() => {
  let groups = visibleGroups.value
  if (selectedPlatform.value !== 'all') {
    groups = groups.filter((g) => g.platform === selectedPlatform.value)
  }
  if (selectedGroupId.value !== 'all') {
    groups = groups.filter((g) => g.id === selectedGroupId.value)
  }
  if (selectedRate.value !== 'all') {
    groups = groups.filter((g) => effectiveRate(g) === selectedRate.value)
  }
  // 模型名搜索:分组内只留命中的模型,整组无命中则隐藏该分组。
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    groups = groups
      .map((g) => ({ ...g, models: g.models.filter((m) => m.name.toLowerCase().includes(q)) }))
      .filter((g) => g.models.length > 0)
  }
  // 专属倍率会改变生效值,不能只依赖后端按默认倍率的排序。
  return [...groups].sort(
    (a, b) => effectiveRate(a) - effectiveRate(b) || a.name.localeCompare(b.name)
  )
})

const totalModels = computed(() => new Set([
  ...filteredDomesticFamilies.value.flatMap(family => family.models.map(model => model.name.trim().toLowerCase())),
  ...filteredGroups.value.flatMap(group => group.models.map(model => model.name.trim().toLowerCase()))
]).size)
</script>
