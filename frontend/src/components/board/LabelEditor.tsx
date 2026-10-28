import { X } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  useAddTaskTag,
  useCreateTag,
  useRemoveTaskTag,
  useTags,
} from '../../api/hooks'
import { TagChip } from '../TagChip'
import type { Tag, Task } from '../../api/types'
import { labelColorFor as colorFor } from '../../lib/palette'

export function LabelEditor({ task }: { task: Task }) {
  const { data: allTags } = useTags()
  const createTag = useCreateTag()
  const addTag = useAddTaskTag(task.id, task.board_id)
  const removeTag = useRemoveTaskTag(task.id, task.board_id)
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)

  const attachedIds = new Set((task.tags ?? []).map((tag) => tag.id))
  const trimmed = query.trim()
  const available = (allTags ?? []).filter(
    (tag) =>
      !attachedIds.has(tag.id) &&
      (!trimmed || tag.name.toLowerCase().includes(trimmed.toLowerCase())),
  )
  const exact = (allTags ?? []).find(
    (tag) => tag.name.toLowerCase() === trimmed.toLowerCase(),
  )

  async function attach(tag: Tag) {
    try {
      await addTag.mutateAsync(tag.id)
      setQuery('')
    } catch (error) {
      toast.error((error as Error).message)
    }
  }

  async function addFromQuery() {
    if (!trimmed) return
    try {
      if (exact) {
        if (!attachedIds.has(exact.id)) await attach(exact)
        return
      }
      const tag = await createTag.mutateAsync({ name: trimmed, color: colorFor(trimmed) })
      await attach(tag)
    } catch (error) {
      toast.error((error as Error).message)
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-1.5">
        {(task.tags ?? []).map((tag) => (
          <TagChip key={tag.id} tag={tag} className="rounded-full py-0.5 pr-1 pl-2 text-xs">
            <button
              className="rounded-full p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground"
              aria-label={`Remove ${tag.name}`}
              onClick={() => removeTag.mutate(tag.id)}
            >
              <X className="size-3" />
            </button>
          </TagChip>
        ))}
        {!(task.tags ?? []).length && <span className="text-xs text-muted-foreground">No labels</span>}
      </div>
      <div className="relative">
        <Input
          value={query}
          placeholder="Add label…"
          className="h-7 text-xs"
          onChange={(event) => {
            setQuery(event.target.value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          onBlur={() => setTimeout(() => setOpen(false), 150)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return
            if (event.key === 'Enter') {
              event.preventDefault()
              void addFromQuery()
            }
            if (event.key === 'Escape') setOpen(false)
          }}
        />
        {open && (available.length > 0 || trimmed) && (
          <div className="absolute z-20 mt-1 flex max-h-48 w-full flex-col overflow-y-auto rounded-md border bg-popover p-1 shadow-md">
            {available.slice(0, 8).map((tag) => (
              <button
                key={tag.id}
                className="flex items-center gap-2 rounded-sm px-2 py-1.5 text-left text-xs hover:bg-accent"
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => void attach(tag)}
              >
                <span
                  className="size-1.5 rounded-full"
                  style={{ backgroundColor: tag.color || undefined }}
                />
                {tag.name}
              </button>
            ))}
            {trimmed && !exact && (
              <button
                className={cn(
                  'flex items-center gap-2 rounded-sm px-2 py-1.5 text-left text-xs hover:bg-accent',
                  available.length > 0 && 'border-t',
                )}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => void addFromQuery()}
              >
                Create “{trimmed}”
              </button>
            )}
          </div>
        )}
      </div>
      {createTag.isPending && <span className="text-xs text-muted-foreground">Creating label…</span>}
    </div>
  )
}
