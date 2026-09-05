<template>
  <BaseDialog :show="show" :title="t('keys.quickImport.title')" width="wide" @close="emit('close')">
    <div class="space-y-4">
      <p class="text-sm text-gray-500">{{ keyName }} · {{ apiKey ? `${apiKey.slice(0, 6)}…${apiKey.slice(-4)}` : '' }}</p>
      <template v-if="!agent">
        <p class="font-medium">{{ t('keys.quickImport.selectAgent') }}</p>
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <button v-for="item in importAgents" :key="item.id" :data-testid="`agent-${item.id}`" class="rounded-xl border border-gray-200 p-4 text-left hover:border-primary-500 dark:border-dark-600" @click="agent = item.id">
            <span class="font-medium">{{ item.name }}</span>
            <span v-if="!supportsAgent(item.id, platform, allowMessagesDispatch)" class="mt-2 block text-xs text-gray-500">{{ t('keys.quickImport.unsupported') }}</span>
          </button>
        </div>
      </template>
      <template v-else>
        <div class="flex items-center justify-between">
          <strong>{{ agentName }}</strong>
          <button class="btn btn-secondary" @click="back">{{ t('keys.quickImport.back') }}</button>
        </div>
        <p v-if="!compatible" class="text-sm text-amber-600">{{ t('keys.quickImport.unsupported') }}</p>
        <div v-if="step === 'actions'" class="space-y-3">
          <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-600">
            <button data-testid="auto" class="btn btn-primary" :disabled="!compatible" @click="step = 'auto'">{{ t('keys.quickImport.auto') }}</button>
            <p class="my-2 text-sm text-gray-500">{{ t('keys.quickImport.autoHint') }}</p>
            <button class="text-sm text-primary-600" @click="step = 'clean'">{{ t('keys.quickImport.clean') }}</button>
          </div>
          <button v-if="!hideCcs" data-testid="ccs" class="btn btn-secondary w-full" :disabled="!compatible || !supportsCcs(agent, platform)" @click="step = 'ccs'">{{ t('keys.importToCcSwitch') }}</button>
          <button data-testid="manual" class="btn btn-secondary w-full" :disabled="!compatible" @click="step = 'manual'">{{ t('keys.quickImport.manual') }}</button>
        </div>
        <UseKeyModal v-else-if="step === 'manual'" :show="show" embedded :client="agent" :api-key="apiKey" :base-url="baseUrl" :platform="platform" :allow-messages-dispatch="allowMessagesDispatch" />
        <div v-else-if="step === 'ccs'" class="space-y-3">
          <button class="btn btn-primary" @click="openCcs">{{ t('keys.quickImport.openCcs') }}</button>
          <p v-if="ccsOpened" role="status" class="text-sm">{{ t('keys.quickImport.ccsOpened') }}</p>
          <a href="https://github.com/farion1231/cc-switch/releases" target="_blank" rel="noopener noreferrer" class="block text-sm text-primary-600">{{ t('keys.quickImport.installCcs') }}</a>
        </div>
        <p v-else class="text-sm text-gray-500">{{ t('keys.quickImport.comingSoon') }}</p>
      </template>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import UseKeyModal from './UseKeyModal.vue'
import type { GroupPlatform } from '@/types'
import { importAgents, supportsAgent, supportsCcs, type ImportAgent } from '@/utils/quickImport'
import { buildCcSwitchImportDeeplink } from '@/utils/ccswitchImport'
const props = withDefaults(defineProps<{ show: boolean; keyId: number; keyName?: string; apiKey: string; baseUrl: string; platform: GroupPlatform | null; allowMessagesDispatch?: boolean; hideCcs?: boolean; active?: boolean }>(), { active: true })
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()
const agent = ref<ImportAgent | null>(null)
const step = ref('actions')
const ccsOpened = ref(false)
const agentName = computed(() => importAgents.find(item => item.id === agent.value)?.name)
const compatible = computed(() => !!agent.value && props.active !== false && supportsAgent(agent.value, props.platform, props.allowMessagesDispatch))
watch(() => [props.show, props.keyId, props.apiKey, props.platform], () => { agent.value = null; step.value = 'actions'; ccsOpened.value = false })
function back() { if (step.value === 'actions') agent.value = null; else step.value = 'actions'; ccsOpened.value = false }
function openCcs() {
  if (!agent.value || !compatible.value || !supportsCcs(agent.value, props.platform)) return
  const usageScript = `({request:{url:"{{baseUrl}}/v1/usage",method:"GET",headers:{Authorization:"Bearer {{apiKey}}"}},extractor:function(r){return {isValid:r.is_active??true,remaining:r.remaining??r.quota?.remaining??r.balance,unit:r.unit??"USD"}}})`
  const link = buildCcSwitchImportDeeplink({ baseUrl: props.baseUrl || window.location.origin, platform: props.platform, clientType: agent.value === 'gemini' ? 'gemini' : 'claude', providerName: props.keyName || 'Sub2API', apiKey: props.apiKey, usageScript })
  window.location.assign(link)
  ccsOpened.value = true
}
</script>
