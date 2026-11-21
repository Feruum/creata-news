import { useMotionTemplate, useMotionValue, useSpring, useTransform } from 'motion/react'
import { type PointerEvent, useEffect, useRef } from 'react'

const spring = { stiffness: 150, damping: 22, mass: 0.6 }

export function useAchievementCardMotion(enabled: boolean) {
  const bounds = useRef<DOMRect | null>(null)
  const pointerX = useMotionValue(0)
  const pointerY = useMotionValue(0)
  const smoothX = useSpring(pointerX, spring)
  const smoothY = useSpring(pointerY, spring)
  const rotateX = useTransform(smoothY, [-1, 1], [11, -11])
  const rotateY = useTransform(smoothX, [-1, 1], [-14, 14])
  const shineX = useTransform(smoothX, [-1, 1], [12, 88])
  const shineY = useTransform(smoothY, [-1, 1], [15, 85])
  const shadowX = useTransform(smoothX, [-1, 1], [-12, 12])
  const shine = useMotionTemplate`radial-gradient(ellipse at ${shineX}% ${shineY}%, #ffffff25, transparent 65%)`

  useEffect(() => {
    if (!enabled) {
      pointerX.set(0)
      pointerY.set(0)
      smoothX.jump(0)
      smoothY.jump(0)
      bounds.current = null
    }
  }, [enabled, pointerX, pointerY, smoothX, smoothY])

  function onPointerEnter(event: PointerEvent<HTMLDivElement>) {
    if (enabled && event.pointerType === 'mouse') bounds.current = event.currentTarget.getBoundingClientRect()
  }
  function onPointerMove(event: PointerEvent<HTMLDivElement>) {
    if (!enabled || event.pointerType !== 'mouse') return
    if (!bounds.current) bounds.current = event.currentTarget.getBoundingClientRect()
    const rect = bounds.current
    pointerX.set(rect.width > 0 ? Math.max(-1, Math.min(1, ((event.clientX - rect.left) / rect.width - 0.5) * 2)) : 0)
    pointerY.set(rect.height > 0 ? Math.max(-1, Math.min(1, ((event.clientY - rect.top) / rect.height - 0.5) * 2)) : 0)
  }
  function onPointerLeave() {
    pointerX.set(0)
    pointerY.set(0)
    bounds.current = null
  }
  return { rotateX, rotateY, shadowX, shine, onPointerEnter, onPointerMove, onPointerLeave }
}
