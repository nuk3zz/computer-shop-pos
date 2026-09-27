import { createFileRoute } from '@tanstack/react-router'
import { RepairQueue } from '@/components/repairs/RepairQueue'

export const Route = createFileRoute('/admin/repairs')({
  component: RepairQueue,
})
