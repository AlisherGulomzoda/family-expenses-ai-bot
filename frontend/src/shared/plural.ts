/**
 * Picks the Russian word form for a number: plural(5, 'трата', 'траты', 'трат') -> 'трат'.
 * one: 1, 21, 101; few: 2–4, 22–24; many: 0, 5–20, 25–30, 11–14.
 */
export function plural(n: number, one: string, few: string, many: string): string {
  const abs = Math.abs(n)
  const mod10 = abs % 10
  const mod100 = abs % 100
  if (mod10 === 1 && mod100 !== 11) return one
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return few
  return many
}
