interface CustomMenuTarget {
  url: string
  page_slug?: string
}

export function resolveCustomMenuRoute(item: CustomMenuTarget, origin: string): string | null {
  if (item.page_slug || !item.url.trim()) return null
  try {
    const url = new URL(item.url, origin)
    if (url.origin !== origin || url.username || url.password) return null
    // Only promote built-in pages; other menu URLs retain their embed behavior.
    if (url.pathname !== '/national-day-2026' && url.pathname !== '/national-day-2026/') return null
    return '/national-day-2026' + url.search + url.hash
  } catch {
    return null
  }
}
