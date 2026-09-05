import { describe, it, expect } from 'vitest'
import { importCommand, cleanupCommand } from '../quickImportCommands'
import { execFileSync } from 'node:child_process'
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
describe('quick import commands', () => {
  it('restricts gateways to HTTPS and binds the chosen Agent', () => {
    expect(() => importCommand('windows', 'opencode', 'http://example.com', 'a'.repeat(64))).toThrow()
    const command = importCommand('windows', 'opencode', 'https://example.com', 'a'.repeat(64))
    expect(command).toContain("/setup/opencode.ps1")
    expect(command).not.toContain('sk-')
    expect(command).not.toContain('python')
    expect(command).not.toContain('\n')
    expect(command.length).toBeLessThan(200)
    expect(command).toContain('-MaximumRedirection 0')
  })
  it('uses local offline restoration for only the selected Agent', () => {
    const command = cleanupCommand('unix', 'claude')
    expect(command).toContain('.sub2api-quick-import/claude/restore.sh')
    expect(command).not.toContain('https:')
    expect(command).not.toContain('opencode')
    expect(() => cleanupCommand('unix', 'gemini')).toThrow()
  })
  it('retries failed cached recovery with the current helper', () => {
    const windows = cleanupCommand('windows', 'codex', 'https://example.com')
    expect(windows).toContain('$LASTEXITCODE -eq 0')
    expect(windows).toContain('/setup/codex-clean.ps1')
    const unix = cleanupCommand('unix', 'codex', 'https://example.com')
    expect(unix).toContain('restore.sh"; then :; else')
    expect(unix).toContain('/setup/codex-clean.sh')
  })
  it.skipIf(process.platform !== 'win32')('only downloads an update when local PowerShell recovery fails', () => {
    const root = mkdtempSync(join(tmpdir(), 'quick-clean-test-'))
    try {
      const folder = join(root, '.sub2api-quick-import', 'codex')
      mkdirSync(folder, { recursive: true })
      for (const code of [0, 1]) {
        writeFileSync(join(folder, 'restore.ps1'), `exit ${code}`)
        const script = `$env:USERPROFILE = '${root.replace(/'/g, "''")}'; function Invoke-RestMethod { "Write-Output 'updated-helper'" }; ${cleanupCommand('windows', 'codex', 'https://example.com')}`
        const output = execFileSync('powershell.exe', ['-NoProfile', '-Command', script], { encoding: 'utf8' })
        expect(output.includes('updated-helper')).toBe(code !== 0)
      }
    } finally {
      rmSync(root, { recursive: true, force: true })
    }
  })
})
