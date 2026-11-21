import { createFileRoute } from '@tanstack/react-router'
import { HermesLauncher } from '@/features/hermes'

export const Route = createFileRoute('/_authenticated/hermes')({
  component: HermesLauncher
})
