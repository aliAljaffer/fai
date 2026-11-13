import { NavLink } from 'react-router-dom'
import { useViewTransition } from './useViewTransition'
import { AppIcon } from '@/components/Logo'
import { ThemeToggle } from '@/components/ThemeToggle'
import { TimerWidget } from '@/components/TimerWidget'
import { UpdateIndicator } from '@/components/UpdateIndicator'
import { useNotesCount } from '@/api/hooks'
import { appVersion, isDesktop } from '@/lib/updates'
import { cn } from '@/lib/utils'
import { useWorkspaceStore } from '@/store/workspace'
import { NAV_ITEMS, SETTINGS_ITEM } from './nav'
import { SidebarLabels } from './SidebarLabels'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'

export function Sidebar() {
  const viewTransition = useViewTransition()
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: notesCount } = useNotesCount(workspaceId)
  const counts: Record<string, number | undefined> = { '/notes': notesCount }
  const version = appVersion()
  const versionLabel = /^\d/.test(version) ? `v${version}` : version
  return (
    <aside className="sticky top-0 hidden h-dvh w-[244px] shrink-0 flex-col gap-[22px] border-r border-sidebar-border bg-sidebar px-4 py-5 text-sidebar-foreground lg:flex">
      <NavLink
        viewTransition={viewTransition}
        to="/"
        end
        aria-label="fai home"
        className="flex items-center gap-2.5 self-start rounded-sm px-1.5 pb-1 outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
      >
        <AppIcon className="size-[30px]" />
        <span className="text-[1.3rem] leading-none font-semibold tracking-tight">fai</span>
      </NavLink>
      <WorkspaceSwitcher className="h-auto w-full rounded-xl border-sidebar-border px-2.5 py-2 text-sm" />
      <div className="flex min-h-0 flex-1 flex-col gap-[22px] overflow-y-auto">
      <nav aria-label="Primary" className="flex flex-col gap-0.5">
        {NAV_ITEMS.map((item) => (
          <NavLink
            viewTransition={viewTransition}
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2.5 rounded-[9px] px-2.5 py-2 text-sm text-muted-foreground outline-none transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-ring/50',
                isActive &&
                  'bg-sidebar-accent text-sidebar-foreground shadow-[inset_0_0_0_1px_var(--sidebar-border)]',
              )
            }
          >
            <item.icon aria-hidden className="size-4" strokeWidth={1.5} />
            {item.label}
            {counts[item.to] !== undefined && (
              <span className="ml-auto font-mono text-[0.6875rem] tabular-nums">{counts[item.to]}</span>
            )}
          </NavLink>
        ))}
      </nav>
      <SidebarLabels />
      </div>
      <div className="flex flex-col gap-[22px]">
        <TimerWidget variant="panel" />
        <div className="flex items-center gap-1.5 border-t border-sidebar-border px-1.5 pt-2">
          <UpdateIndicator />
          <span className="min-w-0 flex-1 truncate font-mono text-[0.6875rem] text-muted-foreground">
            {versionLabel} · {isDesktop() ? 'desktop' : 'self-hosted'}
          </span>
          <ThemeToggle />
          <NavLink
            viewTransition={viewTransition}
            to={SETTINGS_ITEM.to}
            aria-label={SETTINGS_ITEM.label}
            title={SETTINGS_ITEM.label}
            className={({ isActive }) =>
              cn(
                'grid size-[30px] place-items-center rounded-full border border-sidebar-border text-muted-foreground outline-none hover:bg-sidebar-accent hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-ring/50',
                isActive && 'bg-sidebar-accent text-sidebar-foreground',
              )
            }
          >
            <SETTINGS_ITEM.icon aria-hidden className="size-3.5" strokeWidth={1.5} />
          </NavLink>
        </div>
      </div>
    </aside>
  )
}
