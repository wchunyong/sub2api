# Quick Import Implementation Plan

**Goal:** Ship Agent-first quick import with reversible local configuration on a dedicated feature branch.
**Architecture:** Vue wizard preserves existing manual configuration; authenticated one-time tickets deliver typed configuration to a versioned local installer. Each Agent owns its files and recovery records.
**Tech Stack:** Vue/TypeScript, Go/Gin, Python standard-library installer with PowerShell/Shell launchers.
**Spec:** ../specs/2026-09-05-quick-import-design.md

## Global constraints

- Work on `feature/quick-import-agents`; commit each verified feature, never merge release.
- Keep API credentials out of commits, test fixtures, console output and command URLs.
- Select Agent before actions. Preserve existing manual clients, authentication modes and CCS hiding setting.
- Only restore changes owned by the selected Agent; preserve later edits and reject conflicting restoration.
- Run OpenCode against an isolated configuration, first against a mock server then the authorized gateway once its URL is known.
- Python 3.11+ provides JSON and TOML parsing in the local installer; missing runtime leads to installation instructions, no silent install.

## 1. Agent-first wizard

Files: `frontend/src/components/keys/QuickImportModal.vue`, `UseKeyModal.vue`, `frontend/src/utils/quickImport.ts`, `ccswitchImport.ts`, `frontend/src/views/user/KeysView.vue`, dashboard locales and component tests.

- [x] Add failing tests: initial Agent selection; selected Agent carried into manual view; CCS visibility; resetting on changed key.
- [x] Add capability definitions and explicit CCS target mapping. Extract a reusable manual content wrapper using embedded mode, retaining existing standalone tests.
- [x] Replace row actions and add post-create entry. Run focused Vitest suites and typecheck.
- [x] Commit `feat: add agent-first quick import wizard`.

## 2. Reversible local installer

Files: `backend/internal/quickimport/assets/installer.py`, `install.ps1`, `install.sh`, `tests/test_installer.py`.

- [x] Write failing isolated-directory tests for all Agents: merge, repeated install, clean restore, later edits, conflicting edits and cross-Agent isolation.
- [x] Implement structured JSON merging and TOML managed provider section; use atomic writes, restrictive permissions, per-Agent journals, locking and offline recovery.
- [x] Add platform launchers, explicit runtime checks and no-secret errors. Run Python unittest and PowerShell validation.
- [x] Verify actual local OpenCode can load the installed provider and call a mock API; verify cleanup preserves original configuration.
- [x] Commit `feat: add reversible agent configuration scripts`.

## 3. One-time configuration exchange

Files: `backend/internal/quickimport/tickets.go`, tests, `backend/internal/handler/quick_import_handler.go`, APIKey handler wiring and user routes.

- [x] Write failing tests for atomic consumption, expiry, scope, ownership and revoked keys.
- [x] Store only key ID/user ID/Agent/model in short-lived tickets; retrieve key again on redemption. Register authenticated issue and rate-limited public exchange routes. Serve embedded versioned assets.
- [x] Integrate public API base setting and server-side compatibility checks; redact ticket fields from audit logging.
- [x] Run scoped Go tests and commit `feat: add one-time quick import configuration exchange`.

## 4. Connect wizard and validate

Files: `frontend/src/api/quickImport.ts`, wizard, locales, tests and this plan.

- [x] Add failing tests for command generation, stale responses and selected-Agent cleanup.
- [x] Connect ticket API, platform-specific commands, runtime help and offline cleanup path.
- [x] Run relevant front-end tests, typecheck/build, backend tests and installer tests.
- [ ] Run a minimal real OpenCode request with the authorized key against the confirmed gateway, in temporary storage with tools disabled; report outcome without credentials.
- [x] Review diff and secret exposure; commit `feat: connect quick import and cleanup commands`.

## Validation record

- Baseline: 30 existing UseKeyModal and CCS tests passed on Windows; Node reports an existing localstorage-file warning.
- Branch created from local release; no release merge or push requested.


## Implementation and verification — 2026-09-05

Implemented on `feature/quick-import-agents`. Changes have been committed by feature; no merge to release or remote push was performed.

- Agent-first wizard, embedded manual setup, scoped CCS import and post-create entry.
- Redis tickets (5 minute TTL, atomic consumption, no API key retained), authenticated issuance and public rate-limited exchange. Permission and key status checked twice.
- Per-Agent Python installer and Windows/Unix launchers. Structured configuration merge, protected journals, rollback on failure, offline cleanup with later-edit preservation and conflict detection.
- Additional review fixes: Claude-only group restrictions, missing-client installation guidance, empty OpenCode desktop JSONC compatibility, same-origin HTTPS model-list probe before writing.
- Frontend: 45 relevant tests passed, typecheck and scoped ESLint passed. Production build passed with existing bundle-size/Browserslist warnings.
- Backend: quick import package, handler, routes compilation and unit-tagged API contract tests passed.
- Installer: 11 Python tests passed on Windows. PowerShell script parsed successfully.
- OpenCode CLI 1.18.29: actual binary loaded the installer-generated provider, sent 2 requests to the local mock endpoint with expected mock authorization/model, received IMPORT_OK, then cleanup removed owned settings and preserved OpenCode-added `$schema`.
- User-provided production key was not written to code, fixtures or Git. Actual gateway request is pending its API Base URL; local OpenCode configuration had no provider endpoint to identify it. Mock tests cannot establish production key validity.
- macOS/Linux real-machine validation remains outstanding. Only Docker Desktop WSL distribution is present locally. Shell launcher is provided but cross-platform acceptance is not claimed.
- Complex existing OpenCode JSONC, custom config environment overrides and unusual TOML layouts intentionally direct users to manual setup rather than overwrite ambiguous settings.
- ChatGPT product target remains separate from Codex; no unverified ChatGPT import card is shipped.


## OpenCode provider/model refinement — 2026-09-05

- User confirmed the initial local OpenCode setup works. Provider display name is now `lianjieai`; the stable internal provider ID remains unchanged for recovery compatibility.
- Each automatic OpenCode import obtains the authenticated `/v1/models` list, deduplicates valid IDs, maps model names into OpenCode's provider catalog, and preserves the selected default. A missing selected model fails explicitly instead of writing an unavailable default.
- Gateway probe now sends an explicit client User-Agent and JSON Accept header; the gateway rejected the default Python user agent during the earlier local setup.
- Local user configuration synchronized with the gateway's 10 models; OpenCode CLI `models sub2api_quick` lists all 10. Default remains `gpt-5.5`. No generation request was made during this update.
- Installer suite: 13 passing tests. Actual OpenCode mock invocation and cleanup pass.
- This local update has a separate recovery record. Cleanup once undoes this refinement; a second cleanup undoes the initial import. No cleanup was executed on the user's configuration.

## Claude Code / Codex 模型同步推广 — 2026-09-05

- 用户确认 OpenCode 本地验证通过后，将命名、模型目录同步和可恢复导入推广到现有自动导入目标 Claude Code、Codex。
- Claude Code：从当前密钥的 `/v1/models` 获取列表；2.1.242 及以上写入 `modelPicker`，标签使用 `lianjieai`。较旧版本仅配置所选模型和自定义模型名称，提示升级或重新导入以切换模型，不写入其不支持的新字段。
- Codex：供应商名称为 `lianjieai`，按本机版本请求 `/v1/models?client_version=...`，将专用目录写入独立文件并配置 `model_catalog_json`。由网关已有指令模板和推理摘要标志补齐旧客户端要求的兼容字段。
- Codex 目录文件随本次导入记录管理；清理恢复原目录引用并删除本次生成文件。发现用户修改目录时停止清理；清理提交失败时恢复目录和配置。
- 本机 Codex 0.142.5 在临时 CODEX_HOME 下读取真实网关的 6 个模型，与响应集合一致，清理验证通过。仅读取模型接口，未发起生成请求。
- 本机 Claude Code 2.1.215 走旧版配置分支；完整新版菜单尚未在新版实际客户端验收。配置与恢复由隔离测试覆盖。
- OpenCode 实际 CLI 对本地 mock 网关的请求与清理回归通过；用户真实 OpenCode 配置未改动。
- Gemini、Grok、Codex WebSocket 保持已有手动配置能力，本轮不增加未经验证的自动安装适配。macOS/Linux 实机验收仍待完成。
- 参考 Claude 官方 model-config、settings-reference 与 Codex 官方 config-reference；模型列表在每次导入时更新，不启动后台常驻同步任务。
- 最终验证：19 项 Python 安装器测试、quickimport Go 单元测试、实际 OpenCode mock 回归通过；独立代码审查无待修复问题。


## Native helper and short commands — 2026-09-05

- Removed the end-user Python dependency. The bootstrap downloads a SHA256-checked Go executable for Windows, macOS or Linux (amd64/arm64); no Node.js or Go installation is required either.
- `/setup/<agent>.ps1` and `.sh` serve short launchers. The one-time ticket stays in the command argument, never the script URL; API keys remain in POST exchange only.
- Native binaries and checksums are generated before server builds in Docker, Make and GoReleaser (`go run ./cmd/build-quick-import` from backend). Direct development builds must run this generator to enable downloads.
- Verified helpers are cached by content hash; per-Agent recovery scripts use the cached helper offline. New cleanup commands bootstrap once when only legacy Python recovery exists; the native engine reads legacy journal records without Python.
- Maintained per-field restore, locks, pending intent, catalog ownership, same-origin HTTPS, redirect refusal, model sync and version compatibility. JSON decoders retain large integers; complex TOML including unsupported NaN/Inf comparison fails closed.
- Tests: native engine/network/CLI, handler and route suites; actual Windows PowerShell with Python removed from PATH; all three Agent configuration round trips; downloader/checksum and offline recovery; Linux WSL native round trips and launcher recovery. Actual OpenCode binary made two requests to a mock gateway with native-generated configuration and cleanup passed.
- Six binaries cross-compiled; macOS real-machine execution remains unverified. Actual production-key exchange was not performed during these tests.

- Broader embedded-web static-file test still expects `/logo.png`, while the tracked frontend supplies only `/logo.svg`; that pre-existing fixture mismatch fails independently. Targeted API/setup bypass tests pass and the production server builds successfully with embedded helpers.
