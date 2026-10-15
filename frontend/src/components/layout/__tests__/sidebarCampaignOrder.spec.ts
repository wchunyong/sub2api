import { describe, expect, it } from 'vitest'
import { insertCampaignsAfterCanvas } from '../sidebarCampaignOrder'

const item = (label: string, page_slug?: string) => ({ label, page_slug })

describe('user sidebar campaign order', () => {
  it('places both campaigns immediately after the Infinite Canvas custom menu', () => {
    const result = insertCampaignsAfterCanvas(
      [item('模型广场'), item('仪表盘')],
      [item('文档'), item('无限画布'), item('其它工具')],
      [item('新用户权益'), item('十元七天乐')],
    )
    expect(result.map(({ label }) => label)).toEqual([
      '模型广场', '仪表盘', '文档', '无限画布', '新用户权益', '十元七天乐', '其它工具',
    ])
  })

  it('keeps the welcome benefit directly below Infinite Canvas when the other campaign is absent', () => {
    const result = insertCampaignsAfterCanvas([], [item('Canvas', 'infinite-canvas')], [item('新用户权益')])
    expect(result.map(({ label }) => label)).toEqual(['Canvas', '新用户权益'])
  })

  it('puts campaigns after custom menus when Infinite Canvas is not configured', () => {
    const result = insertCampaignsAfterCanvas([item('仪表盘')], [item('文档')], [item('新用户权益')])
    expect(result.map(({ label }) => label)).toEqual(['仪表盘', '文档', '新用户权益'])
  })
})
