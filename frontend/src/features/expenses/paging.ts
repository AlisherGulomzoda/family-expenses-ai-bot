/**
 * Page links to show: the first, the last and the neighbours of the current
 * page, with gaps in between: 1 … 4 5 6 … 12. Short lists are shown in full.
 */
export function pageItems(current: number, pages: number): (number | 'gap')[] {
  if (pages <= 7) return Array.from({ length: pages }, (_, i) => i + 1)
  const keep = new Set([1, pages, current - 1, current, current + 1])
  if (current <= 3) [2, 3, 4].forEach((n) => keep.add(n))
  if (current >= pages - 2) [pages - 3, pages - 2, pages - 1].forEach((n) => keep.add(n))
  const shown = [...keep].filter((n) => n >= 1 && n <= pages).sort((a, b) => a - b)
  const out: (number | 'gap')[] = []
  shown.forEach((n, i) => {
    const prev = shown[i - 1]
    if (prev !== undefined && n - prev > 1) out.push(n - prev === 2 ? n - 1 : 'gap')
    out.push(n)
  })
  return out
}
