/** First letter for an avatar: "Алишер" -> "А", "@madina" -> "M". */
export function initials(name: string): string {
  return name.replace(/^@/, '').trim().charAt(0).toUpperCase() || '?'
}
