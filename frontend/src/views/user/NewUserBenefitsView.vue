<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl rounded-3xl bg-white p-6 shadow-sm dark:bg-dark-800 sm:p-8">
      <NewUserBenefitsContent v-if="active" />
      <p v-else class="text-center text-gray-600 dark:text-gray-300">{{ t('newUserBenefits.expired') }}</p>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import NewUserBenefitsContent from '@/features/newUserBenefits/NewUserBenefitsContent.vue'
import { isNewUserBenefitActive } from '@/features/newUserBenefits/eligibility'

const { t } = useI18n()
const authStore = useAuthStore()
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined
const active = computed(() => authStore.user?.role === 'user' && isNewUserBenefitActive(authStore.user?.created_at, now.value))
onMounted(() => { timer = setInterval(() => { now.value = Date.now() }, 1000) })
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>
