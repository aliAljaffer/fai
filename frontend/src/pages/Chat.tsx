import { Send } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { useAIChat, useAIStatus, type AIChatMessage } from '../api/hooks'

export function Chat() {
  const { data: status } = useAIStatus()
  const chat = useAIChat()
  const [messages, setMessages] = useState<AIChatMessage[]>([])
  const [draft, setDraft] = useState('')
  const transcriptRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const node = transcriptRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [messages, chat.isPending])

  async function send() {
    const content = draft.trim()
    if (!content || chat.isPending) return
    const next: AIChatMessage[] = [...messages, { role: 'user' as const, content }].slice(-10)
    setMessages(next)
    setDraft('')
    try {
      const result = await chat.mutateAsync(next)
      setMessages((current) => [...current, { role: 'assistant' as const, content: result.reply }].slice(-10))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'The assistant did not answer')
    }
  }

  if (status && !status.configured) {
    return (
      <div className="flex max-w-xl flex-col gap-2">
        <h1 className="text-lg font-semibold">Chat</h1>
        <p className="text-sm text-muted-foreground">AI is not configured on this server.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-3">
      <header className="flex flex-col gap-1">
        <h1 className="text-lg font-semibold">Chat</h1>
        <p className="text-xs text-muted-foreground">
          Ask about your tasks, notes, or what to do next. The assistant sees open task titles and
          note titles.
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
              <p className="whitespace-pre-wrap">{message.content}</p>
            </div>
          ))
        )}
        {chat.isPending && (
          <p role="status" className="mr-auto text-sm text-muted-foreground">
            Thinking…
          </p>
        )}
      </div>

      <div className="flex flex-col gap-2">
        <Textarea
          value={draft}
          rows={3}
          aria-label="Message"
          placeholder="Ask Fai about your tasks or notes"
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key !== 'Enter' || event.shiftKey || event.nativeEvent.isComposing) return
            event.preventDefault()
            void send()
          }}
        />
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
