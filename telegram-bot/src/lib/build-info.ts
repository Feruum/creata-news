const releaseVersion = '1.0.0' // x-release-please-version

export const buildInfo = Object.freeze({
  service: 'telegram-bot',
  version: process.env.SERVICE_VERSION ?? releaseVersion,
  commit: process.env.GIT_COMMIT ?? 'unknown',
  built_at: process.env.BUILD_TIME ?? 'unknown',
  environment: process.env.ENVIRONMENT ?? 'development'
})
