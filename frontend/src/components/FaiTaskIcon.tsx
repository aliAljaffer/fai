import { FaCat } from 'react-icons/fa'
import { cn } from '@/lib/utils'

export function FaiTaskIcon({
  className,
  size = 14,
}: {
  className?: string
  size?: number
}) {
  return (
    <span
      role="img"
      aria-label="Added by Fai"
      title="Added by Fai"
      className={cn('inline-flex shrink-0 items-center text-primary', className)}
    >
      <FaCat aria-hidden="true" size={size} />
    </span>
  )
}
