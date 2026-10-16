import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { PERIOD_PRESETS, type PeriodPreset } from '@/shared/dates'

/** Period preset chips of the phone layout: in the filter row and in the drawer. */
export function PresetChips({
  value,
  onChange,
  label,
}: {
  value: string
  onChange: (p: PeriodPreset) => void
  label: string
}) {
  return (
    <ToggleGroup
      type="single"
      variant="outline"
      spacing={2}
      value={value}
      onValueChange={(v) => v && onChange(v as PeriodPreset)}
      className="w-max"
      aria-label={label}
    >
      {PERIOD_PRESETS.map((p) => (
        <ToggleGroupItem
          key={p.id}
          value={p.id}
          className="h-11 rounded-full px-4 whitespace-nowrap data-[state=on]:bg-foreground data-[state=on]:text-background"
        >
          {p.label}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  )
}
