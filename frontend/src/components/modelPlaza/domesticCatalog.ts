import type { PlazaModel } from '@/api/modelPlaza'

export interface PlazaDomesticFamily {
  platform: string
  label: string
  monogram: string
  models: PlazaModel[]
}

/**
 * 国内模型 3 折计划价使用的模型原价（USD / 1M tokens）。
 * 页面数据不参与计费；模型实际可用状态由后台分组决定。
 */
function plannedModel(name: string, platform: string, input: number, output: number, cacheRead: number): PlazaModel {
  return {
    name,
    platform,
    pricing: {
      billing_mode: 'token',
      input_price: input / 1_000_000,
      output_price: output / 1_000_000,
      cache_read_price: cacheRead / 1_000_000,
      cache_write_price: null,
      image_input_price: null,
      image_output_price: null,
      per_request_price: null,
      intervals: []
    },
    official_pricing: null
  }
}

export const domesticCatalogFamilies: PlazaDomesticFamily[] = [
  {
    platform: 'deepseek', label: 'DeepSeek', monogram: 'DS',
    models: [plannedModel('deepseek-v4.1-flash', 'deepseek', 0.3, 1.2, 0.006)]
  },
  {
    platform: 'zhipu', label: 'GLM', monogram: 'GL',
    models: [
      plannedModel('glm-5.1', 'zhipu', 1.4, 4.4, 0.26),
      plannedModel('glm-5.2', 'zhipu', 1.4, 4.4, 0.26),
      plannedModel('glm-5.3', 'zhipu', 1.4, 4.4, 0.26),
      plannedModel('glm-5.3-flash', 'zhipu', 0.15, 0.5, 0.03),
      plannedModel('glm-5v-turbo', 'zhipu', 1.2, 4, 0.24)
    ]
  },
  {
    platform: 'hunyuan', label: '腾讯混元', monogram: 'HY',
    models: [
      plannedModel('hy3', 'hunyuan', 0.132, 0.528, 0.033),
      plannedModel('hy4-preview', 'hunyuan', 0.834, 2.501, 0.042)
    ]
  },
  {
    platform: 'kimi', label: 'Kimi', monogram: 'KM',
    models: [
      plannedModel('kimi-k2.6', 'kimi', 0.95, 4, 0.16),
      plannedModel('kimi-k2.7', 'kimi', 0.95, 4, 0.19),
      plannedModel('kimi-k3', 'kimi', 3, 15, 0.3)
    ]
  },
  {
    platform: 'minimax', label: 'MiniMax', monogram: 'MM',
    models: [plannedModel('minimax-m3', 'minimax', 0.3, 1.2, 0.06)]
  }
]
