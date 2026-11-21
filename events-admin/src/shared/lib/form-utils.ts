import type { FieldError, FieldValues, Path, UseFormReturn } from 'react-hook-form'
import { toast } from 'sonner'
import { getRequestErrorMessage, type RequestErrorContext } from './request-errors'

/**
 * Server-side field error structure
 */
export interface ServerFieldError {
  field: string
  message: string
}

/**
 * Server error response structure
 */
export interface ServerErrorResponse {
  message: string
  errors?: ServerFieldError[]
}

/**
 * Type guard for Connect-RPC errors
 */
function isConnectError(error: unknown): error is { code: string; message: string; metadata: Map<string, string> } {
  return typeof error === 'object' && error !== null && 'code' in error && 'message' in error && 'metadata' in error
}

/**
 * Extract error message from various error types
 */
export function getErrorMessage(error: unknown): string {
  return getRequestErrorMessage(error)
}

/**
 * Parse server errors and set them on form fields
 *
 * @example
 * ```tsx
 * const mutation = useMutation({
 *   mutationFn: createUser,
 *   onError: (error) => {
 *     handleServerErrors(form, error)
 *   },
 * })
 * ```
 */
export function handleServerErrors<T extends FieldValues>(
  form: Pick<UseFormReturn<T>, 'setError'>,
  error: unknown,
  options: {
    /** Show toast for root-level errors */
    showToast?: boolean
    /** Field name mapping from server to form */
    fieldMapping?: Record<string, Path<T>>
    context?: RequestErrorContext
  } = {}
): void {
  const { showToast = true, fieldMapping = {}, context } = options
  const setRootError = () => {
    const message = getRequestErrorMessage(error, context)
    form.setError('root', { type: 'server', message })
    if (showToast) toast.error(message)
  }

  // Handle Connect-RPC errors
  if (isConnectError(error)) {
    // Try to parse structured error details from metadata
    const details = error.metadata.get('error-details')
    if (details) {
      try {
        const parsed = JSON.parse(details) as ServerErrorResponse
        if (parsed.errors?.length) {
          setFieldErrors(form, parsed.errors, fieldMapping)
          return
        }
      } catch {
        // Fall through to generic handling
      }
    }

    // Show generic error
    setRootError()
    return
  }

  // Handle plain Error objects with potential field errors
  if (error instanceof Error) {
    // Check if error has a response property (fetch errors)
    const responseError = error as Error & { response?: { data?: ServerErrorResponse } }
    if (responseError.response?.data?.errors?.length) {
      setFieldErrors(form, responseError.response.data.errors, fieldMapping)
      return
    }

    setRootError()
    return
  }

  // Handle raw ServerErrorResponse
  if (isServerErrorResponse(error)) {
    if (error.errors) {
      setFieldErrors(form, error.errors, fieldMapping)
    }
    if (!error.errors?.length) setRootError()
    return
  }

  // Fallback
  setRootError()
}

function isServerErrorResponse(error: unknown): error is ServerErrorResponse {
  return (
    typeof error === 'object' &&
    error !== null &&
    'message' in error &&
    typeof (error as ServerErrorResponse).message === 'string'
  )
}

function setFieldErrors<T extends FieldValues>(
  form: Pick<UseFormReturn<T>, 'setError'>,
  errors: ServerFieldError[],
  fieldMapping: Record<string, Path<T>>
): void {
  for (const { field, message } of errors) {
    const formField = fieldMapping[field] || (field as Path<T>)
    form.setError(formField, {
      type: 'server',
      message
    })
  }
}

/**
 * Clear server errors when user starts typing
 * Use this in form field onChange handlers
 */
export function clearServerError<T extends FieldValues>(
  form: Pick<UseFormReturn<T>, 'formState' | 'clearErrors'>,
  fieldName: Path<T>
): void {
  const error = form.formState.errors[fieldName] as FieldError | undefined
  if (error?.type === 'server') {
    form.clearErrors(fieldName)
  }
}

/**
 * Create a submit handler with loading state management
 *
 * @example
 * ```tsx
 * const { handleSubmit, isSubmitting } = useSubmitHandler({
 *   form,
 *   onSubmit: async (data) => {
 *     await createEvent(data)
 *   },
 *   onSuccess: () => {
 *     toast.success('Event created!')
 *     onClose()
 *   },
 * })
 * ```
 */
export function createSubmitHandler<T extends FieldValues>(options: {
  form: UseFormReturn<T>
  onSubmit: (data: T) => Promise<void>
  onSuccess?: () => void
  onError?: (error: unknown) => void
  successMessage?: string
}) {
  const { form, onSubmit, onSuccess, onError, successMessage } = options

  return form.handleSubmit(async (data) => {
    try {
      await onSubmit(data)
      if (successMessage) {
        toast.success(successMessage)
      }
      onSuccess?.()
    } catch (error) {
      handleServerErrors(form, error)
      onError?.(error)
    }
  })
}

/**
 * Hook to track form dirty state for unsaved changes warning
 */
export function useFormDirtyWarning<T extends FieldValues>(form: UseFormReturn<T>, enabled = true): void {
  const { isDirty } = form.formState

  // Handle browser refresh/close
  if (typeof window !== 'undefined' && enabled && isDirty) {
    window.onbeforeunload = () => 'You have unsaved changes. Are you sure you want to leave?'
  } else if (typeof window !== 'undefined') {
    window.onbeforeunload = null
  }
}
