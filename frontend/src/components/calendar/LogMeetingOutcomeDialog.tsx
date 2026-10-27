import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
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

  const extractTasks = useExtractTasksFromOutcome(savedOutcome?.id ?? 0)
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

  async function handleSave() {
    const body = {
      title,
      notes_md: notes,
      ends_at: endsAt ? new Date(endsAt).toISOString() : undefined,
    }
    if (savedOutcome) {
      const updated = await updateOutcome.mutateAsync({ id: savedOutcome.id, ...body })
      setSavedOutcome(updated)
    } else {
      const created = await createOutcome.mutateAsync({ occurred_on: occurredOn, ...body })
      setSavedOutcome(created)
    }
    toast.success('Meeting outcome saved')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Log meeting outcome — {occurredOn}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <Input placeholder="Meeting title" value={title} onChange={(e) => setTitle(e.target.value)} />
          <label className="flex flex-col gap-1 text-sm text-muted-foreground">
            Scheduled end (optional, for the conclusion prompt)
            <Input type="datetime-local" value={endsAt} onChange={(e) => setEndsAt(e.target.value)} />
          </label>
          <Textarea
            placeholder="What happened / decisions / action items…"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={6}
          />
          <Button variant="outline" onClick={handleSave} disabled={!title}>
            Save notes
          </Button>

          {savedOutcome && aiStatus?.configured && (
            <div className="flex flex-col gap-2 border-t pt-3">
              <Button
                size="sm"
                variant="outline"
                disabled={extractTasks.isPending || !notes}
                onClick={() =>
                  extractTasks.mutate(undefined, {
                    onSuccess: (tasks) => setSuggestions(tasks.map((t) => ({ ...t, selected: true }))),
                    onError: (err) => toast.error(err.message),
                  })
                }
              >
                {extractTasks.isPending ? 'Analyzing…' : 'Generate tasks from notes'}
              </Button>
              {suggestions.map((s, i) => (
                <div key={i} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={s.selected}
                    onCheckedChange={(checked) =>
                      setSuggestions((prev) => prev.map((p, j) => (j === i ? { ...p, selected: !!checked } : p)))
                    }
                  />
                  <span className="flex-1">{s.title}</span>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() =>
                      setSuggestions((prev) =>
                        prev.map((p, j) => (j === i ? { ...p, suggested_done: !p.suggested_done } : p)),
                      )
                    }
                  >
                    {s.suggested_done ? 'done' : 'open'}
                  </Button>
                </div>
              ))}
              {suggestions.length > 0 && (
                <Button
                  size="sm"
                  onClick={async () => {
                    const selected = suggestions.filter((s) => s.selected)
                    for (const s of selected) {
                      await createTask.mutateAsync({ title: s.title, mark_done: s.suggested_done, board_id: boardId })
                    }
                    toast.success(`Added ${selected.length} tasks`)
                    setSuggestions([])
                  }}
                >
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
