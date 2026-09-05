<template>
  <BaseDialog :show="show" :title="t('keys.quickImport.title')" :width="step === 'manual' ? 'wide' : 'quick-import'" @close="emit('close')">
    <template #title>{{ t('keys.quickImport.title') }}<span class="qi-step">{{ agent ? '02' : '01' }} / {{ t(agent ? 'keys.quickImport.chooseAction' : 'keys.quickImport.selectAgent') }}</span></template>
    <div class="qi-body">
      <div class="qi-key"><span class="qi-dot" />{{ keyName }} · {{ apiKey ? `${apiKey.slice(0, 6)}…${apiKey.slice(-4)}` : '' }}</div>
      <template v-if="!agent">
        <p class="qi-label">{{ t('keys.quickImport.selectAgent') }}</p>
        <div class="qi-agents">
          <button v-for="item in importAgents" :key="item.id" :data-testid="`agent-${item.id}`" class="qi-agent" :disabled="!agentSupported(item.id)" @click="agent = item.id">
            <span class="qi-logo" :class="{ 'qi-chatgpt': item.id === 'codex' }"><img v-if="agentIcons[item.id]" :src="agentIcons[item.id]" alt="" /><Icon v-else name="terminal" /></span>{{ item.name }}
          </button>
        </div>
        <p class="qi-status">{{ t(availableAgents.length ? 'keys.quickImport.supportedOnly' : 'keys.quickImport.unsupported') }}</p>
      </template>
      <template v-else>
        <div class="qi-selected"><span class="qi-logo" :class="{ 'qi-chatgpt': agent === 'codex' }"><img v-if="agentIcons[agent]" :src="agentIcons[agent]" alt="" /><Icon v-else name="terminal" /></span><strong>{{ agentName }}</strong><button class="qi-back" @click="back">← {{ t('keys.quickImport.back') }}</button></div>
        <template v-if="step === 'actions'">
          <div class="qi-actions">
            <div class="qi-choice qi-choice-main"><span class="qi-corner"><span>{{ t('keys.quickImport.recommended') }}</span></span><button class="qi-help" :aria-expanded="help === 'import'" @click="help = help === 'import' ? '' : 'import'">{{ t('keys.quickImport.operationHelp') }} ⓘ</button>
              <button data-testid="auto" class="qi-action" :disabled="loading || !compatible || !automaticAgent(agent)" @click="generate"><Icon name="terminal" size="lg" /><strong>{{ t(loading ? 'keys.quickImport.generating' : copied === 'import' ? 'keys.useKeyModal.copied' : 'keys.quickImport.copyImport') }}</strong><small>{{ t('keys.quickImport.autoHint') }}</small></button>
              <button data-testid="clean" class="qi-clean" :disabled="!automaticAgent(agent)" @click="clean"><Icon name="refresh" size="xs" />{{ t(copied === 'clean' ? 'keys.useKeyModal.copied' : 'keys.quickImport.copyClean') }}</button>
            </div>
            <div v-if="!hideCcs" class="qi-choice"><span class="qi-corner"><span>{{ t('keys.quickImport.recommended') }}</span></span><button class="qi-help" :aria-expanded="help === 'ccs'" @click="help = help === 'ccs' ? '' : 'ccs'">{{ t('keys.quickImport.operationHelp') }} ⓘ</button>
              <button data-testid="ccs" class="qi-action" :disabled="!compatible || !supportsCcs(agent, platform)" @click="openCcs"><Icon name="externalLink" size="lg" /><strong>{{ t('keys.importToCcSwitch') }}</strong><small>{{ t(supportsCcs(agent, platform) ? 'keys.quickImport.ccsHint' : 'keys.quickImport.ccsUnavailable') }}</small></button>
            </div>
            <div class="qi-choice"><button class="qi-help" :aria-expanded="help === 'manual'" @click="help = help === 'manual' ? '' : 'manual'">{{ t('keys.quickImport.operationHelp') }} ⓘ</button><button data-testid="manual" class="qi-action" :disabled="!compatible" @click="step = 'manual'"><Icon name="cog" size="lg" /><strong>{{ t('keys.quickImport.manual') }}</strong><small>{{ t('keys.quickImport.manualHint') }}</small></button></div>
          </div>
          <div v-if="help" class="qi-help-panel" role="note">{{ t(help === 'import' ? (os === 'windows' ? 'keys.quickImport.powershellHelp' : 'keys.quickImport.shellHelp') : help === 'ccs' ? 'keys.quickImport.ccsHelp' : 'keys.quickImport.manualHelp') }}</div>
          <p v-if="os === 'windows'" class="qi-status qi-shell-name">{{ t('keys.quickImport.powershellOnly') }}</p>
          <p class="qi-status" role="status" aria-live="polite">{{ t(copied === 'import' ? 'keys.quickImport.importCopied' : copied === 'clean' ? 'keys.quickImport.cleanCopied' : ccsOpened ? 'keys.quickImport.ccsOpened' : 'keys.quickImport.directHint') }}</p>
          <a v-if="ccsOpened" href="https://github.com/farion1231/cc-switch/releases" target="_blank" rel="noopener noreferrer" class="qi-status">{{ t('keys.quickImport.installCcs') }}</a>
          <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
          <button v-if="error && command" class="qi-back" @click="copyCommand('import')">{{ t('keys.quickImport.copyImport') }}</button>
        </template>
        <UseKeyModal v-else :show="show" embedded :client="agent" :api-key="apiKey" :base-url="baseUrl" :platform="platform" :allow-messages-dispatch="allowMessagesDispatch" />
      </template>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import claudeIcon from '@/assets/agents/claude.ico'
import chatgptIcon from '@/assets/agents/chatgpt.png'
import opencodeIcon from '@/assets/agents/opencode.png'
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
const help = ref('')
watch([agent, step, () => props.show], () => { help.value = '' })
const ccsOpened = ref(false)
const os = ref<ImportOS>(/Windows/i.test(navigator.userAgent) ? 'windows' : 'unix')
const command = ref('')
const error = ref('')
const loading = ref(false)
const copied = ref<'import' | 'clean' | ''>('')
let controller: AbortController | undefined
let generation = 0
let expiry: ReturnType<typeof setTimeout> | undefined
function invalidate() {
  generation++; controller?.abort(); controller = undefined
  clearTimeout(expiry); command.value = ''; error.value = ''; loading.value = false; copied.value = ''
}
watch([agent, step, os, () => props.show, () => props.keyId, () => props.apiKey, () => props.platform, () => props.active, () => props.baseUrl, () => props.claudeCodeOnly, () => props.allowMessagesDispatch], invalidate, { flush: 'sync' })
onBeforeUnmount(invalidate)
async function generate() {
  if (!agent.value || !compatible.value || !automaticAgent(agent.value)) return
  invalidate(); const current = generation; const target = agent.value
  loading.value = true; controller = new AbortController()
  try {
    const ticket = await createImportTicket(props.keyId, target, '', controller.signal)
    if (current !== generation) return
    if (ticket.agent !== target) throw new Error('Agent mismatch')
    command.value = importCommand(os.value, target, importServer(), ticket.ticket)
    await copyCommand('import', current)
    if (current !== generation) return
    expiry = setTimeout(() => { command.value = ''; copied.value = ''; error.value = t('keys.quickImport.expired') }, ticket.expires_in * 1000)
  } catch {
    if (current === generation) error.value = t('keys.quickImport.failed')
  } finally { if (current === generation) loading.value = false }
}
async function copyCommand(kind: 'import' | 'clean', current = generation) {
  if (!agent.value || !automaticAgent(agent.value)) return
  const value = kind === 'clean' ? cleanupCommand(os.value, agent.value, importServer()) : command.value
  try {
    const success = await copyToClipboard(value)
    if (current === generation) {
      copied.value = success ? kind : ''
      error.value = success ? '' : t('keys.quickImport.copyFailed')
    }
  } catch { if (current === generation) error.value = t('keys.quickImport.copyFailed') }
}
async function clean() {
  invalidate()
  await copyCommand('clean')
}
const agentIcons: Partial<Record<ImportAgent, string>> = { claude: claudeIcon, codex: chatgptIcon, opencode: opencodeIcon }
function agentSupported(id: ImportAgent) { return (!props.claudeCodeOnly || id === 'claude') && supportsAgent(id, props.platform, props.allowMessagesDispatch) }
const availableAgents = computed(() => importAgents.filter(item => agentSupported(item.id)))
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

<style scoped>

.qi-body{--qi-bg:light-dark(#ffffff,#1d293b);--qi-text:light-dark(#162334,#f1f5fa);--qi-sub:light-dark(#637185,#a2b0c3);--qi-line:light-dark(#e1e7ee,#354357);--qi-hover:light-dark(#f3f7fa,#25344a);--qi-accent:#0dafa2;--qi-soft:light-dark(#e9faf7,#173c40);font-family:Inter,"Microsoft YaHei",sans-serif;color:var(--qi-text);display:grid;gap:24px;padding:4px;}
 .qi-body *{box-sizing:border-box}.qi-window{background:var(--qi-bg);border:1px solid var(--qi-line);border-radius:20px;overflow:hidden;width:100%;max-width:580px;margin:auto;box-shadow:0 12px 32px #00000012}
.qi-head{display:flex;align-items:center;justify-content:space-between;padding:22px 26px;border-bottom:1px solid var(--qi-line)}.qi-body h2{font-size:20px;font-weight:650;margin:0;line-height:1.4}.qi-step{font-size:12px;color:var(--qi-sub);margin-top:4px}.qi-body button{font:inherit;cursor:pointer;color:inherit}.qi-close{border:0;background:none;padding:10px;color:var(--qi-sub);display:grid;place-items:center;border-radius:9px}.qi-close:hover{background:var(--qi-hover)}.qi-body{padding:24px 26px}.qi-key{display:flex;align-items:center;gap:8px;font-size:12px;color:var(--qi-sub);margin-bottom:22px}.qi-dot{height:6px;width:6px;border-radius:50%;background:var(--qi-accent)}.qi-label{font-size:15px;font-weight:550;margin:0 0 14px}.qi-agents{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}.qi-agent{border:1px solid var(--qi-line);background:transparent;border-radius:14px;display:flex;align-items:center;flex-direction:column;gap:13px;padding:22px 8px;font-size:14px;font-weight:550;transition:background .15s,border-color .15s}.qi-agent:hover{background:var(--qi-soft);border-color:var(--qi-accent)}.qi-logo{width:42px;height:42px;border-radius:12px;display:grid;place-items:center;background:var(--qi-hover)}.qi-logo svg{width:24px;height:24px}.qi-claude{color:#da947a}.qi-foot{padding:16px 26px;color:var(--qi-sub);font-size:12px;background:var(--qi-hover);line-height:1.7}.qi-selected{display:flex;align-items:center;gap:12px;margin-bottom:22px}.qi-name{font-size:17px;font-weight:600}.qi-body .qi-back{margin-left:auto;background:none;border:0;color:var(--qi-sub);font-size:13px;padding:10px 0 10px 12px;display:flex;gap:5px;align-items:center}.qi-actions{display:flex;flex-direction:column;gap:10px}.qi-action{display:flex;align-items:center;gap:14px;width:100%;text-align:left;padding:17px 18px;border-radius:12px;background:transparent;border:1px solid var(--qi-line);min-height:70px}.qi-action:hover{background:var(--qi-hover)}.qi-action.qi-primary{background:var(--qi-accent);border-color:var(--qi-accent);color:#fff}.qi-primary:hover{filter:brightness(1.06)}.qi-action .qi-copy{margin-left:auto;opacity:.8;flex-shrink:0}.qi-action strong{font-size:14px;font-weight:600;display:block}.qi-action small{font-size:12px;line-height:1.6;display:block;color:var(--qi-sub);margin-top:3px}.qi-primary small{color:#e3fffa}.qi-action>svg{width:20px;height:20px;flex-shrink:0}.qi-separator{height:1px;background:var(--qi-line);margin:7px 0}.qi-status{font-size:12px;line-height:1.7;color:var(--qi-sub);margin-top:17px;min-height:21px}.qi-status[data-success="true"]{color:light-dark(#08766e,#5ce0cc)}.qi-action[hidden]{display:none}.qi-reopen{background:var(--qi-bg);border:1px solid var(--qi-line);padding:16px;border-radius:12px}.qi-body [hidden]{display:none!important}@media(max-width:420px){.qi-head,.qi-body{padding:20px}.qi-agents{gap:8px}.qi-agent{font-size:12px;padding:18px 4px}.qi-action{padding:15px 12px;gap:10px}.qi-foot{padding:14px 20px}}
.qi-window{max-width:700px}.qi-actions{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;align-items:stretch}.qi-choice{border:1px solid var(--qi-line);border-radius:14px;overflow:hidden;display:flex;flex-direction:column}.qi-choice-main{background:var(--qi-soft);border-color:var(--qi-accent)}.qi-action{flex-direction:column;text-align:center;justify-content:center;gap:16px;border:0;border-radius:0;min-height:162px;height:100%;padding:22px 10px;background:transparent}.qi-action.qi-primary{color:var(--qi-text);background:transparent;border:0}.qi-action.qi-primary>svg{color:var(--qi-accent)}.qi-primary small{color:var(--qi-sub)}.qi-action .qi-copy{display:none}.qi-action>svg{width:25px;height:25px}.qi-action strong{line-height:1.6}.qi-body .qi-clean{display:flex;align-items:center;justify-content:center;gap:6px;color:var(--qi-sub);background:none;border:0;border-top:1px solid var(--qi-line);padding:13px 5px;min-height:44px;font-size:12px}.qi-clean:hover{background:var(--qi-hover);color:var(--qi-text)}.qi-clean strong{font-weight:400}.qi-clean svg{width:14px;height:14px}.qi-logo img{width:28px;height:28px;object-fit:contain}.qi-choice[hidden]{display:none}.qi-actions.qi-two{grid-template-columns:repeat(2,minmax(0,1fr))}@media(max-width:420px){.qi-actions{gap:7px}.qi-action{padding:18px 5px;min-height:152px}.qi-action strong{font-size:12px}.qi-action small{font-size:11px}.qi-body .qi-clean{font-size:11px}.qi-clean svg{display:none}}
.qi-logo:has(.qi-chatgpt-image){background:#ffffff}.qi-logo .qi-chatgpt-image{width:30px;height:30px;object-fit:contain}

.qi-actions:has(>.qi-choice:nth-child(2):last-child){grid-template-columns:repeat(2,minmax(0,1fr))}.qi-actions:has(>.qi-choice:only-child){grid-template-columns:1fr}.qi-choice-main .qi-action>svg{color:#0dafa2}.qi-action:disabled{opacity:.5;cursor:not-allowed}.qi-action{min-height:190px}.qi-body .qi-clean{flex-shrink:0}.qi-chatgpt{background:#fff}.qi-chatgpt img{width:30px;height:30px}.qi-badge{font-size:11px;border-radius:5px;padding:2px 7px;color:var(--qi-sub);background:var(--qi-hover)}.qi-placeholder{visibility:hidden}.qi-step{display:block;font-size:12px;font-weight:400;color:#8190a4;margin-top:4px}.qi-body{padding:0;display:block}
:global(.dark) .qi-body{--qi-bg:#1d293b;--qi-text:#f1f5fa;--qi-sub:#a2b0c3;--qi-line:#354357;--qi-hover:#25344a;--qi-soft:#173c40}
.qi-agent:disabled,.qi-body .qi-clean:disabled{opacity:.4;cursor:not-allowed}.qi-agent:disabled:hover{background:transparent;border-color:var(--qi-line)}.qi-action{height:auto;min-height:190px;flex:1 0 auto;justify-content:flex-start}.qi-choice{min-height:236px}.qi-body .qi-clean{margin-top:auto}.qi-status{min-height:21px}

.qi-choice{position:relative}.qi-action{padding-top:54px}.qi-corner{position:absolute;top:0;left:0;width:66px;height:66px;background:#facc15;clip-path:polygon(0 0,100% 0,0 100%);pointer-events:none;z-index:1}.qi-corner span{position:absolute;top:14px;left:3px;width:48px;text-align:center;transform:rotate(-45deg);color:#422006;font-size:11px;font-weight:600}.qi-body .qi-help{position:absolute;right:9px;top:10px;z-index:1;padding:7px 3px;font-size:12px;color:var(--qi-sub)}.qi-help:hover{text-decoration:underline}.qi-help-panel{margin-top:16px;padding:14px 16px;border:1px solid var(--qi-line);border-radius:10px;background:var(--qi-hover);font-size:13px;line-height:1.8;white-space:pre-line}.qi-shell-name{font-weight:600;color:var(--qi-text)}@media(max-width:420px){.qi-body .qi-help{font-size:11px;right:4px}.qi-corner{width:48px;height:48px}.qi-corner span{top:9px;left:0;width:36px;font-size:9px}}
</style>
