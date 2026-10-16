import { applyPatch, hasDefaultFilters, parseView, serializeView } from './urlState'

// Friday 2026-10-16: "this month" is 2026-10-01 … 2026-10-31.
const now = new Date(2026, 9, 16, 10, 0, 0)

describe('parseView', () => {
  it('defaults to this month, newest first, page 1', () => {
    expect(parseView('', now)).toEqual({
      from: '2026-10-01',
      to: '2026-10-31',
      sort: 'spent_at',
      order: 'desc',
      page: 1,
    })
  })

  it('reads every parameter', () => {
    const v = parseView(
      '?user_id=100000002&from=2026-09-01&to=2026-09-30&category=food&sort=amount&order=asc&page=3',
      now,
    )
    expect(v).toEqual({
      userId: 100000002,
      from: '2026-09-01',
      to: '2026-09-30',
      category: 'food',
      sort: 'amount',
      order: 'asc',
      page: 3,
    })
  })

  it.each([
    ['?user_id=abc', {}],
    ['?category=fun', {}],
    ['?sort=id;drop&order=up', {}],
    ['?page=0', {}],
    ['?page=-2', {}],
    ['?from=2026-02-31&to=2026-03-05', {}], // not a real date
    ['?from=2026-10-20&to=2026-10-01', {}], // from after to
    ['?from=2026-10-05', {}], // half a period
    ['?from=15.10.2026&to=16.10.2026', {}],
  ])('ignores garbage %s', (search) => {
    expect(parseView(search, now)).toEqual(parseView('', now))
  })
})

describe('serializeView', () => {
  it('leaves defaults out of the URL', () => {
    expect(serializeView(parseView('', now), now)).toBe('')
  })

  it('round-trips a non-default view', () => {
    const search = '?user_id=7&from=2026-09-01&to=2026-09-30&category=health&sort=amount&order=asc&page=2'
    expect(serializeView(parseView(search, now), now)).toBe(search)
  })
})

describe('applyPatch', () => {
  const view = parseView('?category=food&page=4', now)

  it('goes back to page 1 when filters or sorting change', () => {
    expect(applyPatch(view, { userId: 1 }).page).toBe(1)
    expect(applyPatch(view, { sort: 'amount' }).page).toBe(1)
  })

  it('keeps an explicit page', () => {
    expect(applyPatch(view, { page: 5 }).page).toBe(5)
  })

  it('clears optional filters with undefined', () => {
    const next = applyPatch(view, { category: undefined })
    expect('category' in next).toBe(false)
    expect(hasDefaultFilters(next, now)).toBe(true)
  })

  it('detects non-default filters', () => {
    expect(hasDefaultFilters(view, now)).toBe(false)
    expect(hasDefaultFilters(parseView('?sort=amount&page=2', now), now)).toBe(true)
  })
})
