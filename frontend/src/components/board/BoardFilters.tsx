import type { CSSProperties } from 'react'
import { ListFilter, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { labelChipStyle } from '@/lib/palette'
import { cn } from '@/lib/utils'

const PRIORITY_FILTERS = [
  { value: 'all', label: 'Any priority' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
]

export function BoardFilters({
  priority,
  query,
  label,
  labelColor,
  onPriorityChange,
  onQueryChange,
  onClearLabel,
  onClearAll,
}: {
  priority: string
  query: string
  label: string | null
  labelColor: string
  onPriorityChange: (priority: string) => void
  onQueryChange: (query: string) => void
  onClearLabel: () => void
  onClearAll: () => void
}) {
  const active = priority !== 'all' || Boolean(label) || query.trim().length > 0
  const chipStyle: CSSProperties = {
    ...labelChipStyle(labelColor),
    color: `color-mix(in oklch, ${labelColor} 68%, var(--foreground))`,
  }

  return (
    <>
      <div
        role="group"
        aria-label="Filter by priority"
        className="flex items-center rounded-full border p-0.5"
      >
        {PRIORITY_FILTERS.map((item) => (
          <button
            key={item.value}
            type="button"
            aria-pressed={priority === item.value}
            className={cn(
              'rounded-full px-3 py-1 text-[0.8125rem] text-muted-foreground outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50',
              priority === item.value && 'bg-muted text-foreground',
            )}
            onClick={() => onPriorityChange(item.value)}
          >
            {item.label}
          </button>
        ))}
      </div>
      {label && (
        <span
          className="inline-flex h-7 items-center gap-1.5 rounded-full border pr-1 pl-2.5 text-xs font-semibold"
          style={chipStyle}
        >
          <span className="font-normal opacity-70">Label</span>
          {label}
          <button
            type="button"
            aria-label={`Remove label filter ${label}`}
            className="grid size-5 place-items-center rounded-full outline-none hover:bg-foreground/10 focus-visible:ring-2 focus-visible:ring-ring/50"
            onClick={onClearLabel}
          >
            <X aria-hidden className="size-3" />
          </button>
        </span>
      )}
      {active && (
        <button
          type="button"
          className="rounded-sm px-1 text-xs text-muted-foreground outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50"
          onClick={onClearAll}
        >
          Clear filters
        </button>
      )}
      <span className="flex-1" />
      <div className="relative">
        <ListFilter
          aria-hidden
          className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground"
        />
        <Input
          value={query}
          placeholder="Filter this board"
          aria-label="Filter this board"
          className="w-52 pl-7"
          onChange={(event) => onQueryChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Escape') onQueryChange('')
          }}
        />
      </div>
    </>
  )
}
