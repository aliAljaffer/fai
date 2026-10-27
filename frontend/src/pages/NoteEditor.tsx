import MDEditor from '@uiw/react-md-editor'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { AttachmentsPanel } from '../components/AttachmentsPanel'
import { MarkdownRenderer } from '../components/MarkdownRenderer'
import { useDeleteNote, useNote, useUpdateNote } from '../api/hooks'

export function NoteEditor() {
  const { id } = useParams()
  const noteId = Number(id)
  const navigate = useNavigate()
  const { data } = useNote(noteId)
  const updateNote = useUpdateNote()
  const deleteNote = useDeleteNote()

  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')

  useEffect(() => {
    if (data?.note) {
      setTitle(data.note.title)
      setBody(data.note.body_md)
    }
  }, [data?.note])

  if (!data) return null

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <input
          className="border-b bg-transparent text-xl font-medium outline-none"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <div className="flex gap-2">
          <Button onClick={() => updateNote.mutate({ id: noteId, title, body_md: body })}>Save</Button>
          <Button
            variant="destructive"
            onClick={async () => {
              await deleteNote.mutateAsync(noteId)
              navigate('/notes')
            }}
          >
            Delete
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <MDEditor
          value={body}
          onChange={(v) => setBody(v ?? '')}
          height={400}
          preview="edit"
          data-color-mode="dark"
        />
        <div className="rounded-lg border p-4">
          <MarkdownRenderer source={body} />
        </div>
      </div>

      <AttachmentsPanel noteId={noteId} />

      {data.backlinks.length > 0 && (
        <div>
          <h3 className="mb-2 text-sm font-medium text-muted-foreground">Linked from</h3>
          <ul className="flex flex-col gap-1">
            {data.backlinks.map((b) => (
              <li key={b.id}>
                <button className="text-primary underline" onClick={() => navigate(`/notes/${b.id}`)}>
                  {b.title}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
