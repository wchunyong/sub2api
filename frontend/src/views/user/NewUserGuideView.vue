<template>
  <AppLayout>
    <NewUserGuideDialog
      v-if="ready"
      :video-url="guide.video_url"
      :pdf-url="guide.pdf_url"
      @close="closeGuide"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import NewUserGuideDialog from '@/features/newUserGuide/NewUserGuideDialog.vue'
import { getNewUserGuide } from '@/api/activities'
import type { NewUserGuideConfig } from '@/types/payment'

// Used only when the config request fails; the backend always returns effective links.
const FALLBACK_VIDEO_URL = 'https://www.bilibili.com/video/BV1XgeB6UEQg'
const FALLBACK_PDF_URL = 'https://pub-17119e84a4e34086ab498f52e1943e3a.r2.dev/tutorial/win.pdf'

const router = useRouter()
const ready = ref(false)
const guide = reactive<NewUserGuideConfig>({ video_url: FALLBACK_VIDEO_URL, pdf_url: FALLBACK_PDF_URL })

function closeGuide(): void {
  void router.push('/dashboard')
}

onMounted(async () => {
  try {
    const config = await getNewUserGuide()
    guide.video_url = config.video_url || FALLBACK_VIDEO_URL
    guide.pdf_url = config.pdf_url || FALLBACK_PDF_URL
  } catch {
    // Keep fallback links.
  } finally {
    ready.value = true
  }
})
</script>
