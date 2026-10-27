import { useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Sheet, SheetContent, SheetFooter, SheetHeader } from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'
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

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="gap-4 p-4">
        <SheetHeader className="px-0">
          <input
            className="border-b bg-transparent pb-2 text-lg font-medium outline-none"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
        </SheetHeader>
        <Textarea
          placeholder="Description (markdown)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={10}
        />
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
