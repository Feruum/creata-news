/**
 * Animation utilities for neoflat UI design system.
 * Industry-standard SaaS dashboard animations following 2025 UX patterns.
 *
 * Motion principles:
 * - Subtle, professional animations (no flashy effects)
 * - Quick transitions (150-300ms for UI feedback)
 * - Spring physics for natural feel
 * - Stagger for lists/grids
 */

// ============================================================================
// Tailwind CSS Animation Classes
// ============================================================================

/**
 * Standard transition durations (use with Tailwind classes)
 */
export const durations = {
  /** Instant feedback (75ms) - button states, checkboxes */
  instant: 'duration-75',
  /** Fast feedback (150ms) - hover states, toggles */
  fast: 'duration-150',
  /** Normal (200ms) - most UI transitions */
  normal: 'duration-200',
  /** Moderate (300ms) - modals, panels appearing */
  moderate: 'duration-300',
  /** Slow (500ms) - page transitions, complex animations */
  slow: 'duration-500'
} as const

/**
 * Standard easing functions
 */
export const easings = {
  /** Default ease-out for UI (deceleration) */
  out: 'ease-out',
  /** Ease-in-out for symmetrical animations */
  inOut: 'ease-in-out',
  /** Spring-like overshoot for playful elements */
  spring: 'ease-[cubic-bezier(0.34,1.56,0.64,1)]',
  /** Smooth deceleration */
  smooth: 'ease-[cubic-bezier(0.4,0,0.2,1)]'
} as const

/**
 * Pre-composed animation class combinations
 */
export const transitions = {
  /** Default for interactive elements */
  default: 'transition-colors duration-150 ease-out',
  /** For transforms (scale, translate) */
  transform: 'transition-transform duration-200 ease-out',
  /** For opacity changes */
  opacity: 'transition-opacity duration-150 ease-out',
  /** For all properties (use sparingly) */
  all: 'transition-all duration-200 ease-out',
  /** For spring-like bounce effects */
  spring: 'transition-transform duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)]'
} as const

// ============================================================================
// Framer Motion / Motion Variants (for use with motion/react)
// ============================================================================

/**
 * Fade in/out animation variants
 */
export const fadeVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
  exit: { opacity: 0 }
} as const

/**
 * Slide up animation (for modals, toasts, popovers)
 */
export const slideUpVariants = {
  hidden: { opacity: 0, y: 10 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { type: 'spring', stiffness: 500, damping: 30 }
  },
  exit: { opacity: 0, y: 10 }
} as const

/**
 * Slide down animation (for dropdowns)
 */
export const slideDownVariants = {
  hidden: { opacity: 0, y: -10 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { type: 'spring', stiffness: 500, damping: 30 }
  },
  exit: { opacity: 0, y: -10 }
} as const

/**
 * Scale animation (for buttons, cards on hover)
 */
export const scaleVariants = {
  hidden: { opacity: 0, scale: 0.95 },
  visible: {
    opacity: 1,
    scale: 1,
    transition: { type: 'spring', stiffness: 500, damping: 30 }
  },
  exit: { opacity: 0, scale: 0.95 }
} as const

/**
 * Stagger children animation for lists/grids
 */
export const staggerContainer = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.05,
      delayChildren: 0.1
    }
  }
} as const

/**
 * Individual item in staggered list
 */
export const staggerItem = {
  hidden: { opacity: 0, y: 10 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { type: 'spring', stiffness: 500, damping: 30 }
  }
} as const

// ============================================================================
// Spring Configurations (for motion/react)
// ============================================================================

/**
 * Pre-defined spring configurations
 */
export const springs = {
  /** Snappy, responsive spring (buttons, toggles) */
  snappy: { type: 'spring', stiffness: 500, damping: 30 } as const,
  /** Gentle spring (cards, panels) */
  gentle: { type: 'spring', stiffness: 300, damping: 25 } as const,
  /** Bouncy spring (playful elements, notifications) */
  bouncy: { type: 'spring', stiffness: 400, damping: 15 } as const,
  /** Smooth spring (page transitions) */
  smooth: { type: 'spring', stiffness: 200, damping: 20 } as const
} as const

// ============================================================================
// Animation Presets for Common Components
// ============================================================================

/**
 * Modal/Dialog animations
 */
export const modalAnimation = {
  initial: { opacity: 0, scale: 0.95, y: 10 },
  animate: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: springs.snappy
  },
  exit: {
    opacity: 0,
    scale: 0.95,
    y: 10,
    transition: { duration: 0.15 }
  }
} as const

/**
 * Tooltip/Popover animations
 */
export const tooltipAnimation = {
  initial: { opacity: 0, scale: 0.96 },
  animate: {
    opacity: 1,
    scale: 1,
    transition: { duration: 0.15, ease: 'easeOut' }
  },
  exit: {
    opacity: 0,
    scale: 0.96,
    transition: { duration: 0.1 }
  }
} as const

/**
 * Floating action bar animation (bottom bar)
 */
export const floatingBarAnimation = {
  initial: { opacity: 0, y: 20 },
  animate: {
    opacity: 1,
    y: 0,
    transition: springs.snappy
  },
  exit: {
    opacity: 0,
    y: 20,
    transition: { duration: 0.2 }
  }
} as const

/**
 * Toast/Notification animations
 */
export const toastAnimation = {
  initial: { opacity: 0, x: 50, scale: 0.95 },
  animate: {
    opacity: 1,
    x: 0,
    scale: 1,
    transition: springs.snappy
  },
  exit: {
    opacity: 0,
    x: 50,
    transition: { duration: 0.2 }
  }
} as const

/**
 * Sidebar/Drawer animations
 */
export const sidebarAnimation = {
  initial: { x: '-100%' },
  animate: {
    x: 0,
    transition: springs.gentle
  },
  exit: {
    x: '-100%',
    transition: { duration: 0.2 }
  }
} as const

/**
 * Card hover animation (use with whileHover)
 */
export const cardHoverAnimation = {
  scale: 1.02,
  transition: springs.snappy
} as const

/**
 * Button tap animation (use with whileTap)
 */
export const buttonTapAnimation = {
  scale: 0.98
} as const

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Create stagger delay based on index
 * @param index Item index in list
 * @param baseDelay Base delay in seconds (default: 0.05)
 */
export function getStaggerDelay(index: number, baseDelay = 0.05): number {
  return index * baseDelay
}

/**
 * Create animation variants with custom duration
 * @param variants Base variants
 * @param duration Duration in seconds
 */
export function withDuration<T extends Record<string, unknown>>(variants: T, duration: number): T {
  return {
    ...variants,
    visible: {
      ...(variants.visible as Record<string, unknown>),
      transition: { duration }
    }
  }
}
