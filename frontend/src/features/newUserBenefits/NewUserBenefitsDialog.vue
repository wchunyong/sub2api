<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-[100] flex items-center justify-center overflow-y-auto bg-black/65 p-4" role="presentation">
      <section role="dialog" aria-modal="true" :aria-label="t('newUserBenefits.title')" class="relative w-full max-w-3xl rounded-3xl bg-white p-6 shadow-2xl dark:bg-dark-800 sm:p-8">
        <button v-if="remaining === 0" data-test="close-benefits" type="button" :aria-label="t('common.close')" class="absolute right-4 top-4 rounded-full p-2 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700" @click="emit('close')">✕</button>
        <span v-else data-test="close-countdown" class="absolute right-4 top-4 text-xs text-gray-500">{{ t('newUserBenefits.closeAfter', { seconds: remaining }) }}</span>
        <NewUserBenefitsContent @navigate="emit('close')" />
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import NewUserBenefitsContent from './NewUserBenefitsContent.vue'

const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const remaining = ref(3)
let timer: ReturnType<typeof setInterval> | undefined

onMounted(() => {
  timer = setInterval(() => {
    remaining.value = Math.max(0, remaining.value - 1)
    if (remaining.value === 0 && timer) clearInterval(timer)
  }, 1000)
})

onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>
