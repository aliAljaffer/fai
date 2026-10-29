import MDEditor from '@uiw/react-md-editor'
import { useTheme } from 'next-themes'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { AttachmentsPanel } from '../components/AttachmentsPanel'
import { NoteAIPanel } from '../components/notes/NoteAIPanel'
import { MarkdownRenderer } from '../components/MarkdownRenderer'
import { RefPicker } from '../components/refs/RefPicker'
import { useRefPicker } from '../components/refs/useRefPicker'
import { useDeleteNote, useNote, useUpdateNote } from '../api/hooks'
import { toast } from 'sonner'

export function NoteEditor() {
  const { id } = useParams()
  const noteId = Number(id)
  const navigate = useNavigate()
  const { data } = useNote(noteId)
  const updateNote = useUpdateNote()
  const deleteNote = useDeleteNote()
  const { resolvedTheme } = useTheme()

  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const refs = useRefPicker({ value: body, onChange: setBody })

  useEffect(() => {
    if (data?.note) {
      setTitle(data.note.title)
      setBody(data.note.body_md)
    }
  }, [data?.note])

  async function save() {
    setSaving(true)
    try {
      await updateNote.mutateAsync({ id: noteId, title, body_md: body })
      navigate('/notes')
    } catch (error) {
      toast.error((error as Error).message)
      setSaving(false)
    }
  }

  async function remove() {
    if (deleting) return
    setDeleting(true)
    try {
      await deleteNote.mutateAsync(noteId)
      navigate('/notes')
    } catch (error) {
      toast.error((error as Error).message)
      setDeleting(false)
    }
  }

  if (!data) return null

  return (
    <div className="flex flex-col gap-4">
      <h1 className="sr-only">Edit note</h1>
      <div className="flex items-center justify-between">
        <input
          aria-label="Note title"
          className="border-b bg-transparent text-xl font-medium outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <div className="flex gap-2">
          <Button disabled={saving} onClick={() => void save()}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
          <Button
            variant="destructive"
            disabled={deleting}
            onClick={() => void remove()}
          >
            {deleting ? 'Deleting…' : 'Delete'}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <MDEditor
          value={body}
          onChange={(value, event) => {
            setBody(value ?? '')
            refs.handleChange(value ?? '', event?.target ?? null)
          }}
          textareaProps={{
            'aria-label': 'Note body',
            onKeyDown: refs.handleKeyDown,
            onBlur: refs.handleBlur,
            onSelect: refs.sync,
          }}
          height={400}
          preview="edit"
          data-color-mode={resolvedTheme === "light" ? "light" : "dark"}
        />
        <RefPicker picker={refs} />
        <div className="rounded-lg border p-4">
          <MarkdownRenderer source={body} />
        </div>
      </div>

      <NoteAIPanel noteId={noteId} title={title} body={body} />

      <AttachmentsPanel noteId={noteId} />

      {data.backlinks.length > 0 && (
        <div>
          <h2 className="mb-2 text-sm font-medium text-muted-foreground">Linked from</h2>
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
