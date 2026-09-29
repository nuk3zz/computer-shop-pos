import { createFileRoute } from '@tanstack/react-router'
import { SupplyChain } from '@/components/supply/SupplyChain'

export const Route = createFileRoute('/admin/supply-chain')({ component: SupplyChain })
