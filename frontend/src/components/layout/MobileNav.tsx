import { NavLink } from 'react-router-dom'
import { useViewTransition } from './useViewTransition'
import { AppIcon } from '@/components/Logo'
import { ThemeToggle } from '@/components/ThemeToggle'
import { TimerWidget } from '@/components/TimerWidget'
import { UpdateIndicator } from '@/components/UpdateIndicator'
import { cn } from '@/lib/utils'
import { ALL_NAV_ITEMS } from './nav'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'

export function MobileTopBar() {
  const viewTransition = useViewTransition()
  return (
    <div className="flex items-center gap-2 border-b px-4 py-3 sm:px-6 lg:hidden">
      <NavLink
        viewTransition={viewTransition}
        to="/"
        end
        aria-label="fai home"
        className="rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
      >
        <AppIcon className="size-6" />
      </NavLink>
      <WorkspaceSwitcher className="max-w-44" />
      <div className="ml-auto flex items-center gap-1.5">
        <TimerWidget variant="compact" />
        <ThemeToggle />
        <UpdateIndicator />
      </div>
    </div>
  )
}

export function MobileTabBar() {
  const viewTransition = useViewTransition()
  return (
    <nav
      aria-label="Primary"
      className="fixed inset-x-0 bottom-0 z-40 flex overflow-x-auto border-t bg-background/95 backdrop-blur lg:hidden"
    >
      {ALL_NAV_ITEMS.map((item) => (
        <NavLink
          viewTransition={viewTransition}
          key={item.to}
          to={item.to}
          end={item.end}
          className={({ isActive }) =>
            cn(
              'flex min-w-16 flex-1 flex-col items-center justify-center gap-0.5 px-2 py-2 text-[0.65rem] text-muted-foreground',
              isActive && 'font-semibold text-foreground',
            )
          }
        >
          <item.icon aria-hidden className="size-4" />
          {item.label}
        </NavLink>
      ))}
    </nav>
  )
}
