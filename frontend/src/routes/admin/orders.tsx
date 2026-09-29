import { createFileRoute } from '@tanstack/react-router'
import { ProductOrderQueue } from '@/components/orders/ProductOrderQueue'

export const Route = createFileRoute('/admin/orders')({ component: ProductOrderQueue })
