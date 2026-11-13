import type { ReactNode } from 'react'
import { JiraImportBar } from '@/components/JiraImportBar'
import { PendingOutcomesBanner } from '@/components/calendar/PendingOutcomesBanner'
import { MobileTabBar, MobileTopBar } from './MobileNav'
import { Sidebar } from './Sidebar'

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <MobileTopBar />
        <PendingOutcomesBanner />
        <JiraImportBar />
        <main
          id="main-content"
          tabIndex={-1}
          className="flex flex-1 flex-col p-4 pb-20 outline-none sm:p-6 lg:px-[30px] lg:pt-[22px] lg:pb-7"
        >
          {children}
        </main>
      </div>
      <MobileTabBar />
    </div>
  )
}
