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
  return (
    <span
      className={cn('inline-flex items-center gap-1 border', className)}
      style={labelChipStyle(tag.color)}
    >
      <span className="size-1.5 rounded-full" style={{ backgroundColor: tag.color || undefined }} />
      {tag.name}
      {children}
    </span>
  )
}
