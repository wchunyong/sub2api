<template>
  <BaseDialog :show="show" :title="t('nav.contactUs')" width="narrow" @close="emit('close')">
    <p class="mb-5 text-sm leading-6 text-gray-600 dark:text-dark-300">
      {{ t('nav.contactSupportHint') }}
    </p>
    <div class="space-y-3 rounded-xl bg-gray-50 p-4 text-sm text-gray-800 dark:bg-dark-800 dark:text-dark-100">
      <p>TG: <a href="https://t.me/sfv_123" target="_blank" rel="noopener noreferrer" class="break-all text-primary-600 hover:underline dark:text-primary-400">https://t.me/sfv_123</a></p>
      <p>QQ: 3505152117</p>
      <p>WX: lianjieai_top</p>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'

defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()
</script>
