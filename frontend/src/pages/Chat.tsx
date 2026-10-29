import { Send } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'
import { Textarea } from '@/components/ui/textarea'
import { RefPicker } from '../components/refs/RefPicker'
import { useRefPicker } from '../components/refs/useRefPicker'
import { useAIChat, useAIStatus, useChatHistory, useConfirmChatAction, type AIChatMessage, type PendingChatAction } from '../api/hooks'
import { useWorkspaceStore } from '../store/workspace'

export function Chat() {
  const { data: status } = useAIStatus()
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId)
  const { data: history } = useChatHistory(workspaceId)
  const chat = useAIChat()
  const confirmAction = useConfirmChatAction()
  const [messages, setMessages] = useState<AIChatMessage[]>([])
  const [pending, setPending] = useState<PendingChatAction | null>(null)
  const [draft, setDraft] = useState('')
  const transcriptRef = useRef<HTMLDivElement>(null)
  const refs = useRefPicker({ value: draft, onChange: setDraft })

  useEffect(() => {
    const node = transcriptRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [messages, chat.isPending])

  useEffect(() => {
    if (!history?.length) return
    setMessages(
      history.map((message) => ({
        role: message.role,
        content: message.content,
      })),
    )
  }, [history])

  async function send() {
    const content = draft.trim()
    if (!content || chat.isPending) return
    const next: AIChatMessage[] = [...messages, { role: 'user' as const, content }].slice(-10)
    setMessages(next)
    setDraft('')
    try {
      const result = await chat.mutateAsync({ messages: next, workspaceId })
      setMessages((current) => [...current, { role: 'assistant' as const, content: result.reply }].slice(-10))
      setPending(result.pending ?? null)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'The assistant did not answer')
    }
  }

  if (status && !status.configured) {
    return (
      <div className="flex max-w-xl flex-col gap-2">
        <h1 className="text-lg font-semibold">Chat</h1>
        <p className="text-sm text-muted-foreground">
          AI is not configured. Add a provider in Settings to use chat.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-3">
      <header className="flex flex-col gap-1">
        <h1 className="text-lg font-semibold">Chat</h1>
        <p className="text-xs text-muted-foreground">
          Ask about your tasks, notes, or what to do next. It can also create tasks and reminders,
          for example "Create a task to refresh the Cloudflare token on Wednesday". The assistant
          sees open task titles and note titles.
        </p>
      </header>

      <div
        ref={transcriptRef}
        role="log"
        aria-live="polite"
        aria-label="Conversation"
        className="flex h-[55dvh] flex-col gap-3 overflow-y-auto rounded-md border p-3"
      >
        {messages.length === 0 ? (
          <div className="flex flex-1 flex-col items-start justify-center gap-1 text-sm">
            <p className="font-medium">No messages yet.</p>
            <p className="text-muted-foreground">
              Ask what to work on next, or how to break a note into tasks.
            </p>
          </div>
        ) : (
          messages.map((message, index) => (
            <div
              key={index}
              className={
                message.role === 'user'
                  ? 'ml-auto max-w-[85%] rounded-md border bg-card px-3 py-2 text-sm'
                  : 'mr-auto max-w-[85%] rounded-md bg-muted/50 px-3 py-2 text-sm'
              }
            >
              <p className="mb-1 text-[0.65rem] font-medium tracking-wide text-muted-foreground uppercase">
                {message.role === 'user' ? 'You' : 'Fai'}
              </p>
              <MarkdownRenderer source={message.content} />
            </div>
          ))
        )}
        {chat.isPending && (
          <p role="status" className="mr-auto text-sm text-muted-foreground">
            Thinking…
          </p>
        )}
        {pending && (
          <div className="mr-auto flex max-w-[85%] flex-col gap-2 rounded-md border bg-muted/50 px-3 py-2 text-sm">
            <p>
              Move <span className="font-mono text-xs">{pending.task_ref}</span> (
              {pending.task_title}) to {pending.column_name}?
            </p>
            <div className="flex gap-2">
              <Button
                size="sm"
                disabled={confirmAction.isPending}
                onClick={async () => {
                  try {
                    const result = await confirmAction.mutateAsync(pending)
                    setMessages((current) =>
                      [...current, { role: 'assistant' as const, content: result.reply }].slice(-10),
                    )
                    setPending(null)
                  } catch (error) {
                    toast.error(error instanceof Error ? error.message : 'Could not confirm')
                  }
                }}
              >
                Confirm
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setPending(null)}>
                Cancel
              </Button>
            </div>
          </div>
        )}
      </div>

      <div className="flex flex-col gap-2">
        <Textarea
          value={draft}
          rows={3}
          aria-label="Message"
          placeholder="Ask Fai about your tasks or notes"
          onChange={(event) => {
            setDraft(event.target.value)
            refs.handleChange(event.target.value, event.currentTarget)
          }}
          onKeyDown={(event) => {
            if (refs.handleKeyDown(event)) return
            if (event.key !== 'Enter' || event.shiftKey || event.nativeEvent.isComposing) return
            event.preventDefault()
            void send()
          }}
          onBlur={refs.handleBlur}
          onSelect={refs.sync}
        />
        <RefPicker picker={refs} />
        <div className="flex items-center justify-between gap-2">
          <p className="text-xs text-muted-foreground">Enter sends. Shift and Enter adds a line.</p>
          <Button disabled={!draft.trim() || chat.isPending} onClick={() => void send()}>
            <Send />
            Send
          </Button>
        </div>
      </div>
    </div>
  )
}
