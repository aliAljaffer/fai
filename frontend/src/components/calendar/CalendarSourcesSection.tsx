import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  useCalendarSources,
  useCreateCalendarSource,
  useDeleteCalendarSource,
  useUpdateCalendarSourceColor,
} from '../../api/hooks'
import { useWorkspaceStore } from '../../store/workspace'
import { STATUS_COLORS } from '../../lib/palette'

export function CalendarSourcesSection() {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: sources, isError, refetch } = useCalendarSources(workspaceId)
  const createSource = useCreateCalendarSource(workspaceId)
  const deleteSource = useDeleteCalendarSource()
  const updateColor = useUpdateCalendarSourceColor()
  const [name, setName] = useState('')
  const [url, setURL] = useState('')
  const [color, setColor] = useState(STATUS_COLORS.todo)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  async function submit() {
    const trimmedName = name.trim()
    const trimmedURL = url.trim()
    if (!trimmedName || !trimmedURL || createSource.isPending) return
    try {
      await createSource.mutateAsync({ name: trimmedName, url: trimmedURL, color })
      setName('')
      setURL('')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not import calendar')
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Calendars</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">
          Sync an Outlook or other iCalendar feed into Fai with its https (or webcal) link. Events
          refresh in the background and stay out of Fai's own calendar export.
        </p>

        <div className="flex flex-col gap-2">
          {isError && (
            <div className="flex items-center justify-between gap-2 rounded-md border border-destructive/40 p-2 text-sm">
              <span>Could not load imported calendars.</span>
              <Button size="sm" variant="outline" onClick={() => refetch()}>
                Retry
              </Button>
            </div>
          )}
          {!sources?.length && !isError && (
            <p className="text-sm text-muted-foreground">No imported calendars yet.</p>
          )}
          {sources?.map((source) => (
            <div key={source.id} className="flex flex-col gap-1 rounded-md border px-2 py-1.5">
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  defaultValue={source.color}
                  aria-label={`Event color for ${source.name}`}
                  title="Event color"
                  className="h-6 w-6 shrink-0 cursor-pointer rounded border border-input bg-transparent p-0.5"
                  onChange={(event) =>
                    updateColor.mutate(
                      { id: source.id, color: event.target.value },
                      { onError: (error) => toast.error(error.message) },
                    )
                  }
                />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{source.name}</p>
                  <p className="truncate text-xs text-muted-foreground">{source.url}</p>
                </div>
                <Button
                  size="icon-xs"
                  variant="ghost"
                  aria-label={`Remove ${source.name}`}
                  disabled={deleteSource.isPending}
                  onClick={() => setConfirmDelete(source.id)}
                >
                  <Trash2 />
                </Button>
              </div>
              {confirmDelete === source.id && (
                <div className="flex items-center justify-end gap-2 border-t pt-1 text-xs">
                  <span>Remove this feed?</span>
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => setConfirmDelete(null)}
                  >
                    Cancel
                  </Button>
                  <Button
                    size="xs"
                    variant="destructive"
                    disabled={deleteSource.isPending}
                    onClick={() =>
                      deleteSource.mutate(source.id, {
                        onSuccess: () => setConfirmDelete(null),
                        onError: (error) => toast.error(error.message),
                      })
                    }
                  >
                    Remove
                  </Button>
                </div>
              )}
            </div>
          ))}
        </div>

        <div className="grid gap-2 border-t pt-3">
          <p className="text-xs font-medium text-muted-foreground">Add calendar</p>
          <Input
            value={name}
            placeholder="Name, for example Outlook"
            aria-label="Calendar name"
            onChange={(event) => setName(event.target.value)}
          />
          <Input
            value={url}
            placeholder="https://example.com/calendar.ics"
            aria-label="Calendar feed URL"
            onChange={(event) => setURL(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                void submit()
              }
            }}
          />
          <p className="text-xs text-muted-foreground">
            Outlook on the web: Settings &gt; Calendar &gt; Shared calendars &gt; Publish a
            calendar, then copy the ICS or webcal link.
          </p>
          <div className="flex items-center gap-2">
            <input
              type="color"
              value={color}
              aria-label="Event color"
              className="h-8 w-10 cursor-pointer rounded border border-input bg-transparent p-0.5"
              onChange={(event) => setColor(event.target.value)}
            />
            <Button
              size="sm"
              className="ml-auto"
              disabled={!name.trim() || !url.trim() || createSource.isPending}
              onClick={() => void submit()}
            >
              <Plus />
              Import calendar
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
