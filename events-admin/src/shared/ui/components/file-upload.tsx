import { Check, FileIcon, ImageIcon, Loader2, RotateCw, Upload, X, ZoomIn, ZoomOut } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Cropper, type CropperRef, getTransformedImageSize, type Transforms } from 'react-advanced-cropper'
import { type Accept, type FileError, type FileRejection, useDropzone } from 'react-dropzone'
import 'react-advanced-cropper/dist/style.css'

import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { Slider } from '@/shared/ui/components/slider'
import { cn } from '@/shared/ui/lib/utils'

interface FileUploadProps {
  /** Accepted file types */
  accept?: Accept
  /** Maximum file size in bytes */
  maxSize?: number
  /** Maximum number of files */
  maxFiles?: number
  /** Whether multiple files can be uploaded */
  multiple?: boolean
  /** Current files */
  value?: File[]
  /** Callback when files change */
  onChange?: (files: File[]) => void
  /** Whether the input is disabled */
  disabled?: boolean
  /** Custom class name */
  className?: string
  /** Placeholder text */
  placeholder?: string
  /** Whether currently uploading */
  isUploading?: boolean
}

export function FileUpload({
  accept = {
    'image/*': ['.png', '.jpg', '.jpeg', '.gif', '.webp'],
    'application/pdf': ['.pdf']
  },
  maxSize = 5 * 1024 * 1024, // 5MB
  maxFiles = 1,
  multiple = false,
  value = [],
  onChange,
  disabled = false,
  className,
  placeholder = 'Drag & drop files here, or click to select',
  isUploading = false
}: FileUploadProps) {
  const onDrop = useCallback(
    (acceptedFiles: File[]) => {
      if (onChange) {
        const newFiles = multiple ? [...value, ...acceptedFiles].slice(0, maxFiles) : acceptedFiles.slice(0, 1)
        onChange(newFiles)
      }
    },
    [onChange, value, multiple, maxFiles]
  )

  const removeFile = useCallback(
    (index: number) => {
      if (onChange) {
        const newFiles = value.filter((_, i) => i !== index)
        onChange(newFiles)
      }
    },
    [onChange, value]
  )

  const { getRootProps, getInputProps, isDragActive, fileRejections } = useDropzone({
    onDrop,
    accept,
    maxSize,
    maxFiles: multiple ? maxFiles : 1,
    multiple,
    disabled: disabled || isUploading
  })

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  const getFileIcon = (file: File) => {
    if (file.type.startsWith('image/')) {
      return <ImageIcon className='size-4' />
    }
    return <FileIcon className='size-4' />
  }

  return (
    <div className={cn('space-y-3', className)}>
      <div
        {...getRootProps()}
        className={cn(
          'relative flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-6 transition-colors',
          isDragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25 hover:border-primary/50',
          disabled && 'cursor-not-allowed opacity-50',
          fileRejections.length > 0 && 'border-destructive'
        )}
      >
        <input {...getInputProps()} />
        {isUploading ? (
          <Loader2 className='size-8 animate-spin text-muted-foreground' />
        ) : (
          <Upload className='size-8 text-muted-foreground' />
        )}
        <p className='mt-2 text-center text-sm text-muted-foreground'>
          {isDragActive ? 'Drop files here...' : placeholder}
        </p>
        <p className='mt-1 text-center text-xs text-muted-foreground'>
          Max {formatFileSize(maxSize)} per file
          {multiple && `, up to ${maxFiles} files`}
        </p>
      </div>

      {/* File rejections */}
      {fileRejections.length > 0 && (
        <div className='text-sm text-destructive'>
          {fileRejections.map((rejection: FileRejection) => (
            <p key={rejection.file.name}>
              {rejection.file.name}: {rejection.errors.map((e: FileError) => e.message).join(', ')}
            </p>
          ))}
        </div>
      )}

      {/* File previews */}
      {value.length > 0 && (
        <div className='space-y-2'>
          {value.map((file, index) => {
            const isImage = file.type.startsWith('image/')
            const previewUrl = isImage && file instanceof Blob ? URL.createObjectURL(file) : null

            if (isImage && !previewUrl) {
              return (
                <div
                  key={`${file.name}-${index}`}
                  className='flex items-center gap-3 rounded-lg border bg-muted/50 p-3'
                >
                  <div className='flex size-10 items-center justify-center rounded bg-muted'>{getFileIcon(file)}</div>
                  <div className='flex-1 min-w-0'>
                    <p className='truncate text-sm font-medium'>{file.name}</p>
                    <p className='text-xs text-muted-foreground'>{formatFileSize(file.size)}</p>
                  </div>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='size-8 shrink-0'
                    onClick={() => removeFile(index)}
                    disabled={disabled}
                  >
                    <X className='size-4' />
                    <span className='sr-only'>Remove file</span>
                  </Button>
                </div>
              )
            }

            return (
              <div key={`${file.name}-${index}`} className='flex items-center gap-3 rounded-lg border bg-muted/50 p-3'>
                {/* Preview thumbnail for images */}
                {previewUrl ? (
                  <img
                    src={previewUrl}
                    alt={file.name}
                    className='size-10 rounded object-cover'
                    onLoad={() => URL.revokeObjectURL(previewUrl)}
                  />
                ) : (
                  <div className='flex size-10 items-center justify-center rounded bg-muted'>{getFileIcon(file)}</div>
                )}

                <div className='flex-1 min-w-0'>
                  <p className='truncate text-sm font-medium'>{file.name}</p>
                  <p className='text-xs text-muted-foreground'>{formatFileSize(file.size)}</p>
                </div>

                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  className='size-8 shrink-0'
                  onClick={() => removeFile(index)}
                  disabled={disabled}
                >
                  <X className='size-4' />
                  <span className='sr-only'>Remove file</span>
                </Button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ============================================================================
// IMAGE UPLOAD WITH CROP EDITOR
// ============================================================================

type ImageAspectRatio = 'square' | 'portrait' | 'video' | 'wide' | 'free' | 'original'

const aspectClasses: Record<ImageAspectRatio, string> = {
  square: 'aspect-square',
  portrait: 'aspect-[4/5]',
  video: 'aspect-video',
  wide: 'aspect-[3/1]',
  free: 'aspect-video',
  original: 'aspect-video'
}

const aspectRatios = { square: 1, portrait: 4 / 5, video: 16 / 9, wide: 3, free: undefined }
const aspectLabels = { square: '1:1', portrait: '4:5', video: '16:9', wide: '3:1', free: 'Free', original: 'Original' }

interface ImageUploadProps {
  value?: File | string | null
  onChange?: (file: File | null) => void
  disabled?: boolean
  className?: string
  aspectRatio?: ImageAspectRatio
  aspectRatioOptions?: readonly ImageAspectRatio[]
  showRemoveButton?: boolean
}

/**
 * Image upload component with built-in cropping, rotation, and zoom controls.
 * Opens a modal editor when an image is selected.
 */
export function ImageUpload({
  value,
  onChange,
  disabled,
  className,
  aspectRatio = 'square',
  aspectRatioOptions,
  showRemoveButton = true
}: ImageUploadProps) {
  const [editorOpen, setEditorOpen] = useState(false)
  const [rawImage, setRawImage] = useState<string | null>(null)
  const [zoom, setZoom] = useState(1)
  const [selectedAspectRatio, setSelectedAspectRatio] = useState(aspectRatio)
  const [imageTransforms, setImageTransforms] = useState<Transforms>()
  const [cropperVersion, setCropperVersion] = useState(0)
  const cropperRef = useRef<CropperRef>(null)

  useEffect(() => {
    return () => {
      if (rawImage) URL.revokeObjectURL(rawImage)
    }
  }, [rawImage])

  const onDrop = useCallback(
    (acceptedFiles: File[]) => {
      const file = acceptedFiles[0]
      if (file) {
        setRawImage(URL.createObjectURL(file))
        setSelectedAspectRatio(aspectRatio)
        setImageTransforms(undefined)
        setZoom(1)
        setEditorOpen(true)
      }
    },
    [aspectRatio]
  )

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: { 'image/*': ['.png', '.jpg', '.jpeg', '.gif', '.webp'] },
    maxSize: 5 * 1024 * 1024,
    maxFiles: 1,
    multiple: false,
    disabled
  })

  const handleRotate = () => {
    cropperRef.current?.rotateImage(90)
  }

  const handleSave = async () => {
    const cropper = cropperRef.current
    if (!cropper) return

    const canvas = cropper.getCanvas()
    if (!canvas) return

    // Convert canvas to blob
    canvas.toBlob(
      (blob) => {
        if (blob) {
          const file = new File([blob], 'cropped-image.jpg', { type: 'image/jpeg' })
          onChange?.(file)
          setEditorOpen(false)
          setRawImage(null)
        }
      },
      'image/jpeg',
      0.9
    )
  }

  const handleCancel = () => {
    setEditorOpen(false)
    setRawImage(null)
    setZoom(1)
  }

  const handleRemove = (e: React.MouseEvent) => {
    e.stopPropagation()
    onChange?.(null)
  }

  const previewUrl = useMemo(
    () => (typeof value === 'string' ? value : value instanceof Blob ? URL.createObjectURL(value) : null),
    [value]
  )
  useEffect(() => {
    return () => {
      if (previewUrl && typeof value !== 'string') URL.revokeObjectURL(previewUrl)
    }
  }, [previewUrl, value])

  return (
    <>
      <div className={cn('relative', className)}>
        <div
          {...getRootProps()}
          className={cn(
            'relative flex cursor-pointer items-center justify-center overflow-hidden rounded-lg border-2 border-dashed transition-colors',
            aspectClasses[aspectRatio],
            isDragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25 hover:border-primary/50',
            disabled && 'cursor-not-allowed opacity-50'
          )}
        >
          <input {...getInputProps({ 'aria-label': 'Upload image' })} />
          {previewUrl ? (
            <img src={previewUrl} alt='Upload preview' className='size-full object-contain' />
          ) : (
            <div className='flex flex-col items-center gap-2 text-muted-foreground'>
              <ImageIcon className='size-8' />
              <span className='text-sm'>Click or drag to upload</span>
              <span className='text-xs opacity-70'>Supports crop, rotate & zoom</span>
            </div>
          )}
        </div>
        {previewUrl && !disabled && showRemoveButton && (
          <Button
            type='button'
            variant='destructive'
            size='icon'
            className='absolute -top-2 -right-2 size-6'
            onClick={handleRemove}
            aria-label='Remove image'
          >
            <X className='size-3' />
          </Button>
        )}
      </div>

      {/* Crop Editor Dialog */}
      <Dialog open={editorOpen} onOpenChange={(open) => !open && handleCancel()}>
        <DialogContent className='flex max-h-[calc(100dvh-2rem)] w-[min(600px,calc(100vw-2rem))] flex-col gap-0 overflow-y-auto p-0'>
          <DialogHeader className='shrink-0 border-b px-6 py-4'>
            <DialogTitle>Edit Image</DialogTitle>
          </DialogHeader>

          <div className='min-h-0 flex-1 overflow-hidden bg-black/95 p-4'>
            {rawImage && (
              <div className='relative h-[min(40dvh,320px)] min-h-32 w-full'>
                <Cropper
                  key={`${selectedAspectRatio}-${cropperVersion}`}
                  ref={cropperRef}
                  src={rawImage}
                  className='h-full w-full'
                  aspectRatio={(state) => {
                    if (selectedAspectRatio === 'original') {
                      const size = getTransformedImageSize(state)
                      return size.width / size.height
                    }
                    return aspectRatios[selectedAspectRatio] ?? {}
                  }}
                  defaultSize={getTransformedImageSize}
                  defaultTransforms={imageTransforms}
                  stencilProps={{ grid: true }}
                  transformImage={{
                    adjustStencil: false
                  }}
                />
              </div>
            )}
          </div>

          {/* Controls */}
          <div className='shrink-0 space-y-4 border-t bg-muted/30 px-6 py-4'>
            {aspectRatioOptions && (
              <fieldset aria-label='Image aspect ratio' className='flex flex-wrap justify-center gap-2'>
                {aspectRatioOptions.map((option) => (
                  <Button
                    key={option}
                    type='button'
                    size='sm'
                    variant={selectedAspectRatio === option ? 'secondary' : 'outline'}
                    aria-pressed={selectedAspectRatio === option}
                    onClick={() => {
                      if (option === selectedAspectRatio) return
                      setImageTransforms(cropperRef.current?.getTransforms() ?? undefined)
                      setSelectedAspectRatio(option)
                      setZoom(1)
                    }}
                  >
                    {aspectLabels[option]}
                  </Button>
                ))}
              </fieldset>
            )}
            {/* Zoom Control */}
            <div className='flex items-center gap-4'>
              <ZoomOut className='size-4 text-muted-foreground' />
              <Slider
                aria-label='Image zoom'
                value={[zoom]}
                onValueChange={(values) => {
                  const val = typeof values === 'number' ? values : (values[0] ?? 1)
                  setZoom(val)
                  cropperRef.current?.zoomImage(val / zoom)
                }}
                min={0.5}
                max={3}
                step={0.1}
                className='flex-1'
              />
              <ZoomIn className='size-4 text-muted-foreground' />
            </div>

            {/* Rotate Button */}
            <div className='flex flex-wrap justify-center gap-2'>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={() => {
                  setImageTransforms(cropperRef.current?.getTransforms() ?? undefined)
                  setCropperVersion((version) => version + 1)
                  setZoom(1)
                }}
              >
                Fit image
              </Button>
              <Button type='button' variant='outline' size='sm' onClick={handleRotate} className='gap-2'>
                <RotateCw className='size-4' />
                Rotate 90°
              </Button>
            </div>
          </div>

          <DialogFooter className='shrink-0 border-t px-6 py-4'>
            <Button type='button' variant='outline' onClick={handleCancel}>
              Cancel
            </Button>
            <Button type='button' onClick={handleSave} className='gap-2'>
              <Check className='size-4' />
              Apply
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
