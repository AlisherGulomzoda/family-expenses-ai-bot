// Money formatting per design-spec §1: "24 350 ₽", "1 250 000,50 ₽".
// Amounts come from the API as rubles (JSON number, at most 2 decimals).

const NBSP = ' '

const wholeFormat = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 0 })
const centsFormat = new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** Formats rubles: no decimals for whole amounts, two otherwise. */
export function formatRub(amount: number): string {
  const whole = Number.isInteger(Math.round(amount * 100) / 100)
  const text = (whole ? wholeFormat : centsFormat).format(amount)
  // Intl groups with U+202F (narrow no-break space), which Onest lacks.
  return text.replaceAll(' ', NBSP) + NBSP + '₽'
}

/**
 * Parses what a person types into the amount field: "349,90", "349.9",
 * "1 200". Returns kopecks, or null if the text is not a valid amount.
 * Works on the digits as text, so there are no floating point surprises.
 */
export function parseRubInput(text: string): number | null {
  const s = text.replace(/[\s  ]/g, '').replace(',', '.')
  const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(s)
  if (!match) return null
  const rubles = Number(match[1])
  const kopecks = Number((match[2] ?? '').padEnd(2, '0'))
  const total = rubles * 100 + kopecks
  return Number.isSafeInteger(total) ? total : null
}

/** Kopecks to the API's rubles: 34990 -> 349.9. */
export function kopecksToRubles(kopecks: number): number {
  return kopecks / 100
}

/** Rubles from the API to the amount field text: 349.9 -> "349,90". */
export function rublesToInput(amount: number): string {
  const kopecks = Math.round(amount * 100)
  const rubles = Math.trunc(kopecks / 100)
  const rest = kopecks % 100
  return rest === 0 ? String(rubles) : `${rubles},${String(rest).padStart(2, '0')}`
}
