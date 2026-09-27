import { createFileRoute } from '@tanstack/react-router'
import { ClientManagement } from '@/components/clients/ClientManagement'

export const Route = createFileRoute('/admin/clients')({
  component: ClientManagement,
})
