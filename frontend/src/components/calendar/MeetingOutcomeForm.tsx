import { format } from 'date-fns'
import { Sparkles, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { RefPicker } from '../refs/RefPicker'
import { useRefPicker } from '../refs/useRefPicker'
import {
  useAIStatus,
  useCreateMeetingOutcome,
  useCreateTaskFromOutcome,
  useExtractTasksFromOutcome,
  useUpdateMeetingOutcome,
} from '../../api/hooks'
import type { ExtractedTask, ImportedEvent, MeetingOutcome } from '../../api/types'

function localInput(value: string | null | undefined) {
  return value ? format(new Date(value), "yyyy-MM-dd'T'HH:mm") : ''
}

export function MeetingOutcomeForm({
  outcome,
  occurredOn,
  boardId,
  event,
  onClose,
}: {
  outcome?: MeetingOutcome
  occurredOn: string
  boardId: number
  event?: ImportedEvent
  onClose: () => void
}) {
  const { data: aiStatus } = useAIStatus()
  const createOutcome = useCreateMeetingOutcome()
  const updateOutcome = useUpdateMeetingOutcome()

  const [title, setTitle] = useState('')
  const [notes, setNotes] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [savedOutcome, setSavedOutcome] = useState<MeetingOutcome | undefined>(outcome)
  const [suggestions, setSuggestions] = useState<(ExtractedTask & { selected: boolean })[]>([])
  const refs = useRefPicker({ value: notes, onChange: setNotes })

  const extractTasks = useExtractTasksFromOutcome()
  const createTask = useCreateTaskFromOutcome(savedOutcome?.id ?? 0)

  useEffect(() => {
    // Opening from a calendar event fills the summary with the event's own data.
    setTitle(outcome?.title ?? event?.title ?? '')
    setNotes(outcome?.notes_md ?? '')
    setEndsAt(outcome?.ends_at ? localInput(outcome.ends_at) : event?.end && !event.all_day ? localInput(event.end) : '')
    setSavedOutcome(outcome)
    setSuggestions([])
  }, [outcome, event])

  async function save(): Promise<MeetingOutcome | undefined> {
    if (!title.trim()) return undefined
    const body = {
      title: title.trim(),
      notes_md: notes,
      ends_at: endsAt ? new Date(endsAt).toISOString() : undefined,
      calendar_source_id: event?.source_id,
      event_uid: event?.id,
      event_title: event?.title,
      event_url: event?.url,
    }
    try {
      const saved = savedOutcome
        ? await updateOutcome.mutateAsync({
            id: savedOutcome.id,
            title: body.title,
            notes_md: body.notes_md,
            ends_at: body.ends_at,
          })
        : await createOutcome.mutateAsync({ occurred_on: occurredOn, ...body })
      setSavedOutcome(saved)
      return saved
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not save meeting summary')
      return undefined
    }
  }

  async function generate(outcomeId: number) {
    try {
      const tasks = await extractTasks.mutateAsync(outcomeId)
      setSuggestions(tasks.map((task) => ({ ...task, selected: true })))
      if (!tasks.length) toast.info('No action items found in these notes')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not generate tasks')
    }
  }

  async function saveAndGenerate() {
    const saved = await save()
    if (saved) await generate(saved.id)
  }

  async function addSelected() {
    if (!savedOutcome) return
    const selected = suggestions.filter((suggestion) => suggestion.selected)
    if (!selected.length) return
    try {
      const arc = await createTask.mutateAsync({
        title: savedOutcome.title || 'Meeting summary',
        board_id: boardId,
        kind: 'arc',
      })
      for (const suggestion of selected) {
        await createTask.mutateAsync({
          title: suggestion.title,
          mark_done: suggestion.suggested_done,
          board_id: boardId,
          due_date: suggestion.due_date,
          kind: 'task',
          parent_task_id: arc.id,
          steps: suggestion.steps,
        })
      }
      toast.success(`Added ${selected.length} tasks to "${arc.title}"`)
      setSuggestions([])
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not add tasks')
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-md border bg-muted/30 p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold">Meeting summary, {occurredOn}</h3>
          <p className="text-xs text-muted-foreground">
            {event
              ? `From ${event.title}. Save what happened, then let AI turn action items into tasks.`
              : 'Save what happened, then let AI turn action items into tasks.'}
          </p>
        </div>
        <Button size="icon-sm" variant="ghost" aria-label="Close meeting summary" onClick={onClose}>
          <X />
        </Button>
      </div>
      <Input
        placeholder="Meeting title"
        aria-label="Meeting title"
        value={title}
        onChange={(changeEvent) => setTitle(changeEvent.target.value)}
      />
      <label className="flex flex-col gap-1 text-sm text-muted-foreground">
        Scheduled end (optional, for the conclusion prompt)
        <Input
          type="datetime-local"
          value={endsAt}
          onChange={(changeEvent) => setEndsAt(changeEvent.target.value)}
        />
      </label>
      <Textarea
        placeholder="What happened, decisions, action items…"
        aria-label="Meeting notes"
        value={notes}
        onChange={(changeEvent) => {
          setNotes(changeEvent.target.value)
          refs.handleChange(changeEvent.target.value, changeEvent.currentTarget)
        }}
        onKeyDown={refs.handleKeyDown}
        onBlur={refs.handleBlur}
        onSelect={refs.sync}
        rows={5}
      />
      <RefPicker picker={refs} />
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" onClick={() => void save()} disabled={!title.trim()}>
          Save summary
        </Button>
        {aiStatus?.configured && (
          <Button
            size="sm"
            disabled={!title.trim() || !notes.trim() || extractTasks.isPending}
            onClick={() => void saveAndGenerate()}
          >
            <Sparkles />
            {extractTasks.isPending ? 'Generating…' : 'Save and generate tasks'}
          </Button>
        )}
        <Button size="sm" variant="ghost" onClick={onClose}>
          Close
        </Button>
      </div>

      {savedOutcome && aiStatus?.configured && (
        <div className="flex flex-col gap-2 border-t pt-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={extractTasks.isPending || !notes.trim()}
              onClick={() => void generate(savedOutcome.id)}
            >
              <Sparkles />
              {extractTasks.isPending ? 'Analyzing…' : 'Generate tasks from notes'}
            </Button>
            <span className="text-xs text-muted-foreground">
              {suggestions.length ? `${suggestions.length} proposed tasks` : 'No tasks generated yet.'}
            </span>
          </div>
          {suggestions.map((suggestion, index) => (
            <div key={`${suggestion.title}-${index}`} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={suggestion.selected}
                onCheckedChange={(checked) =>
                  setSuggestions((current) =>
                    current.map((item, itemIndex) =>
                      itemIndex === index ? { ...item, selected: Boolean(checked) } : item,
                    ),
                  )
                }
              />
              <span className="flex-1">{suggestion.title}</span>
              {suggestion.due_date && (
                <span className="text-xs text-muted-foreground">{suggestion.due_date}</span>
              )}
              <Button
                size="sm"
                variant="ghost"
                onClick={() =>
                  setSuggestions((current) =>
                    current.map((item, itemIndex) =>
                      itemIndex === index
                        ? { ...item, suggested_done: !item.suggested_done }
                        : item,
                    ),
                  )
                }
              >
                {suggestion.suggested_done ? 'done' : 'open'}
              </Button>
            </div>
          ))}
          {suggestions.length > 0 && (
            <Button size="sm" onClick={() => void addSelected()}>
              Add selected tasks
            </Button>
          )}
        </div>
      )}
    </div>
  )
}
