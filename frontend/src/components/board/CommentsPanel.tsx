import { Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { useCreateTaskComment, useDeleteTaskComment, useTaskComments } from '../../api/hooks'
import { MarkdownRenderer } from '../MarkdownRenderer'

export function CommentsPanel({ taskId }: { taskId: number }) {
  const { data: comments } = useTaskComments(taskId)
  const create = useCreateTaskComment(taskId)
  const remove = useDeleteTaskComment(taskId)
  const [body, setBody] = useState('')

  function add() {
    const bodyMd = body.trim()
    if (!bodyMd) return
    create.mutate(bodyMd, {
      onSuccess: () => setBody(''),
      onError: (error) => toast.error(error.message),
    })
  }

  return (
    <div className="flex flex-col gap-3">
      <Textarea
        aria-label="New comment"
        placeholder="Add a comment (Markdown)"
        value={body}
        onChange={(event) => setBody(event.target.value)}
        rows={3}
      />
      <div className="flex justify-end">
        <Button size="sm" disabled={!body.trim() || create.isPending} onClick={add}>
          {create.isPending ? 'Posting…' : 'Comment'}
        </Button>
      </div>

      {(comments ?? []).map((comment) => (
        <div key={comment.id} className="flex flex-col gap-1 rounded-lg border p-3">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span className="font-medium text-foreground">{comment.author || 'You'}</span>
            <span>{new Date(comment.created_at).toLocaleString()}</span>
            <Button
              size="icon-sm"
              variant="ghost"
              className="ml-auto"
              title="Delete comment"
              aria-label="Delete comment"
              onClick={() =>
                remove.mutate(comment.id, { onError: (error) => toast.error(error.message) })
              }
            >
              <Trash2 />
            </Button>
          </div>
          <MarkdownRenderer source={comment.body_md} />
        </div>
      ))}
    </div>
  )
}
