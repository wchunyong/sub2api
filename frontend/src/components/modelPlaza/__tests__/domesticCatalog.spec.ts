import { describe, expect, it } from 'vitest'
import { domesticCatalogFamilies } from '../domesticCatalog'

describe('domestic model catalog', () => {
  it('covers the planned domestic families without the two hidden DeepSeek models', () => {
    const names = domesticCatalogFamilies.flatMap(family => family.models.map(model => model.name))
    expect(names).toHaveLength(12)
    expect(names).toContain('hy3')
    expect(names).toContain('hy4-preview')
    expect(names).toContain('kimi-k2.7')
    expect(names).toContain('minimax-m3')
    expect(names).not.toContain('deepseek-v4-flash')
    expect(names).not.toContain('deepseek-v4-pro')
  })

  it('keeps base prices for 0.3 planned calculations', () => {
    const hunyuan = domesticCatalogFamilies.find(family => family.platform === 'hunyuan')!
    const hy3 = hunyuan.models.find(model => model.name === 'hy3')!
    expect((hy3.pricing?.input_price ?? NaN) * 1_000_000 * 0.3).toBeCloseTo(0.0396)
    expect((hy3.pricing?.output_price ?? NaN) * 1_000_000 * 0.3).toBeCloseTo(0.1584)
    expect((hy3.pricing?.cache_read_price ?? NaN) * 1_000_000 * 0.3).toBeCloseTo(0.0099)
  })
})
