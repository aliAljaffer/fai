import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { Task } from '../../api/types'

const priorityVariant: Record<string, 'default' | 'secondary' | 'destructive'> = {
  high: 'destructive',
  medium: 'secondary',
  low: 'default',
}

export function TaskCard({ task, onClick }: { task: Task; onClick: () => void }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: task.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  return (
    <Card
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onClick={onClick}
      className={cn('cursor-grab gap-2 py-3 active:cursor-grabbing', isDragging && 'opacity-40')}
    >
      <CardContent className="px-3">
        <div className="text-sm">{task.title}</div>
        {task.priority !== 'medium' && (
          <Badge variant={priorityVariant[task.priority] ?? 'secondary'} className="mt-2">
            {task.priority}
          </Badge>
        )}
      </CardContent>
    </Card>
  )
}
