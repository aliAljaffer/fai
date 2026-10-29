import { cn } from '@/lib/utils'

export function StatusPill({
  label,
  color,
  className,
}: {
  label: string
  color?: string
  className?: string
}) {
  if (!color) {
    return <span className={cn('text-muted-foreground', className)}>{label}</span>
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[0.7rem] font-medium',
        className,
      )}
      style={{
        backgroundColor: `color-mix(in oklch, ${color} 14%, var(--background))`,
        borderColor: `color-mix(in oklch, ${color} 34%, var(--border))`,
        color: `color-mix(in oklch, ${color} 68%, var(--foreground))`,
      }}
    >
      <span className="size-1.5 shrink-0 rounded-full" style={{ backgroundColor: color }} />
      {label}
    </span>
  )
}
