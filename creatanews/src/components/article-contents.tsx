'use client'

import { useEffect, useState } from 'react'
import styles from './article-view.module.css'

type SectionLink = { id: string; label: string }

export function ArticleContents({ label, sections }: { label: string; sections: SectionLink[] }) {
  const [current, setCurrent] = useState<string>()

  useEffect(() => {
    const targets = sections.flatMap(({ id }) => {
      const element = document.getElementById(id)
      return element ? [element] : []
    })
    const anchorInset = parseFloat(getComputedStyle(document.documentElement).scrollPaddingTop) || 0
    let frame: number | undefined

    function update() {
      frame = undefined
      const maxScroll = document.documentElement.scrollHeight - window.innerHeight
      const atPageEnd = maxScroll > 0 && maxScroll - window.scrollY < 2
      const readingLine = atPageEnd ? window.innerHeight : Math.min(96, window.innerHeight / 5)
      const fragment = window.location.hash.slice(1)
      let active: string | undefined
      for (const target of targets) {
        const top = target.getBoundingClientRect().top
        if (top > readingLine) break
        active = target.id
        // A link near the end of the page can stop short of the anchor inset.
        if (target.id === fragment && (Math.abs(top - anchorInset) < 2 || (atPageEnd && top > anchorInset))) break
      }
      setCurrent(active)
    }

    function schedule() {
      frame ??= requestAnimationFrame(update)
    }

    const resize = new ResizeObserver(schedule)
    resize.observe(document.body)
    window.addEventListener('scroll', schedule, { passive: true })
    window.addEventListener('resize', schedule)
    window.addEventListener('hashchange', schedule)
    schedule()
    return () => {
      if (frame !== undefined) cancelAnimationFrame(frame)
      resize.disconnect()
      window.removeEventListener('scroll', schedule)
      window.removeEventListener('resize', schedule)
      window.removeEventListener('hashchange', schedule)
    }
  }, [sections])

  return <nav className={styles.contents} aria-label={label}>
    <p>{label}</p>
    <ul>{sections.map(({ id, label }) => (
      <li key={id}><a href={`#${id}`} aria-current={id === current ? 'location' : undefined}>{label}</a></li>
    ))}</ul>
  </nav>
}
