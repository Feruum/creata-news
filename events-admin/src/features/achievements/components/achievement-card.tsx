import { AchievementEffect } from '@studyverse/proto'
import { motion, useReducedMotion } from 'motion/react'
import { useState } from 'react'
import { useAchievementCardMotion } from '../hooks/use-achievement-card-motion'
import type { AchievementFormValues } from '../validation'
import './achievement-card.css'

type CardAppearance = Pick<
  AchievementFormValues,
  'title' | 'description' | 'imageUrl' | 'backgroundColor' | 'accentColor' | 'effect'
>

export function AchievementCard({ achievement }: { achievement: CardAppearance }) {
  const reducedMotion = useReducedMotion()
  const animated = reducedMotion === false && achievement.effect > AchievementEffect.NONE
  const tilt = useAchievementCardMotion(animated)
  const [failedImage, setFailedImage] = useState<string | null>(null)
  const fallback = `${import.meta.env.BASE_URL}images/studyverse-mascot.png`
  const image = !achievement.imageUrl || failedImage === achievement.imageUrl ? fallback : achievement.imageUrl
  return (
    <div
      className='achievement-stage'
      onPointerEnter={tilt.onPointerEnter}
      onPointerMove={tilt.onPointerMove}
      onPointerLeave={tilt.onPointerLeave}
    >
      <motion.div aria-hidden className='achievement-shadow' style={{ x: animated ? tilt.shadowX : 0 }} />
      <motion.div
        className='achievement-floating-card'
        animate={animated ? { y: [0, -7, 0], rotateZ: [-1, 1, -1] } : { y: 0, rotateZ: 0 }}
        transition={animated ? { duration: 6, repeat: Infinity, ease: 'easeInOut' } : { duration: 0 }}
      >
        <motion.article
          className='achievement-card'
          aria-label={`${achievement.title || 'Achievement'} preview`}
          data-animated={animated}
          style={{
            rotateX: animated ? tilt.rotateX : 0,
            rotateY: animated ? tilt.rotateY : 0,
            background: achievement.backgroundColor || undefined
          }}
        >
          <motion.div
            aria-hidden
            className='achievement-sheen'
            style={{ backgroundImage: animated ? tilt.shine : undefined }}
          />
          <div className='achievement-card-heading'>
            <img src={`${import.meta.env.BASE_URL}images/studyverse-logo.svg`} alt='' width={24} height={24} />
            <span>
              Studyverse
              <br />
              Achievement
            </span>
          </div>
          <div className='achievement-mascot' aria-hidden>
            <img
              src={image}
              alt=''
              width={630}
              height={630}
              className='achievement-card-image'
              draggable={false}
              onError={() => setFailedImage(achievement.imageUrl)}
            />
            {achievement.effect > AchievementEffect.NONE && (
              <div
                className='achievement-holographic'
                data-effect={achievement.effect}
                style={{ maskImage: `url(${JSON.stringify(image)})` }}
              >
                <div className='achievement-spectrum' />
              </div>
            )}
          </div>
          <div className='achievement-card-copy'>
            <h2>{achievement.title || 'Your achievement'}</h2>
            <span
              className='achievement-name-rule'
              aria-hidden
              style={{ backgroundColor: achievement.accentColor || undefined }}
            />
            <p>Earned in Studyverse</p>
          </div>
        </motion.article>
      </motion.div>
    </div>
  )
}
