import { Slider as SliderPrimitive } from '@base-ui/react/slider'
import { cn } from '@/shared/ui/lib/utils'

function Slider({ className, defaultValue, value, min = 0, max = 100, ...props }: SliderPrimitive.Root.Props) {
  const _values = Array.isArray(value) ? value : Array.isArray(defaultValue) ? defaultValue : [min]

  return (
    <SliderPrimitive.Root
      className={cn('relative flex w-full touch-none select-none items-center', className)}
      data-slot='slider'
      defaultValue={defaultValue}
      value={value}
      min={min}
      max={max}
      thumbAlignment='edge'
      {...props}
    >
      <SliderPrimitive.Control className='relative flex w-full items-center py-2'>
        <SliderPrimitive.Track
          data-slot='slider-track'
          className='relative h-1.5 w-full grow overflow-hidden rounded-full bg-primary/20'
        >
          <SliderPrimitive.Indicator data-slot='slider-range' className='absolute h-full bg-primary' />
        </SliderPrimitive.Track>
        {Array.from({ length: _values.length }, (_, index) => (
          <SliderPrimitive.Thumb
            data-slot='slider-thumb'
            aria-label={props['aria-label']}
            aria-labelledby={props['aria-labelledby']}
            /* biome-ignore lint/suspicious/noArrayIndexKey: Thumb position is a stable identity; value keys would remount during dragging. */
            key={index}
            className='block h-4 w-4 rounded-full border border-primary/50 bg-background transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50'
          />
        ))}
      </SliderPrimitive.Control>
    </SliderPrimitive.Root>
  )
}

export { Slider }
