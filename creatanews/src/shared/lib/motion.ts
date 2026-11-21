export function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function scrollBehavior(): ScrollBehavior {
  return prefersReducedMotion() ? 'auto' : 'smooth'
}

export function reveal(element: HTMLElement, delay = 0) {
  if (prefersReducedMotion()) return
  return element.animate(
    [
      { opacity: 0, transform: 'translateY(18px)' },
      { opacity: 1, transform: 'translateY(0)' }
    ],
    { duration: 500, delay, easing: 'cubic-bezier(.22, 1, .36, 1)', fill: 'backwards' }
  )
}
