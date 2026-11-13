import { Search } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { useUIStore } from '@/store/ui'
import { commandKey } from './nav'
import { useActiveWorkspace } from './useActiveWorkspace'

export function PageHeader({
  title,
  description,
  actions,
  className,
}: {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  className?: string
}) {
  const setCommandOpen = useUIStore((state) => state.setCommandOpen)
  const { active } = useActiveWorkspace()

  return (
    <header className={cn('flex flex-wrap items-end justify-between gap-x-[22px] gap-y-3', className)}>
      <div className="flex min-w-0 flex-col gap-1">
        <p className="eyebrow">
          Workspace{active ? ` / ${active.name}` : ''}
        </p>
        <h1 className="min-w-0 text-[2.6rem] leading-[1.1] font-normal tracking-tight">
          {title}
        </h1>
        {description && (
          <div className="mt-1 max-w-prose font-mono text-xs tracking-[0.02em] text-muted-foreground">
            {description}
          </div>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-2.5 [&_[data-slot=button]]:h-[38px] [&_[data-slot=button]]:rounded-full [&_[data-slot=button]]:px-4 [&_[data-slot=button]]:text-sm">
        <button
          type="button"
          aria-label="Open command palette"
          onClick={() => setCommandOpen(true)}
          className="flex h-[38px] items-center gap-2 rounded-full border bg-background px-3.5 text-sm text-muted-foreground outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 sm:w-[250px]"
        >
          <Search aria-hidden className="size-3.5" />
          <span className="hidden flex-1 truncate text-left sm:inline">Search</span>
          <kbd className="hidden rounded-md border px-1.5 py-0.5 font-mono text-[0.6875rem] sm:inline">{commandKey}</kbd>
        </button>
        {actions}
      </div>
    </header>
  )
}
