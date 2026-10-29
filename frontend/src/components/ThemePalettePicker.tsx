import { cn } from '@/lib/utils'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { ThemePalette } from '../lib/themes'

export function PaletteSwatches({
  colors,
  className,
}: {
  colors: string[]
  className?: string
}) {
  return (
    <span className={cn('inline-flex shrink-0 overflow-hidden rounded-[3px] border', className)}>
      {colors.slice(0, 6).map((color, index) => (
        <span key={`${color}-${index}`} className="h-3 w-3" style={{ backgroundColor: color }} />
      ))}
    </span>
  )
}

export function PaletteSelect({
  value,
  palettes,
  onChange,
  inheritLabel,
  ariaLabel,
  disabled,
}: {
  value: string
  palettes: ThemePalette[]
  onChange: (value: string) => void
  inheritLabel?: string
  ariaLabel?: string
  disabled?: boolean
}) {
  const items = [
    ...(inheritLabel ? [{ value: 'inherit', label: inheritLabel }] : []),
    ...palettes.map((palette) => ({ value: palette.id, label: palette.name })),
  ]

  return (
    <Select
      items={items}
      value={value}
      disabled={disabled}
      onValueChange={(next) => next && onChange(next)}
    >
      <SelectTrigger className="w-full" aria-label={ariaLabel}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {inheritLabel && (
          <SelectItem value="inherit">
            <span className="text-muted-foreground">{inheritLabel}</span>
          </SelectItem>
        )}
        {palettes.map((palette) => (
          <SelectItem key={palette.id} value={palette.id}>
            <PaletteSwatches colors={palette.colors} />
            <span className="min-w-0 truncate">{palette.name}</span>
            {palette.custom && (
              <span className="ml-auto text-[0.65rem] text-muted-foreground">Custom</span>
            )}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
