<template>
  <div class="space-y-10">
    <section v-for="family in families" :key="family.platform" class="space-y-3">
      <div class="flex items-center gap-2.5">
        <span class="flex h-9 w-9 items-center justify-center rounded-xl bg-white text-xs font-bold text-primary-700 ring-1 ring-primary-200 dark:bg-dark-800 dark:text-primary-300 dark:ring-primary-500/20">{{ family.monogram }}</span>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ family.label }}</h2>
        <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.card.modelCount', { count: family.models.length }) }}</span>
        <span class="ml-2 h-px flex-1 bg-gray-200 dark:bg-dark-700"></span>
      </div>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <PlazaModelCard v-for="model in family.models" :key="model.name" :model="model" :rate-multiplier="0.3" planned />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PlazaModelCard from './PlazaModelCard.vue'
import type { PlazaDomesticFamily } from './domesticCatalog'

defineProps<{ families: PlazaDomesticFamily[] }>()
const { t } = useI18n()
</script>
