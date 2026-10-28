import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Play, Square } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { Task } from '../../api/types'

const priorityVariant: Record<string, 'default' | 'secondary' | 'destructive'> = {
  high: 'destructive',
  medium: 'secondary',
  low: 'default',
}

interface TaskCardProps {
  task: Task
  onClick?: () => void
  runningTaskId?: number | null
  onToggleTimer?: (task: Task) => void
  overlay?: boolean
}

function TaskCardBody({ task, runningTaskId, onToggleTimer, overlay }: TaskCardProps) {
  const running = runningTaskId === task.id

  return (
    <CardContent className="flex flex-col gap-2 px-3">
      <div className="text-sm leading-snug">{task.title}</div>
      {!!task.tags?.length && (
        <div className="flex flex-wrap gap-1">
          {task.tags.map((tag) => (
            <span
              key={tag.id}
              className="inline-flex items-center gap-1 rounded-full border px-1.5 py-0.5 text-[0.65rem] text-muted-foreground"
            >
              <span
                className="size-1.5 rounded-full"
                style={{ backgroundColor: tag.color || undefined }}
              />
              {tag.name}
            </span>
          ))}
        </div>
      )}
      <div className="flex items-center justify-between gap-2">
        {task.priority !== 'medium' ? (
          <Badge variant={priorityVariant[task.priority] ?? 'secondary'}>{task.priority}</Badge>
        ) : (
          <span />
        )}
        {!overlay && onToggleTimer && (
          <Button
            size="icon-xs"
            variant={running ? 'destructive' : 'ghost'}
            title={running ? 'Stop timer' : 'Start timer'}
            aria-label={running ? 'Stop timer' : 'Start timer'}
            className={cn('text-muted-foreground', running && 'text-destructive')}
            onClick={(event) => {
              event.stopPropagation()
              onToggleTimer(task)
            }}
            onPointerDown={(event) => event.stopPropagation()}
          >
            {running ? <Square className="fill-current" /> : <Play />}
          </Button>
        )}
      </div>
    </CardContent>
  )
}

function SortableTaskCard(props: TaskCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: props.task.id,
    data: { type: 'task' },
  })

  return (
    <Card
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      {...attributes}
      {...listeners}
      onClick={props.onClick}
      className={cn(
        'cursor-grab touch-none gap-2 py-3 select-none active:cursor-grabbing',
        isDragging && 'opacity-30',
      )}
    >
      <TaskCardBody {...props} />
    </Card>
  )
}

export function TaskCard(props: TaskCardProps) {
  if (props.overlay) {
    return (
      <Card className="rotate-2 gap-2 py-3 shadow-xl ring-2 ring-ring/40">
        <TaskCardBody {...props} overlay />
      </Card>
    )
  }
  return <SortableTaskCard {...props} />
}
