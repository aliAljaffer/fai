import { useNavigate, useSearchParams } from 'react-router-dom'
import rehypeHighlight from 'rehype-highlight'
import ReactMarkdown, { defaultUrlTransform } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { wikilinksToMarkdownLinks } from '../lib/wikilinks'
import { splitTaskRefs, taskRefsToMarkdownLinks } from '../lib/taskrefs'
import { useNoteByTitle, useResolveRefs } from '../api/hooks'

export function MarkdownRenderer({ source }: { source: string }) {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()

  function openTask(id: number) {
    const next = new URLSearchParams(params)
    next.set('task', String(id))
    setParams(next, { replace: true })
  }

  return (
    <div className="prose prose-sm max-w-none dark:prose-invert">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        urlTransform={(url) =>
          url.startsWith('taskref:') || url.startsWith('wikilink:') ? url : defaultUrlTransform(url)
        }
        components={{
          a: ({ href, children }) => {
            if (href?.startsWith('wikilink:')) {
              const title = decodeURIComponent(href.replace('wikilink:', ''))
              return <WikiLink title={title} onNavigate={(id) => navigate(`/notes/${id}`)}>{children}</WikiLink>
            }
            if (href?.startsWith('taskref:')) {
              const ref = decodeURIComponent(href.replace('taskref:', ''))
              return <TaskRefLink taskRef={ref} onOpen={openTask}>{children}</TaskRefLink>
            }
            return (
              <a href={href} target="_blank" rel="noreferrer">
                {children}
              </a>
            )
          },
        }}
      >
        {wikilinksToMarkdownLinks(taskRefsToMarkdownLinks(source))}
      </ReactMarkdown>
    </div>
  )
}

export function TaskRefText({ text }: { text: string }) {
  const [params, setParams] = useSearchParams()

  function openTask(id: number) {
    const next = new URLSearchParams(params)
    next.set('task', String(id))
    setParams(next, { replace: true })
  }

  return (
    <>
      {splitTaskRefs(text).map((part, index) =>
        part.ref ? (
          <TaskRefLink key={index} taskRef={part.ref} onOpen={openTask}>
            {part.ref}
          </TaskRefLink>
        ) : (
          <span key={index}>{part.text}</span>
        ),
      )}
    </>
  )
}

function WikiLink({
  title,
  onNavigate,
  children,
}: {
  title: string
  onNavigate: (id: number) => void
  children: React.ReactNode
}) {
  const { data: note } = useNoteByTitle(title)
  return (
    <a
      className={note ? 'text-primary underline' : 'text-destructive underline decoration-dotted'}
      href="#"
      onClick={(e) => {
        e.preventDefault()
        if (note) onNavigate(note.id)
      }}
      title={note ? undefined : `"${title}" does not exist yet`}
    >
      {children}
    </a>
  )
}

function TaskRefLink({
  taskRef,
  onOpen,
  children,
}: {
  taskRef: string
  onOpen: (id: number) => void
  children: React.ReactNode
}) {
  const { data, isFetching } = useResolveRefs(taskRef)
  const number = Number(taskRef.split('-')[1])
  const match =
    data?.find((item) => item.ref.toLowerCase() === taskRef.toLowerCase()) ??
    data?.find((item) => item.ref_num === number)

  if (!match) {
    return (
      <span
        className={isFetching ? undefined : 'underline decoration-dotted'}
        title={isFetching ? undefined : `No task matches ${taskRef}`}
      >
        {children}
      </span>
    )
  }

  return (
    <a
      className="text-primary underline"
      href="#"
      title={match.title}
      onClick={(e) => {
        e.preventDefault()
        onOpen(match.id)
      }}
    >
      {children}
    </a>
  )
}
