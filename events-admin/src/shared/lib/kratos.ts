import { Configuration, FrontendApi } from '@ory/client'

const KRATOS_PUBLIC_URL = import.meta.env.VITE_KRATOS_PUBLIC_URL || 'http://localhost:4433'

const verificationEndpoint = `${KRATOS_PUBLIC_URL.replace(/\/+$/, '')}/self-service/verification`

export function verificationSubmitUrl(flow: string) {
  const url = new URL(verificationEndpoint)
  url.searchParams.set('flow', flow)
  return url.href
}

export function startEmailVerification() {
  return `${verificationEndpoint}/browser`
}

// Initialize Ory Kratos SDK
export const kratos = new FrontendApi(
  new Configuration({
    basePath: KRATOS_PUBLIC_URL,
    baseOptions: {
      withCredentials: true // Essential for cookie-based sessions
    }
  })
)
