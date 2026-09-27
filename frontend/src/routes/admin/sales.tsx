import { createFileRoute } from '@tanstack/react-router'
import { SalesWorkspace } from '@/components/sales/SalesWorkspace'

export const Route = createFileRoute('/admin/sales')({
  component: SalesWorkspace,
})
