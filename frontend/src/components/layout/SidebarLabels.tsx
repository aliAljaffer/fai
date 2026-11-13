import { Link, useLocation } from 'react-router-dom'
import { useViewTransition } from './useViewTransition'
import { useTags } from '@/api/hooks'
import { TagChip } from '@/components/TagChip'
import { cn } from '@/lib/utils'

export function SidebarLabels() {
  const { data: tags } = useTags()
  const location = useLocation()
  const viewTransition = useViewTransition()
  const activeTag =
    location.pathname === '/list' ? new URLSearchParams(location.search).get('tag') : null

  if (!tags?.length) return null

  return (
    <section aria-label="Labels" className="flex min-h-0 flex-col gap-2.5">
      <h2 className="px-2.5 eyebrow">
        Labels
      </h2>
      <ul className="flex min-h-0 flex-col gap-0.5 overflow-y-auto">
        {tags.map((tag) => (
          <li key={tag.id}>
            <Link
              viewTransition={viewTransition}
              to={`/list?tag=${encodeURIComponent(tag.name)}`}
              aria-current={activeTag === tag.name ? 'true' : undefined}
              className={cn(
                'flex rounded-[9px] px-2.5 py-2 outline-none hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-ring/50',
                activeTag === tag.name && 'bg-sidebar-accent',
              )}
            >
              <TagChip tag={tag} className="h-[22px] rounded-full px-[9px] text-xs leading-none font-semibold" />
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
