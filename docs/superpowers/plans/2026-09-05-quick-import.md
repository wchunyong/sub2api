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

- [ ] Add failing tests: initial Agent selection; selected Agent carried into manual view; CCS visibility; resetting on changed key.
- [ ] Add capability definitions and explicit CCS target mapping. Extract a reusable manual content wrapper using embedded mode, retaining existing standalone tests.
- [ ] Replace row actions and add post-create entry. Run focused Vitest suites and typecheck.
- [ ] Commit `feat: add agent-first quick import wizard`.

## 2. Reversible local installer

Files: `backend/internal/quickimport/assets/installer.py`, `install.ps1`, `install.sh`, `tests/test_installer.py`.

- [ ] Write failing isolated-directory tests for all Agents: merge, repeated install, clean restore, later edits, conflicting edits and cross-Agent isolation.
- [ ] Implement structured JSON merging and TOML managed provider section; use atomic writes, restrictive permissions, per-Agent journals, locking and offline recovery.
- [ ] Add platform launchers, explicit runtime checks and no-secret errors. Run Python unittest and PowerShell validation.
- [ ] Verify actual local OpenCode can load the installed provider and call a mock API; verify cleanup preserves original configuration.
- [ ] Commit `feat: add reversible agent configuration scripts`.

## 3. One-time configuration exchange

Files: `backend/internal/quickimport/tickets.go`, tests, `backend/internal/handler/quick_import_handler.go`, APIKey handler wiring and user routes.

- [ ] Write failing tests for atomic consumption, expiry, scope, ownership and revoked keys.
- [ ] Store only key ID/user ID/Agent/model in short-lived tickets; retrieve key again on redemption. Register authenticated issue and rate-limited public exchange routes. Serve embedded versioned assets.
- [ ] Integrate public API base setting and server-side compatibility checks; redact ticket fields from audit logging.
- [ ] Run scoped Go tests and commit `feat: add one-time quick import configuration exchange`.

## 4. Connect wizard and validate

Files: `frontend/src/api/quickImport.ts`, wizard, locales, tests and this plan.

- [ ] Add failing tests for command generation, stale responses and selected-Agent cleanup.
- [ ] Connect ticket API, platform-specific commands, runtime help and offline cleanup path.
- [ ] Run relevant front-end tests, typecheck/build, backend tests and installer tests.
- [ ] Run a minimal real OpenCode request with the authorized key against the confirmed gateway, in temporary storage with tools disabled; report outcome without credentials.
- [ ] Review diff and secret exposure; commit `feat: connect quick import and cleanup commands`.

## Validation record

- Baseline: 30 existing UseKeyModal and CCS tests passed on Windows; Node reports an existing localstorage-file warning.
- Branch created from local release; no release merge or push requested.
