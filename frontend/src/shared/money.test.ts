import { formatRub, kopecksToRubles, parseRubInput, rublesToInput } from '@/shared/money'

const NBSP = ' '

describe('formatRub', () => {
  it.each([
    [350, `350${NBSP}₽`],
    [24350, `24${NBSP}350${NBSP}₽`],
    [350.5, `350,50${NBSP}₽`],
    [349.9, `349,90${NBSP}₽`],
    [1250000.5, `1${NBSP}250${NBSP}000,50${NBSP}₽`],
    [0, `0${NBSP}₽`],
    [0.01, `0,01${NBSP}₽`],
  ])('%s → %s', (amount, want) => {
    expect(formatRub(amount)).toBe(want)
  })

  it('never uses the narrow no-break space that Onest lacks', () => {
    expect(formatRub(1234567.89)).not.toContain(' ')
  })
})

describe('parseRubInput', () => {
  it.each([
    ['350', 35000],
    ['349,90', 34990],
    ['349.9', 34990],
    ['1 200', 120000],
    [`1${NBSP}250${NBSP}000,50`, 125000050],
    [' 0,01 ', 1],
    ['0', 0],
  ])('%j → %d kopecks', (text, want) => {
    expect(parseRubInput(text)).toBe(want)
  })

  it.each(['', 'abc', '-350', '350,555', '3.5e2', '1,2,3', '350р', ','])('rejects %j', (text) => {
    expect(parseRubInput(text)).toBeNull()
  })
})

describe('conversions', () => {
  it('round-trips kopecks, API rubles and input text', () => {
    expect(kopecksToRubles(34990)).toBe(349.9)
    expect(rublesToInput(349.9)).toBe('349,90')
    expect(rublesToInput(350)).toBe('350')
    expect(rublesToInput(0.1 + 0.2)).toBe('0,30') // float noise is rounded away
  })
})
