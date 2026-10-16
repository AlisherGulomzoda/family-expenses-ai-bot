import { CATEGORIES, CATEGORY_CODES, isCategoryCode } from '@/shared/categories'

describe('categories', () => {
  it('follow the API order and Russian labels', () => {
    // Same order and names as GET /api/categories (backend expense.Categories).
    expect(CATEGORY_CODES).toEqual(['food', 'transport', 'housing', 'entertainment', 'health', 'shopping', 'other'])
    expect(CATEGORY_CODES.map((c) => CATEGORIES[c].label)).toEqual([
      'Еда',
      'Транспорт',
      'Жильё',
      'Развлечения',
      'Здоровье',
      'Покупки',
      'Другое',
    ])
  })

  it('use chart-1…7 in order, each with its own color', () => {
    CATEGORY_CODES.forEach((code, i) => {
      const n = i + 1
      expect(CATEGORIES[code].chart).toBe(`var(--chart-${n})`)
      expect(CATEGORIES[code].badge).toBe(`bg-chart-${n}-soft text-chart-${n}-ink`)
      expect(CATEGORIES[code].fill).toBe(`bg-chart-${n}`)
    })
  })

  it('validates codes', () => {
    expect(isCategoryCode('entertainment')).toBe(true)
    expect(isCategoryCode('fun')).toBe(false)
    expect(isCategoryCode('toString')).toBe(false)
  })
})
