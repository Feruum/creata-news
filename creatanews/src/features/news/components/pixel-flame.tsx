import styles from './pixel-flame.module.css'

// Original four-frame pixel artwork. Integer coordinates keep the edges crisp at small sizes.
const frames = [
  {
    outer: 'M6 1h2v3h2v3h2v5h-1v2H9v1H4v-1H2v-2H1V8h2V5h2v3h1Z',
    middle: 'M6 5h2v3h2v4H9v2H4v-1H3V9h2v1h1Z',
    core: 'M6 9h2v2h1v2H8v1H5v-1H4v-2h2Z'
  },
  {
    outer: 'M5 0h2v3h2v3h2v2h1v4h-1v2H9v1H4v-1H2v-2H1V7h2V4h1v5h1Z',
    middle: 'M5 5h2v2h2v2h1v3H9v2H4v-1H3V9h2Z',
    core: 'M6 8h1v3h2v2H8v1H5v-1H4v-2h2Z'
  },
  {
    outer: 'M7 1h2v4h1v3h2v4h-1v2H9v1H4v-1H2v-2H1V9h1V6h2v2h2V4h1Z',
    middle: 'M7 6h1v2h2v4H9v2H4v-1H3v-3h2V8h2Z',
    core: 'M7 9h1v2h1v2H8v1H5v-1H4v-2h3Z'
  },
  {
    outer: 'M6 2h2v3h2v2h1v3h1v2h-1v2H9v1H4v-1H2v-2H1V8h2V5h2v4h1Z',
    middle: 'M6 6h2v3h2v3H9v2H4v-1H3v-3h3Z',
    core: 'M6 10h2v1h1v2H8v1H5v-1H4v-1h2Z'
  }
]

export function PixelFlame() {
  return (
    <svg className={styles.flame} viewBox='0 0 13 16' fill='none' aria-hidden='true' focusable='false'>
      {frames.map((frame) => (
        <g key={frame.outer} className={styles.frame}>
          <path d={frame.outer} fill='var(--logo-blue)' />
          <path d={frame.middle} fill='var(--brand-cyan)' />
          <path d={frame.core} fill='white' />
        </g>
      ))}
    </svg>
  )
}
