<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- Background Decoration -->
    <div class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar />
    <NewUserBenefitsDialog v-if="showNewUserBenefits" @close="dismissNewUserBenefits" />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="p-4 md:p-6 lg:p-8">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import NewUserBenefitsDialog from '@/features/newUserBenefits/NewUserBenefitsDialog.vue'
import { isNewUserBenefitActive, newUserBenefitDismissalKey } from '@/features/newUserBenefits/eligibility'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')
const benefitNow = ref(Date.now())
const showNewUserBenefits = ref(false)
let benefitTimer: ReturnType<typeof setInterval> | undefined

watch([() => authStore.user?.id, () => authStore.user?.created_at, benefitNow], () => {
  const user = authStore.user
  if (!user || user.role !== 'user' || !isNewUserBenefitActive(user.created_at, benefitNow.value)) {
    showNewUserBenefits.value = false
    return
  }
  showNewUserBenefits.value = localStorage.getItem(newUserBenefitDismissalKey(user.id)) !== '1'
}, { immediate: true })

function dismissNewUserBenefits(): void {
  const userId = authStore.user?.id
  if (userId) localStorage.setItem(newUserBenefitDismissalKey(userId), '1')
  showNewUserBenefits.value = false
}

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
  benefitTimer = setInterval(() => { benefitNow.value = Date.now() }, 1000)
})
onBeforeUnmount(() => { if (benefitTimer) clearInterval(benefitTimer) })

defineExpose({ replayTour })
</script>
