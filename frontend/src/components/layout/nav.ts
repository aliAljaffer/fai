import {
  BarChart3,
  CalendarDays,
  FileText,
  Home,
  LayoutGrid,
  List,
  MessageSquare,
  Settings,
  type LucideIcon,
} from 'lucide-react'

export interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
}

export const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'Overview', icon: Home, end: true },
  { to: '/board', label: 'Board', icon: LayoutGrid },
  { to: '/list', label: 'List', icon: List },
  { to: '/calendar', label: 'Calendar', icon: CalendarDays },
  { to: '/notes', label: 'Notes', icon: FileText },
  { to: '/chat', label: 'Chat', icon: MessageSquare },
  { to: '/stats', label: 'Stats', icon: BarChart3 },
]

export const SETTINGS_ITEM: NavItem = { to: '/settings', label: 'Settings', icon: Settings }

export const ALL_NAV_ITEMS = [...NAV_ITEMS, SETTINGS_ITEM]

export function titleForPath(pathname: string) {
  if (pathname.startsWith('/notes/')) return 'Note'
  if (pathname.startsWith('/tasks/')) return 'Task'
  return ALL_NAV_ITEMS.find((item) => item.to === pathname)?.label
}

const isMac =
  typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.userAgent)

export const commandKey = isMac ? '⌘K' : 'Ctrl K'
