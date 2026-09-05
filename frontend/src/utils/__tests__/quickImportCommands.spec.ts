import { describe, it, expect } from 'vitest'
import { importCommand, cleanupCommand } from '../quickImportCommands'
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
})
