import { Sparkles } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import type { ExtractedTask, MeetingOutcome } from '../../api/types'

export function LogMeetingOutcomeDialog({
  open,
  onOpenChange,
  outcome,
  occurredOn,
  boardId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  outcome?: MeetingOutcome
  occurredOn: string
  boardId: number
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
    if (open) {
      setTitle(outcome?.title ?? '')
      setNotes(outcome?.notes_md ?? '')
      setEndsAt(outcome?.ends_at ? outcome.ends_at.slice(0, 16) : '')
      setSavedOutcome(outcome)
      setSuggestions([])
    }
  }, [open, outcome])

  async function save(): Promise<MeetingOutcome | undefined> {
    if (!title.trim()) return undefined
    const body = {
      title: title.trim(),
      notes_md: notes,
      ends_at: endsAt ? new Date(endsAt).toISOString() : undefined,
    }
    try {
      const saved = savedOutcome
        ? await updateOutcome.mutateAsync({ id: savedOutcome.id, ...body })
        : await createOutcome.mutateAsync({ occurred_on: occurredOn, ...body })
      setSavedOutcome(saved)
      toast.success('Meeting summary saved')
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
      for (const suggestion of selected) {
        await createTask.mutateAsync({
          title: suggestion.title,
          mark_done: suggestion.suggested_done,
          board_id: boardId,
          due_date: suggestion.due_date,
        })
      }
      toast.success(`Added ${selected.length} tasks`)
      setSuggestions([])
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not add tasks')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg rounded-lg">
        <DialogHeader>
          <DialogTitle>Meeting summary, {occurredOn}</DialogTitle>
          <DialogDescription>
            Save what happened here, then let AI turn the action items into tasks.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <Input
            placeholder="Meeting title"
            aria-label="Meeting title"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
          />
          <label className="flex flex-col gap-1 text-sm text-muted-foreground">
            Scheduled end (optional, for the conclusion prompt)
            <Input
              type="datetime-local"
              value={endsAt}
              onChange={(event) => setEndsAt(event.target.value)}
            />
          </label>
          <Textarea
            placeholder="What happened, decisions, action items…"
            aria-label="Meeting notes"
            value={notes}
            onChange={(event) => {
              setNotes(event.target.value)
              refs.handleChange(event.target.value, event.currentTarget)
            }}
            onKeyDown={refs.handleKeyDown}
            onBlur={refs.handleBlur}
            onSelect={refs.sync}
            rows={6}
          />
          <RefPicker picker={refs} />
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={() => void save()} disabled={!title.trim()}>
              Save summary
            </Button>
            {aiStatus?.configured && (
              <Button
                disabled={!title.trim() || !notes.trim() || extractTasks.isPending}
                onClick={() => void saveAndGenerate()}
              >
                <Sparkles />
                {extractTasks.isPending ? 'Generating…' : 'Save and generate tasks'}
              </Button>
            )}
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
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
