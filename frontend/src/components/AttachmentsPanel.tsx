import { Download, Paperclip, Trash2 } from 'lucide-react'
import { useRef } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { api } from '../api/client'
import {
  useDeleteAttachment,
  useNoteAttachments,
  useTaskAttachments,
  useUploadNoteAttachment,
  useUploadTaskAttachment,
} from '../api/hooks'
import { formatBytes } from '../lib/format'

export function AttachmentsPanel({ taskId, noteId }: { taskId?: number; noteId?: number }) {
  const taskAttachments = useTaskAttachments(taskId)
  const noteAttachments = useNoteAttachments(noteId)
  const attachments = (taskId !== undefined ? taskAttachments.data : noteAttachments.data) ?? []
  const uploadTask = useUploadTaskAttachment(taskId)
  const uploadNote = useUploadNoteAttachment(noteId)
  const remove = useDeleteAttachment()
  const inputRef = useRef<HTMLInputElement>(null)
  const uploading = uploadTask.isPending || uploadNote.isPending

  function upload(files: FileList | null) {
    if (!files?.length) return
    for (const file of Array.from(files)) {
      const options = { onError: (err: Error) => toast.error(err.message) }
      if (taskId !== undefined) uploadTask.mutate(file, options)
      else uploadNote.mutate(file, options)
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-muted-foreground">Attachments</h3>
        <Button size="sm" variant="outline" disabled={uploading} onClick={() => inputRef.current?.click()}>
          <Paperclip />
          {uploading ? 'Uploading…' : 'Upload'}
        </Button>
        <input
          ref={inputRef}
          type="file"
          multiple
          hidden
          onChange={(e) => {
            upload(e.target.files)
            e.target.value = ''
          }}
        />
      </div>

      {attachments.map((a) => (
        <div key={a.id} className="flex items-center gap-2 rounded-lg border px-2 py-1.5 text-sm">
          <Paperclip className="size-4 shrink-0 text-muted-foreground" />
          <span className="min-w-0 truncate" title={a.filename}>
            {a.filename}
          </span>
          <span className="ml-auto shrink-0 text-xs text-muted-foreground">{formatBytes(a.size_bytes)}</span>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={`Download ${a.filename}`}
            onClick={() =>
              api.download(`/attachments/${a.id}`, a.filename).catch((err) => toast.error(err.message))
            }
          >
            <Download />
          </Button>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={`Delete ${a.filename}`}
            onClick={() =>
              remove.mutate(a.id, { onError: (err) => toast.error(err.message) })
            }
          >
            <Trash2 />
          </Button>
        </div>
      ))}

      {!attachments.length && (
        <p className="text-xs text-muted-foreground">No attachments yet. Files up to 25 MiB.</p>
      )}
    </div>
  )
}
