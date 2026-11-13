import { useLayoutEffect, useMemo, useRef, useState, type CSSProperties } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { X } from 'lucide-react'
import { useViewTransition } from './useViewTransition'
import { useTags } from '@/api/hooks'
import { useWorkspaceStore } from '@/store/workspace'
import { TagChip } from '@/components/TagChip'
import { activeLabel, filtersInPlace, labelFilterLink } from '@/lib/labelFilter'
import { tagColor } from '@/lib/palette'
import { cn } from '@/lib/utils'

// Five rows of 22px chips with 6px gaps, plus 4px of list padding.
const COLLAPSED_HEIGHT = 138

export function SidebarLabels() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: tags } = useTags(workspaceId)
  const location = useLocation()
  const viewTransition = useViewTransition()
  const listRef = useRef<HTMLUListElement>(null)
  const [showAll, setShowAll] = useState(false)
  const [overflows, setOverflows] = useState(false)
  const inPlace = filtersInPlace(location)
  const selected = activeLabel(location)

  const ordered = useMemo(() => {
    const all = tags ?? []
    if (!selected) return all
    return [...all].sort((a, b) => Number(b.name === selected) - Number(a.name === selected))
  }, [tags, selected])

  useLayoutEffect(() => {
    const list = listRef.current
    if (list) setOverflows(list.scrollHeight > COLLAPSED_HEIGHT)
  }, [ordered])

  if (!ordered.length) return null

  return (
    <section aria-label="Labels" className="flex shrink-0 flex-col gap-2.5">
      <div className="flex items-center justify-between px-2.5">
        <h2 className="eyebrow">Labels</h2>
        {selected && (
          <Link
            replace
            to={labelFilterLink(location, null)}
            className="flex items-center gap-1 rounded-sm text-xs text-muted-foreground outline-none hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-ring/50"
          >
            Clear
            <X aria-hidden className="size-3" />
          </Link>
        )}
      </div>
      <ul
        ref={listRef}
        style={showAll ? undefined : { maxHeight: COLLAPSED_HEIGHT }}
        className="flex flex-wrap gap-1.5 overflow-hidden px-1.5 py-0.5"
      >
        {ordered.map((tag) => {
          const isSelected = tag.name === selected
          return (
            <li key={tag.id} className="flex">
              <Link
                replace={inPlace}
                viewTransition={!inPlace && viewTransition}
                to={labelFilterLink(location, isSelected ? null : tag.name)}
                aria-current={isSelected ? 'true' : undefined}
                style={{ '--label': tagColor(tag) } as CSSProperties}
                className={cn(
                  'flex rounded-full outline-none transition-opacity focus-visible:ring-2 focus-visible:ring-ring/50',
                  selected && !isSelected && 'opacity-40 hover:opacity-100 focus-visible:opacity-100',
                  isSelected && 'ring-2 ring-(--label)/30',
                )}
              >
                <TagChip
                  tag={tag}
                  className="h-[22px] rounded-full px-[9px] text-xs leading-none font-semibold"
                />
              </Link>
            </li>
          )
        })}
      </ul>
      {overflows && (
        <button
          type="button"
          aria-expanded={showAll}
          className="self-start rounded-sm px-2.5 text-xs font-medium text-muted-foreground outline-none hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-ring/50"
          onClick={() => setShowAll((current) => !current)}
        >
          {showAll ? 'Show fewer' : `Show all ${ordered.length}`}
        </button>
      )}
    </section>
  )
}
