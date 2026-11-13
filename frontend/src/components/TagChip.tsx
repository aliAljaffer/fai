import { cn } from '@/lib/utils'
import { labelChipStyle, tagColor } from '../lib/palette'
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
  const color = tagColor(tag)
  const style = {
    ...labelChipStyle(color),
    color: `color-mix(in oklch, ${color} 68%, var(--foreground))`,
  }

  return (
    <span className={cn('inline-flex items-center gap-1 border', className)} style={style}>
      <span className="size-1.5 rounded-full" style={{ backgroundColor: color }} />
      {tag.name}
      {children}
    </span>
  )
}
