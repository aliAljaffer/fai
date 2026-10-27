import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useBoards, useJiraImport } from '../api/hooks'

export function Settings() {
  const { data: boards } = useBoards()
  const jiraImport = useJiraImport()
  const [jql, setJql] = useState('')
  const [forceUpdate, setForceUpdate] = useState(false)

  const boardId = boards?.[0]?.id

  return (
    <div className="flex flex-col gap-6">
      <Card className="max-w-xl">
        <CardHeader>
          <CardTitle>Jira import</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground">
            Imports issues from Jira as tasks on the first board. Leave the JQL field blank to use
            the server-configured default project. Already-imported issues are skipped unless
            "force update" is checked.
          </p>
          <Textarea
            placeholder='Optional JQL, e.g. project = "DEV" AND status != Done'
            value={jql}
            onChange={(e) => setJql(e.target.value)}
            rows={3}
          />
          <label className="flex items-center gap-2 text-sm">
            <Input
              type="checkbox"
              className="h-4 w-4"
              checked={forceUpdate}
              onChange={(e) => setForceUpdate(e.target.checked)}
            />
            Force update already-imported issues
          </label>
          <Button
            disabled={!boardId || jiraImport.isPending}
            onClick={() => {
              if (!boardId) return
              jiraImport.mutate(
                { board_id: boardId, jql: jql || undefined, force_update: forceUpdate },
                {
                  onSuccess: (result) =>
                    toast.success(
                      `Imported ${result.imported}, updated ${result.updated}, skipped ${result.skipped}`,
                    ),
                  onError: (err) => toast.error(err.message),
                },
              )
            }}
          >
            {jiraImport.isPending ? 'Importing…' : 'Import from Jira'}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
