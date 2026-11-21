import { Configuration, FrontendApi } from '@ory/client'

export const kratos = new FrontendApi(
  new Configuration({
    basePath: import.meta.env.VITE_KRATOS_PUBLIC_URL || 'http://localhost:4433',
    baseOptions: { withCredentials: true },
  })
)
