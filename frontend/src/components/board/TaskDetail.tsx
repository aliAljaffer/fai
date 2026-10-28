import MDEditor from '@uiw/react-md-editor'
import { Play, Square } from 'lucide-react'
import { useTheme } from 'next-themes'
import { useRef, useState } from 'react'
import { useHotkeys } from 'react-hotkeys-hook'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Sheet, SheetContent, SheetFooter, SheetHeader } from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { AttachmentsPanel } from '../AttachmentsPanel'
import { AIPanel } from './AIPanel'
import { JiraPanel } from './JiraPanel'
import { LabelEditor } from './LabelEditor'
import { SubtasksPanel } from './SubtasksPanel'
import { MarkdownRenderer } from '../MarkdownRenderer'
import { useRunningTime, useStartTime, useStopTime, useTaskTimeEntries } from '../../api/hooks'
import { formatDuration } from '../../lib/format'
import type { Task } from '../../api/types'

export function TaskDetail({
  task,
  onClose,
  onSave,
  onDelete,
}: {
  task: Task
  onClose: () => void
  onSave: (fields: { title: string; description_md: string; priority: string }) => void
  onDelete: () => void
}) {
  const [title, setTitle] = useState(task.title)
  const [description, setDescription] = useState(task.description_md)
  const [priority, setPriority] = useState(task.priority)
  const titleRef = useRef<HTMLInputElement>(null)
  const { resolvedTheme } = useTheme()

  useHotkeys('e', () => titleRef.current?.focus(), { preventDefault: true })

  const { data: running } = useRunningTime()
  const { data: entries } = useTaskTimeEntries(task.id)
  const startTime = useStartTime()
  const stopTime = useStopTime()
  const isRunning = running?.task_id === task.id
  const totalSeconds = (entries ?? []).reduce((sum, e) => sum + (e.duration_seconds ?? 0), 0)
  const sessionSeconds = isRunning
    ? Math.max(0, Math.floor((Date.now() - new Date(running.started_at).getTime()) / 1000))
    : 0

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full gap-4 overflow-y-auto p-4 sm:max-w-xl">
        <SheetHeader className="px-0">
          <div className="flex items-center gap-2">
            <input
              ref={titleRef}
              className="flex-1 border-b bg-transparent pb-2 text-lg font-medium outline-none"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
            <Button
              size="icon-sm"
              variant={isRunning ? 'destructive' : 'outline'}
              title={isRunning ? 'Stop timer' : 'Start timer'}
              aria-label={isRunning ? 'Stop timer' : 'Start timer'}
              onClick={() => (isRunning ? stopTime.mutate(task.id) : startTime.mutate(task.id))}
            >
              {isRunning ? <Square className="fill-current" /> : <Play />}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            {formatDuration(totalSeconds)} tracked
            {isRunning && ` · ${formatDuration(sessionSeconds)} this session`}
          </p>
        </SheetHeader>

        <Tabs defaultValue="details">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="subtasks">Subtasks</TabsTrigger>
            <TabsTrigger value="time">Time</TabsTrigger>
            <TabsTrigger value="files">Files</TabsTrigger>
            <TabsTrigger value="ai">AI</TabsTrigger>
          </TabsList>

          <TabsContent value="details" className="flex flex-col gap-4">
            <MDEditor
              value={description}
              onChange={(v) => setDescription(v ?? '')}
              height={220}
              preview="edit"
              data-color-mode={resolvedTheme === "light" ? "light" : "dark"}
            />
            {description && (
              <div className="rounded-lg border p-3">
                <MarkdownRenderer source={description} />
              </div>
            )}
            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-muted-foreground">Labels</span>
              <LabelEditor task={task} />
            </div>
            <Select value={priority} onValueChange={(value) => value && setPriority(value)}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="low">low</SelectItem>
                <SelectItem value="medium">medium</SelectItem>
                <SelectItem value="high">high</SelectItem>
              </SelectContent>
            </Select>
            <JiraPanel taskId={task.id} />
          </TabsContent>

          <TabsContent value="subtasks">
            <SubtasksPanel task={task} />
          </TabsContent>

          <TabsContent value="time" className="flex items-center gap-3">
            <Button
              size="sm"
              variant={isRunning ? 'destructive' : 'outline'}
              onClick={() => (isRunning ? stopTime.mutate(task.id) : startTime.mutate(task.id))}
            >
              {isRunning ? 'Stop timer' : 'Start timer'}
            </Button>
            <span className="text-sm text-muted-foreground">{formatDuration(totalSeconds)} tracked</span>
          </TabsContent>

          <TabsContent value="files">
            <AttachmentsPanel taskId={task.id} />
          </TabsContent>

          <TabsContent value="ai">
            <AIPanel task={task} />
          </TabsContent>
        </Tabs>

        <SheetFooter className="mt-auto flex-row justify-start px-0">
          <Button
            onClick={() => {
              onSave({ title, description_md: description, priority })
              onClose()
            }}
          >
            Save
          </Button>
          <Button variant="destructive" onClick={onDelete}>
            Delete
          </Button>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
