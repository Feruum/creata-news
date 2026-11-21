import { useEffect, useRef, useState } from 'react'

export function CheckInQrCode({ value }: { value: string }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    const container = containerRef.current
    if (!container || !value) return
    let cancelled = false

    const render = async () => {
      const { default: QRCodeStyling } = await import('qr-code-styling')
      if (cancelled) return

      const qr = new QRCodeStyling({
        width: 220,
        height: 220,
        type: 'svg',
        data: value,
        margin: 8,
        dotsOptions: { type: 'rounded', color: '#16181d' },
        cornersSquareOptions: { type: 'extra-rounded', color: '#16181d' },
        cornersDotOptions: { type: 'dot', color: '#16181d' },
        backgroundOptions: { color: '#ffffff' },
        qrOptions: { errorCorrectionLevel: 'M' }
      })

      container.replaceChildren()
      qr.append(container)
      setError(false)
    }

    void render().catch(() => {
      if (!cancelled) setError(true)
    })

    return () => {
      cancelled = true
      container.replaceChildren()
    }
  }, [value])

  return (
    <div className='flex flex-col items-center gap-2 rounded-lg border bg-background p-4'>
      <p className='text-sm font-medium'>Scan to check in</p>
      {error ? (
        <p role='status' className='py-8 text-xs text-muted-foreground'>
          QR preview unavailable. Use the check-in link below.
        </p>
      ) : (
        <div
          ref={containerRef}
          role='img'
          aria-label='Live event check-in QR code'
          className='min-h-[220px] min-w-[220px]'
        />
      )}
      <p className='text-center text-xs text-muted-foreground'>The code refreshes with the check-in token.</p>
    </div>
  )
}
