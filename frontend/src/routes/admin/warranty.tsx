import { createFileRoute } from '@tanstack/react-router'
import { WarrantyReturns } from '@/components/warranty/WarrantyReturns'

export const Route = createFileRoute('/admin/warranty')({ component: WarrantyReturns })
