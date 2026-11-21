import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { agentApi, type DocumentSource } from '@/shared/lib/agent-api'
import { Trash2, FileText, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

function DocumentsPage() {
  const queryClient = useQueryClient()

  const { data: documents, isLoading, error } = useQuery({
    queryKey: ['documents'],
    queryFn: agentApi.listDocuments,
  })

  const deleteMutation = useMutation({
    mutationFn: agentApi.deleteDocument,
    onSuccess: (_, source) => {
      queryClient.invalidateQueries({ queryKey: ['documents'] })
      toast.success(`Deleted "${source}"`)
    },
    onError: (err) => toast.error(err.message),
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/5 p-4 text-sm text-destructive">
        Failed to load documents: {error.message}
      </div>
    )
  }

  if (!documents?.length) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <FileText className="h-12 w-12 text-muted-foreground/50 mb-3" />
        <p className="text-muted-foreground">No documents in the knowledge base yet.</p>
        <p className="text-sm text-muted-foreground/70 mt-1">Upload PDFs or text files to get started.</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {documents.length} document source{documents.length !== 1 ? 's' : ''}
        </p>
      </div>
      <div className="rounded-lg border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left font-medium">Source</th>
              <th className="px-4 py-3 text-left font-medium">Chunks</th>
              <th className="px-4 py-3 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {documents.map((doc: DocumentSource) => (
              <tr key={doc.source} className="border-b last:border-0 hover:bg-muted/30 transition-colors">
                <td className="px-4 py-3 font-medium">{doc.source}</td>
                <td className="px-4 py-3 text-muted-foreground">{doc.count}</td>
                <td className="px-4 py-3 text-right">
                  <button
                    onClick={() => {
                      if (confirm(`Delete "${doc.source}" and all its chunks?`)) {
                        deleteMutation.mutate(doc.source)
                      }
                    }}
                    disabled={deleteMutation.isPending}
                    className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/documents')({
  component: DocumentsPage,
})
