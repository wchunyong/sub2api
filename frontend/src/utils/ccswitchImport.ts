import type { GroupPlatform } from '@/types'
import type { ImportAgent } from './quickImport'
import { buildCodexModelsManifestUrl } from '@/api/codex'

export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-5.5'
export const GROK_CC_SWITCH_MODEL = 'grok-4.5'

export type CcSwitchClientType = 'claude' | 'gemini'

export interface CcSwitchImportConfig {
  app: string
  endpoint: string
  model?: string
  apiFormat?: 'anthropic' | 'openai_chat' | 'openai_responses' | 'gemini_native'
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  platform?: GroupPlatform | null
  clientType: CcSwitchClientType
  providerName: string
  apiKey: string
  usageScript: string
  agent?: ImportAgent
  model?: string
  models?: string[]
}

/**
 * Balance query CC Switch runs against the imported provider. CC Switch fills
 * `{{baseUrl}}` with the provider's base URL as stored — Codex and Grok imports
 * carry a trailing `/v1` (see `withV1Endpoint`), Claude ones do not, and users
 * may edit it either way afterwards — then evaluates the script, so the URL
 * strips an existing `/v1` instead of blindly appending one (`/v1/v1/usage`
 * is a 404 and CC Switch shows "query failed").
 */
export const CC_SWITCH_USAGE_SCRIPT = `({
    request: {
      url: "{{baseUrl}}".replace(/\\/+$/, "").replace(/\\/v1$/, "") + "/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`

function withV1Endpoint(baseUrl: string): string {
  const normalizedBaseUrl = baseUrl.replace(/\/+$/, '')
  return normalizedBaseUrl.endsWith('/v1') ? normalizedBaseUrl : `${normalizedBaseUrl}/v1`
}

function withoutTrailingSlashes(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '')
}

export function resolveCcSwitchImportConfig(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType,
  baseUrl: string
): CcSwitchImportConfig {
  switch (platform || 'anthropic') {
    case 'antigravity':
      return {
        app: clientType === 'gemini' ? 'gemini' : 'claude',
        endpoint: `${baseUrl.replace(/\/+$/, '')}/antigravity`
      }
    case 'openai':
      return {
        app: 'codex',
        // CC Switch's Codex provider appends the OpenAI-compatible path itself.
        // Passing /v1 here can make the client request /v1/v1/....
        endpoint: withoutTrailingSlashes(baseUrl),
        model: OPENAI_CC_SWITCH_CODEX_MODEL
      }
    case 'gemini':
      return {
        app: 'gemini',
        endpoint: baseUrl
      }
    case 'grok':
      return {
        app: 'grokbuild',
        endpoint: withV1Endpoint(baseUrl),
        model: GROK_CC_SWITCH_MODEL
      }
    default:
      return {
        app: 'claude',
        endpoint: baseUrl
      }
  }
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const config = resolveSelectedAgentConfig(input) ?? resolveCcSwitchImportConfig(input.platform, input.clientType, input.baseUrl)
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', config.app],
    ['name', input.providerName],
    ['homepage', input.baseUrl],
    ['endpoint', config.endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', btoa(input.usageScript)],
    ['usageAutoInterval', '30']
  ]

  if (config.model) {
    entries.splice(2, 0, ['model', config.model])
  }
  if (config.apiFormat) {
    entries.push(['apiFormat', config.apiFormat])
  }
  if (input.agent === 'claude' && input.platform !== 'anthropic' && input.model) {
    entries.push(['haikuModel', input.model], ['sonnetModel', input.model], ['opusModel', input.model])
  }

  // CC Switch keeps Claude's imported env block. Claude Code then discovers
  // every model exposed by this key's gateway rather than one hard-coded row.
  if (input.agent === 'claude' && input.models?.length) {
    entries.push(['config', btoa(JSON.stringify({ env: { CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY: '1' } }))])
  }

  // CC Switch preserves model_catalog_url from a Codex provider TOML. Codex
  // fetches its full model manifest from the same gateway after import.
  if (input.agent === 'codex' && input.models?.length) {
    const endpoint = config.endpoint
    const catalogUrl = buildCodexModelsManifestUrl(endpoint)
    const toml = `model_provider = "lianjieai"\nmodel = ${JSON.stringify(input.model || input.models[0])}\n\n[model_providers.lianjieai]\nname = "lianjieai"\nbase_url = ${JSON.stringify(endpoint)}\nwire_api = "responses"\nmodel_catalog_url = ${JSON.stringify(catalogUrl)}\n`
    entries[entries.findIndex(([name]) => name === 'configFormat')] = ['configFormat', 'toml']
    entries.push(['config', btoa(toml)])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}

function resolveSelectedAgentConfig(input: CcSwitchImportDeeplinkInput): CcSwitchImportConfig | undefined {
  if (input.agent === 'codex' || input.agent === 'opencode' || input.agent === 'pi') {
    const root = withoutTrailingSlashes(input.baseUrl).replace(/\/v1$/, '')
    const openCodeFormat = input.platform === 'anthropic' || input.platform === 'antigravity'
      ? 'anthropic' : input.platform === 'gemini'
        ? 'gemini_native' : input.platform === 'openai'
          ? 'openai_responses' : 'openai_chat'
    const endpoint = input.agent === 'pi' && input.platform === 'anthropic'
      ? root
      : input.agent === 'pi' && input.platform === 'antigravity'
        ? `${root}/antigravity`
        : (input.agent === 'opencode' || input.agent === 'pi') && input.platform === 'antigravity'
      ? `${root}/antigravity/v1`
      : (input.agent === 'opencode' || input.agent === 'pi') && input.platform === 'gemini'
        ? `${root}/v1beta`
        : withV1Endpoint(input.baseUrl)
    return {
      app: input.agent,
      endpoint,
      model: input.model || (input.agent === 'codex' && input.platform === 'openai' ? OPENAI_CC_SWITCH_CODEX_MODEL : undefined),
      apiFormat: input.agent === 'codex' ? 'openai_responses' : openCodeFormat
    }
  }
  if (input.agent === 'claude') {
    const root = withoutTrailingSlashes(input.baseUrl).replace(/\/v1$/, '')
    return {
      app: 'claude',
      endpoint: input.platform === 'antigravity' ? `${root}/antigravity` : root,
      model: input.model,
      apiFormat: 'anthropic'
    }
  }
}
