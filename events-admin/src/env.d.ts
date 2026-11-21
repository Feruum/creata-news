/// <reference types="vite/client" />

declare global {
  interface ImportMetaEnv {
    readonly VITE_API_URL: string
    readonly VITE_KRATOS_PUBLIC_URL: string
    readonly VITE_OTEL_ENABLED: string
    readonly VITE_OTEL_ENDPOINT: string
    readonly VITE_OTEL_SERVICE_NAME: string
    readonly VITE_OTEL_SERVICE_VERSION: string
    readonly VITE_OTEL_ENVIRONMENT: string
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv
  }
}

export {}
