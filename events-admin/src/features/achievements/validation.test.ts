import assert from 'node:assert/strict'
import { test } from 'node:test'
import { AchievementEffect, AchievementRuleKind, AchievementStatus } from '@studyverse/proto'
import {
  achievementFormSchema,
  selectedRecipientIds,
  validateAchievementImage,
  validateAwardAction
} from './validation'

const valid = {
  title: 'Campus explorer',
  description: '',
  imageUrl: '',
  badgeIconUrl: '',
  backgroundColor: '#152034',
  accentColor: '#8bd7ff',
  effect: AchievementEffect.NONE,
  status: AchievementStatus.DRAFT,
  ruleKind: AchievementRuleKind.UNSPECIFIED,
  threshold: 1,
  eventId: 0,
  organizationId: 0,
  categoryId: 0
}

test('automatic criteria require a positive integer and retain combined filters', () => {
  for (const threshold of [0, -1, 1.5, Number.NaN]) {
    assert.equal(
      achievementFormSchema.safeParse({ ...valid, ruleKind: AchievementRuleKind.REGISTRATIONS, threshold }).success,
      false
    )
  }
  const result = achievementFormSchema.parse({
    ...valid,
    ruleKind: AchievementRuleKind.REGISTRATIONS,
    threshold: 3,
    eventId: 8,
    organizationId: 9,
    categoryId: 4
  })
  assert.equal(result.eventId, 8)
  assert.equal(result.organizationId, 9)
  assert.equal(result.categoryId, 4)
  assert.equal(achievementFormSchema.safeParse({ ...valid, title: ' ' }).success, false)
})

test('award recipients use only checked existing users, remove duplicates, and ignore invalid IDs', () => {
  assert.deepEqual(
    selectedRecipientIds({ '12': true, '012': true, '15': false, '-1': true, invalid: true, '0': true }),
    [12]
  )
})

test('grant and revoke require recipients, an audit reason, and a bounded batch', () => {
  assert.ok(validateAwardAction([], 'Review correction'))
  assert.ok(validateAwardAction([12], '   '))
  assert.ok(
    validateAwardAction(
      Array.from({ length: 501 }, (_, index) => index + 1),
      'Review correction'
    )
  )
  assert.ok(validateAwardAction([12], 'я'.repeat(1001)))
  assert.equal(
    validateAwardAction(
      Array.from({ length: 500 }, (_, index) => index + 1),
      'Review correction'
    ),
    undefined
  )
})

test('enum values and UTF-8 text limits match the server contract', () => {
  assert.equal(
    achievementFormSchema.safeParse({
      ...valid,
      effect: AchievementEffect.HOLOGRAPHIC,
      status: AchievementStatus.ARCHIVED
    }).success,
    true
  )
  assert.equal(achievementFormSchema.safeParse({ ...valid, effect: AchievementEffect.UNSPECIFIED }).success, false)
  for (const effect of [AchievementEffect.AURORA, AchievementEffect.GOLD_FOIL, AchievementEffect.SPARKLE]) {
    assert.equal(achievementFormSchema.safeParse({ ...valid, effect }).success, true)
  }
  assert.equal(achievementFormSchema.safeParse({ ...valid, effect: 6 }).success, false)
  assert.equal(achievementFormSchema.safeParse({ ...valid, status: AchievementStatus.UNSPECIFIED }).success, false)
  assert.equal(achievementFormSchema.safeParse({ ...valid, title: 'я'.repeat(101) }).success, false)
})

test('uploads accept PNG JPEG WebP up to 5 MB and reject empty or unsupported files', () => {
  assert.equal(validateAchievementImage({ type: 'image/webp', size: 5 * 1024 * 1024 }), undefined)
  assert.ok(validateAchievementImage({ type: 'image/svg+xml', size: 123 }))
  assert.ok(validateAchievementImage({ type: 'image/png', size: 0 }))
  assert.ok(validateAchievementImage({ type: 'image/jpeg', size: 5 * 1024 * 1024 + 1 }))
})
