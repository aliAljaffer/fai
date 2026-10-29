import { Sparkles } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import {
  useAIStatus,
  useBoards,
  useCreateTaskFromNote,
  useNoteTaskPlan,
} from '../../api/hooks'
import { useUIStore } from '../../store/ui'
import { useWorkspaceStore } from '../../store/workspace'

export function NoteAIPanel({
  noteId,
  title,
  body,
}: {
  noteId: number
  title: string
  body: string
}) {
  const { data: status } = useAIStatus()
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: boards } = useBoards(workspaceId)
  const { selectedBoardId, setSelectedBoardId } = useUIStore()
  const board = boards?.find((item) => item.id === selectedBoardId) ?? boards?.[0]
  const generate = useNoteTaskPlan(noteId)
  const createTask = useCreateTaskFromNote(noteId)
  const navigate = useNavigate()

  const [draftTitle, setDraftTitle] = useState('')
  const [subtasks, setSubtasks] = useState<{ title: string; selected: boolean }[]>([])
  const [createdTaskId, setCreatedTaskId] = useState<number | null>(null)

  if (!status?.configured) return null

  async function plan() {
    try {
      const result = await generate.mutateAsync({ title, body_md: body })
      setDraftTitle(result.title)
      setSubtasks(result.subtasks.map((subtask) => ({ title: subtask, selected: true })))
      setCreatedTaskId(null)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not generate a task plan')
    }
  }

  async function create() {
    if (!board || !draftTitle.trim()) return
    const selected = subtasks.filter((subtask) => subtask.selected).map((subtask) => subtask.title)
    try {
      const result = await createTask.mutateAsync({
        board_id: board.id,
        title: draftTitle.trim(),
        subtasks: selected,
        note_body: body,
      })
      setCreatedTaskId(result.task.id)
      toast.success(`Created task with ${result.subtasks.length} steps`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not create task')
    }
  }

  return (
    <section className="flex flex-col gap-3 rounded-md border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-medium">AI task plan</h2>
          <p className="text-xs text-muted-foreground">
            Turn this note into one task and a short list of steps.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {subtasks.length > 0 && (
            <Button size="sm" variant="ghost" disabled={generate.isPending} onClick={() => void plan()}>
              Regenerate
            </Button>
          )}
          <Button size="sm" variant="outline" disabled={generate.isPending} onClick={() => void plan()}>
            <Sparkles />
            {generate.isPending ? 'Generating…' : 'Generate plan'}
          </Button>
        </div>
      </div>

      {subtasks.length > 0 && (
        <div className="flex flex-col gap-2">
          <Input
            value={draftTitle}
            aria-label="Task title"
            onChange={(event) => setDraftTitle(event.target.value)}
          />
          <ul className="flex flex-col gap-1.5">
            {subtasks.map((subtask, index) => (
              <li key={`${subtask.title}-${index}`} className="flex items-center gap-2 text-sm">
                <Checkbox
                  checked={subtask.selected}
                  onCheckedChange={(checked) =>
                    setSubtasks((current) =>
                      current.map((item, itemIndex) =>
                        itemIndex === index ? { ...item, selected: Boolean(checked) } : item,
                      ),
                    )
                  }
                />
                <span>{subtask.title}</span>
              </li>
            ))}
          </ul>
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              disabled={!board || !draftTitle.trim() || createTask.isPending}
              onClick={() => void create()}
            >
              Create task
            </Button>
            <span className="text-xs text-muted-foreground">
              {board ? `Creates on ${board.name}.` : 'Create a board first.'}
            </span>
            {createdTaskId && board && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  setSelectedBoardId(board.id)
                  navigate(`/?task=${createdTaskId}`)
                }}
              >
                Open task
              </Button>
            )}
          </div>
        </div>
      )}
    </section>
  )
}
