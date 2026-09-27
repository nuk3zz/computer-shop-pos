import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import apiClient from '@/api/client'
import { toastHelpers } from '@/lib/toast-helpers'
import { buildWhatsAppUrl, formatMoney, loadShopSettings, renderWhatsAppTemplate } from '@/lib/shop-settings'
import type { Order } from '@/types'
import { CheckCircle2, Clock3, MessageCircle, PackageCheck, Search, Wrench } from 'lucide-react'

const statusConfig = {
  pending: { label: 'Waiting', className: 'bg-slate-100 text-slate-700', next: 'confirmed', action: 'Start diagnosis' },
  confirmed: { label: 'Diagnosing', className: 'bg-blue-100 text-blue-700', next: 'preparing', action: 'Start repair' },
  preparing: { label: 'In progress', className: 'bg-amber-100 text-amber-800', next: 'ready', action: 'Waiting for customer' },
  ready: { label: 'Waiting for customer', className: 'bg-emerald-100 text-emerald-700', next: 'completed', action: 'Mark delivered' },
  served: { label: 'Waiting for customer', className: 'bg-emerald-100 text-emerald-700', next: 'completed', action: 'Mark delivered' },
  completed: { label: 'Delivered / completed', className: 'bg-green-100 text-green-800', next: null, action: null },
  cancelled: { label: 'Cancelled', className: 'bg-red-100 text-red-700', next: null, action: null },
} as const

type FilterKey = 'active' | 'all' | Order['status']

export function RepairQueue() {
  const [filter, setFilter] = useState<FilterKey>('active')
  const queryClient = useQueryClient()

  const { data: orders = [], isLoading } = useQuery({
    queryKey: ['repair-orders'],
    queryFn: () => apiClient.getOrders({ order_type: 'service', per_page: 100 }).then((response) => response.data || []),
    refetchInterval: 30_000,
  })

  const updateStatus = useMutation({
    mutationFn: ({ orderId, status }: { orderId: string; status: Order['status'] }) => apiClient.updateOrderStatus(orderId, status),
    onSuccess: () => {
      toastHelpers.apiSuccess('Repair ticket', 'Status updated')
      queryClient.invalidateQueries({ queryKey: ['repair-orders'] })
      queryClient.invalidateQueries({ queryKey: ['dashboardStats'] })
    },
    onError: (error) => toastHelpers.apiError('Update repair ticket', error),
  })

  const visibleOrders = orders.filter((order) => {
    if (filter === 'all') return true
    if (filter === 'active') return !['completed', 'cancelled'].includes(order.status)
    return order.status === filter
  })

  const openWhatsApp = (order: Order) => {
    if (!order.customer_phone) return
    const settings = loadShopSettings()
    const template = order.status === 'confirmed'
      ? settings.whatsapp_diagnosing
      : order.status === 'preparing'
        ? settings.whatsapp_repairing
        : order.status === 'completed'
          ? settings.whatsapp_completed
          : settings.whatsapp_ready
    const message = renderWhatsAppTemplate(template, {
      customer: order.customer_name || 'customer',
      jobNumber: order.order_number,
    })
    window.open(buildWhatsAppUrl(order.customer_phone, message), '_blank', 'noopener,noreferrer')
  }

  return (
    <div className="min-h-screen bg-slate-50 p-6">
      <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 className="flex items-center gap-3 text-3xl font-bold tracking-tight"><Wrench className="h-8 w-8" /> Repair Tickets</h1>
          <p className="mt-1 text-muted-foreground">Track diagnostics, repair work, and devices ready for collection.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {([
            ['active', 'Active'],
            ['pending', 'Waiting'],
            ['confirmed', 'Diagnosing'],
            ['preparing', 'In progress'],
            ['ready', 'Waiting for customer'],
            ['completed', 'Completed'],
            ['all', 'All'],
          ] as Array<[FilterKey, string]>).map(([value, label]) => (
            <Button key={value} variant={filter === value ? 'default' : 'outline'} size="sm" onClick={() => setFilter(value)}>{label}</Button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{Array.from({ length: 6 }).map((_, index) => <div key={index} className="h-64 animate-pulse rounded-xl bg-slate-200" />)}</div>
      ) : visibleOrders.length === 0 ? (
        <div className="flex min-h-96 flex-col items-center justify-center rounded-xl border border-dashed bg-white text-center">
          <Search className="mb-3 h-12 w-12 text-slate-300" />
          <h2 className="font-semibold">No repair tickets in this view</h2>
          <p className="mt-1 text-sm text-muted-foreground">Create one from Sales & Services.</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {visibleOrders.map((order) => {
            const config = statusConfig[order.status]
            return (
              <Card key={order.id} className="overflow-hidden">
                <CardHeader className="border-b bg-white pb-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <CardTitle className="text-lg">{order.customer_name || 'Unnamed customer'}</CardTitle>
                      <p className="mt-1 font-mono text-xs text-muted-foreground">{order.order_number}</p>
                    </div>
                    <Badge className={config.className}>{config.label}</Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4 p-5">
                  <div className="space-y-2">
                    {(order.items || []).map((item) => (
                      <div key={item.id} className="flex justify-between gap-3 text-sm">
                        <span>{item.quantity} × {item.product?.name || 'Service item'}</span>
                        <span className="font-medium">{formatMoney(item.total_price)}</span>
                      </div>
                    ))}
                  </div>
                  {order.notes && <div className="rounded-lg bg-slate-100 p-3 text-sm text-slate-700"><span className="font-medium">Device / fault:</span> {order.notes}</div>}
                  {order.customer_phone && <div className="text-sm text-muted-foreground">WhatsApp: {order.customer_phone}</div>}
                  <div className="flex items-center justify-between border-t pt-3 text-sm">
                    <span className="flex items-center gap-1 text-muted-foreground"><Clock3 className="h-4 w-4" /> {new Date(order.created_at).toLocaleString()}</span>
                    <span className="font-bold">{formatMoney(order.total_amount)}</span>
                  </div>
                  <div className="flex gap-2">
                    {order.customer_phone && ['confirmed', 'preparing', 'ready', 'served', 'completed'].includes(order.status) && (
                      <Button variant="outline" onClick={() => openWhatsApp(order)} title="Open WhatsApp with a pre-filled status message">
                        <MessageCircle className="mr-2 h-4 w-4" /> WhatsApp
                      </Button>
                    )}
                    {config.next && config.action && (
                      <Button
                        className="flex-1"
                        onClick={() => updateStatus.mutate({ orderId: order.id, status: config.next })}
                        disabled={updateStatus.isPending}
                      >
                        {config.next === 'completed' ? <CheckCircle2 className="mr-2 h-4 w-4" /> : <PackageCheck className="mr-2 h-4 w-4" />}
                        {config.action}
                      </Button>
                    )}
                    {!['completed', 'cancelled'].includes(order.status) && (
                      <Button variant="outline" onClick={() => updateStatus.mutate({ orderId: order.id, status: 'cancelled' })} disabled={updateStatus.isPending}>Cancel</Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
