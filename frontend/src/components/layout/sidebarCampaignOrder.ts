interface CustomSidebarItem {
  label: string
  page_slug?: string
}

/** Keep campaign links immediately below the configured Infinite Canvas menu. */
export function insertCampaignsAfterCanvas<T extends CustomSidebarItem>(
  fixedItems: T[],
  customItems: T[],
  campaignItems: T[],
): T[] {
  const canvasIndex = customItems.findIndex(item =>
    item.label.trim() === '无限画布' || item.page_slug?.trim().toLowerCase() === 'infinite-canvas',
  )
  const insertAt = canvasIndex < 0 ? customItems.length : canvasIndex + 1
  return [
    ...fixedItems,
    ...customItems.slice(0, insertAt),
    ...campaignItems,
    ...customItems.slice(insertAt),
  ]
}
