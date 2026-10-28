import { useNavigate } from 'react-router-dom'
import rehypeHighlight from 'rehype-highlight'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { wikilinksToMarkdownLinks } from '../lib/wikilinks'
import { useNoteByTitle } from '../api/hooks'

export function MarkdownRenderer({ source }: { source: string }) {
  const navigate = useNavigate()

  return (
    <div className="prose prose-sm max-w-none dark:prose-invert">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          a: ({ href, children }) => {
            if (href?.startsWith('wikilink:')) {
              const title = decodeURIComponent(href.replace('wikilink:', ''))
              return <WikiLink title={title} onNavigate={(id) => navigate(`/notes/${id}`)}>{children}</WikiLink>
            }
            return (
              <a href={href} target="_blank" rel="noreferrer">
                {children}
              </a>
            )
          },
        }}
      >
        {wikilinksToMarkdownLinks(source)}
      </ReactMarkdown>
    </div>
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
