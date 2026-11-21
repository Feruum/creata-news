import { createFileRoute } from '@tanstack/react-router'
import { useCallback, useState } from 'react'
import { useDropzone } from 'react-dropzone'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { agentApi } from '@/shared/lib/agent-api'
import { Upload, FileText, X, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { cn } from '@/shared/lib/utils'

interface FileWithStatus {
  file: File
  status: 'pending' | 'uploading' | 'success' | 'error'
  error?: string
  chunks?: number
}

function UploadPage() {
  const [files, setFiles] = useState<FileWithStatus[]>([])
  const queryClient = useQueryClient()

  const onDrop = useCallback((accepted: File[]) => {
    const newFiles = accepted.map((file) => ({ file, status: 'pending' as const }))
    setFiles((prev) => [...prev, ...newFiles])
  }, [])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/pdf': ['.pdf'],
      'text/plain': ['.txt'],
    },
    maxSize: 50 * 1024 * 1024,
  })

  const uploadMutation = useMutation({
    mutationFn: async () => {
      const pending = files.filter((f) => f.status === 'pending')
      for (let i = 0; i < pending.length; i++) {
        const item = pending[i]
        setFiles((prev) =>
          prev.map((f) => (f.file === item.file ? { ...f, status: 'uploading' } : f))
        )
        try {
          const result = await agentApi.uploadDocument(item.file)
          setFiles((prev) =>
            prev.map((f) =>
              f.file === item.file ? { ...f, status: 'success', chunks: result.chunks } : f
            )
          )
        } catch (err) {
          setFiles((prev) =>
            prev.map((f) =>
              f.file === item.file
                ? { ...f, status: 'error', error: err instanceof Error ? err.message : 'Upload failed' }
                : f
            )
          )
        }
      }
      queryClient.invalidateQueries({ queryKey: ['documents'] })
      toast.success('Upload complete')
    },
  })

  const removeFile = (file: File) => {
    setFiles((prev) => prev.filter((f) => f.file !== file))
  }

  const pendingCount = files.filter((f) => f.status === 'pending').length
  const isUploading = uploadMutation.isPending

  return (
    <div className="space-y-6 max-w-2xl">
      <div
        {...getRootProps()}
        className={cn(
          'flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-10 transition-colors cursor-pointer',
          isDragActive ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50 hover:bg-muted/30'
        )}
      >
        <input {...getInputProps()} />
        <Upload className={cn('h-10 w-10 mb-3', isDragActive ? 'text-primary' : 'text-muted-foreground')} />
        <p className="text-sm font-medium">
          {isDragActive ? 'Drop files here' : 'Drag & drop files, or click to browse'}
        </p>
        <p className="text-xs text-muted-foreground mt-1">PDF and TXT files, up to 50MB each</p>
      </div>

      {files.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">{files.length} file{files.length !== 1 ? 's' : ''}</p>
            {pendingCount > 0 && (
              <button
                onClick={() => uploadMutation.mutate()}
                disabled={isUploading}
                className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
              >
                {isUploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
                {isUploading ? 'Uploading...' : `Upload ${pendingCount} file${pendingCount !== 1 ? 's' : ''}`}
              </button>
            )}
          </div>

          <div className="rounded-lg border divide-y">
            {files.map((item, i) => (
              <div key={`${item.file.name}-${i}`} className="flex items-center gap-3 px-4 py-3">
                <FileText className="h-4 w-4 shrink-0 text-muted-foreground" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{item.file.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {(item.file.size / 1024).toFixed(1)} KB
                    {item.chunks != null && ` · ${item.chunks} chunks`}
                    {item.error && ` · ${item.error}`}
                  </p>
                </div>
                {item.status === 'pending' && (
                  <button onClick={() => removeFile(item.file)} className="text-muted-foreground hover:text-foreground">
                    <X className="h-4 w-4" />
                  </button>
                )}
                {item.status === 'uploading' && <Loader2 className="h-4 w-4 animate-spin text-primary" />}
                {item.status === 'success' && <CheckCircle2 className="h-4 w-4 text-green-500" />}
                {item.status === 'error' && <AlertCircle className="h-4 w-4 text-destructive" />}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/upload')({
  component: UploadPage,
})
