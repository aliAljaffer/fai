import { cn } from '@/lib/utils'
import { labelChipStyle } from '../lib/palette'
import type { Tag } from '../api/types'

export function TagChip({
  tag,
  className,
  children,
}: {
  tag: Tag
  className?: string
  children?: React.ReactNode
}) {
  const color = tag.color
  const base = labelChipStyle(color)
  const style =
    base && color
      ? {
          ...base,
          color: `color-mix(in oklch, ${color} 68%, var(--foreground))`,
        }
      : undefined

  return (
    <span className={cn('inline-flex items-center gap-1 border', className)} style={style}>
      <span className="size-1.5 rounded-full" style={{ backgroundColor: tag.color || undefined }} />
      {tag.name}
      {children}
    </span>
  )
}
