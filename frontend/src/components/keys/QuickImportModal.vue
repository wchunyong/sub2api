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
            <button data-testid="auto" class="btn btn-primary" :disabled="!compatible || !automaticAgent(agent)" @click="step = 'auto'">{{ t('keys.quickImport.auto') }}</button>
            <p class="my-2 text-sm text-gray-500">{{ t('keys.quickImport.autoHint') }}</p>
            <button data-testid="clean" class="text-sm text-primary-600 disabled:opacity-50" :disabled="!automaticAgent(agent)" @click="step = 'clean'">{{ t('keys.quickImport.clean') }}</button>
            <p v-if="!automaticAgent(agent)" class="mt-2 text-xs text-gray-500">{{ t('keys.quickImport.manualOnly') }}</p>
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
        <div v-else class="space-y-4">
          <p class="text-sm text-gray-500">{{ t(step === 'clean' ? 'keys.quickImport.cleanHint' : 'keys.quickImport.runtimeHint') }}</p>
          <label class="block text-sm">
            {{ t('keys.quickImport.system') }}
            <select v-model="os" class="input mt-1 w-full">
              <option value="windows">Windows (PowerShell)</option>
              <option value="unix">macOS / Linux / WSL</option>
            </select>
          </label>
          <template v-if="step === 'auto'">
            <label class="block text-sm">{{ t('keys.quickImport.model') }}
              <input v-model="model" maxlength="200" class="input mt-1 w-full" :placeholder="t('keys.quickImport.modelHint')" />
            </label>
            <button data-testid="generate" class="btn btn-primary" :disabled="loading || !compatible" @click="generate">{{ t(loading ? 'keys.quickImport.generating' : 'keys.quickImport.generate') }}</button>
            <p v-if="command" class="text-xs text-gray-500">{{ t('keys.quickImport.expiry') }}</p>
          </template>
          <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
          <template v-if="displayCommand">
            <pre class="max-h-52 overflow-auto whitespace-pre-wrap break-all rounded-lg bg-gray-950 p-4 text-xs text-gray-100"><code>{{ displayCommand }}</code></pre>
            <button data-testid="copy-command" class="btn btn-secondary" @click="copyCommand">{{ t(copied ? 'keys.useKeyModal.copied' : 'keys.useKeyModal.copy') }}</button>
            <p class="text-sm text-gray-500">{{ t('keys.quickImport.resultHint') }}</p>
          </template>
          <details class="text-sm text-gray-500"><summary class="cursor-pointer">{{ t('keys.quickImport.terminalHelp') }}</summary>
            <p class="mt-2">{{ t(os === 'windows' ? 'keys.quickImport.windowsHelp' : 'keys.quickImport.unixHelp') }}</p>
            <p class="mt-2">{{ t('keys.quickImport.installHelp') }}</p>
          </details>
        </div>
      </template>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import UseKeyModal from './UseKeyModal.vue'
import type { GroupPlatform } from '@/types'
import { importAgents, supportsAgent, supportsCcs, type ImportAgent } from '@/utils/quickImport'
import { buildCcSwitchImportDeeplink } from '@/utils/ccswitchImport'
import { automaticAgent, cleanupCommand, importCommand, type ImportOS } from '@/utils/quickImportCommands'
import { createImportTicket, importServer } from '@/api/quickImport'
import { useClipboard } from '@/composables/useClipboard'
const props = withDefaults(defineProps<{ show: boolean; keyId: number; keyName?: string; apiKey: string; baseUrl: string; platform: GroupPlatform | null; allowMessagesDispatch?: boolean; hideCcs?: boolean; active?: boolean; claudeCodeOnly?: boolean }>(), { active: true })
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const agent = ref<ImportAgent | null>(null)
const step = ref('actions')
const ccsOpened = ref(false)
const os = ref<ImportOS>(/Windows/i.test(navigator.userAgent) ? 'windows' : 'unix')
const model = ref('')
const command = ref('')
const error = ref('')
const loading = ref(false)
const copied = ref(false)
let controller: AbortController | undefined
let generation = 0
let expiry: ReturnType<typeof setTimeout> | undefined
function invalidate() {
  generation++; controller?.abort(); controller = undefined
  clearTimeout(expiry); command.value = ''; error.value = ''; loading.value = false; copied.value = false
}
const displayCommand = computed(() => step.value === 'clean' && agent.value && automaticAgent(agent.value) ? cleanupCommand(os.value, agent.value) : command.value)
watch([agent, step, os, model, () => props.show, () => props.keyId, () => props.apiKey, () => props.platform, () => props.active, () => props.baseUrl], invalidate, { flush: 'sync' })
onBeforeUnmount(invalidate)
async function generate() {
  if (!agent.value || !compatible.value || !automaticAgent(agent.value)) return
  invalidate(); const current = generation; const target = agent.value
  loading.value = true; controller = new AbortController()
  try {
    const ticket = await createImportTicket(props.keyId, target, model.value, controller.signal)
    if (current !== generation) return
    if (ticket.agent !== target) throw new Error('Agent mismatch')
    command.value = importCommand(os.value, target, importServer(), ticket.ticket)
    expiry = setTimeout(() => { command.value = ''; error.value = t('keys.quickImport.expired') }, ticket.expires_in * 1000)
  } catch {
    if (current === generation) error.value = t('keys.quickImport.failed')
  } finally { if (current === generation) loading.value = false }
}
async function copyCommand() {
  const value = displayCommand.value
  const current = generation
  const success = await copyToClipboard(value)
  if (current === generation) copied.value = success
}
const agentName = computed(() => importAgents.find(item => item.id === agent.value)?.name)
const compatible = computed(() => !!agent.value && props.active !== false && (!props.claudeCodeOnly || agent.value === 'claude') && supportsAgent(agent.value, props.platform, props.allowMessagesDispatch))
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
