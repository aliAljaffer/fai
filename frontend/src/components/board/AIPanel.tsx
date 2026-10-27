import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { useAIBreakdown, useAIChat, useAIStatus } from '../../api/hooks'
import type { Task } from '../../api/types'

export function AIPanel({ task }: { task: Task }) {
  const { data: status } = useAIStatus()
  const breakdown = useAIBreakdown(task.id)
  const chat = useAIChat()
  const [prompt, setPrompt] = useState('')
  const [reply, setReply] = useState('')

  if (!status?.configured) {
    return <p className="text-sm text-muted-foreground">AI is not configured on this server.</p>
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <Button
          variant="outline"
          size="sm"
          disabled={breakdown.isPending}
          onClick={() =>
            breakdown.mutate(undefined, {
              onSuccess: (created) => toast.success(`Created ${created.length} subtasks`),
              onError: (err) => toast.error(err.message),
            })
          }
        >
          {breakdown.isPending ? 'Generating…' : 'Generate subtasks'}
        </Button>
      </div>
      <div className="flex flex-col gap-2">
        <Textarea
          placeholder="Ask about this task…"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          rows={3}
        />
        <Button
          variant="outline"
          size="sm"
          disabled={chat.isPending || !prompt}
          onClick={() =>
            chat.mutate(prompt, {
              onSuccess: (res) => setReply(res.reply),
              onError: (err) => toast.error(err.message),
            })
          }
        >
          {chat.isPending ? 'Thinking…' : 'Send'}
        </Button>
        {reply && <div className="rounded-md border p-3 text-sm whitespace-pre-wrap">{reply}</div>}
      </div>
    </div>
  )
}
