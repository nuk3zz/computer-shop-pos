import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Banknote, Box, CheckCircle2, PackageCheck, Phone, Truck } from 'lucide-react'
import apiClient from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatMoney } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'
import type { Order } from '@/types'

const fulfillmentLabels: Record<string, string> = {
  pending_packing: 'Pending packing',
  packed: 'Packed',
  with_courier: 'Handed to courier',
  delivered: 'Delivered',
  completed: 'Completed',
}

const methodLabels: Record<string, string> = {
  in_store: 'Instant sale',
  pickup: 'Customer pickup',
  delivery: 'Prepaid delivery',
  cash_on_delivery: 'Cash on delivery',
}

export function ProductOrderQueue() {
  const queryClient = useQueryClient()
  const { data: orders = [], isLoading } = useQuery({
    queryKey: ['product-orders'],
    queryFn: () => apiClient.getOrders({ order_type: 'sale', per_page: 100 }).then((response) => response.data || []),
  })

  const refreshBusinessData = () => {
    queryClient.invalidateQueries({ queryKey: ['product-orders'] })
    queryClient.invalidateQueries({ queryKey: ['orders'] })
    queryClient.invalidateQueries({ queryKey: ['dashboardStats'] })
    queryClient.invalidateQueries({ queryKey: ['salesReport'] })
    queryClient.invalidateQueries({ queryKey: ['incomeReport'] })
    queryClient.invalidateQueries({ queryKey: ['customers'] })
  }

  const updateFulfillment = useMutation({
    mutationFn: ({ orderId, status }: { orderId: string; status: Order['fulfillment_status'] }) => apiClient.updateFulfillmentStatus(orderId, status),
    onSuccess: () => { refreshBusinessData(); toastHelpers.apiSuccess('Product order', 'Delivery status updated') },
    onError: (error) => toastHelpers.apiError('Update product order', error),
  })

  const recordPayment = useMutation({
    mutationFn: (order: Order) => apiClient.processPayment(order.id, { payment_method: 'cash', amount: order.total_amount - paidAmount(order) }),
    onSuccess: () => { refreshBusinessData(); toastHelpers.apiSuccess('Payment', 'Cash-on-delivery payment recorded') },
    onError: (error) => toastHelpers.apiError('Record payment', error),
  })

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="flex items-center gap-3 text-3xl font-bold tracking-tight"><Truck className="h-8 w-8" /> Product Orders</h1>
        <p className="mt-1 text-muted-foreground">Track packing, courier delivery, and cash-on-delivery payments.</p>
      </div>

      {isLoading ? <div className="text-muted-foreground">Loading product orders…</div> : orders.length === 0 ? (
        <div className="rounded-xl border border-dashed bg-white p-12 text-center text-muted-foreground">No product orders yet. Create one from Sales & Services.</div>
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          {orders.map((order) => {
            const paid = paidAmount(order)
            const isPaid = paid >= order.total_amount
            return (
              <Card key={order.id}>
                <CardHeader className="space-y-2 pb-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <CardTitle className="text-lg">{order.order_number}</CardTitle>
                    <div className="flex gap-2"><Badge variant="outline">{methodLabels[order.fulfillment_type] || order.fulfillment_type}</Badge><Badge className={isPaid ? 'bg-emerald-600' : 'bg-amber-500'}>{isPaid ? 'Paid' : `${formatMoney(order.total_amount - paid)} due`}</Badge></div>
                  </div>
                  <div className="text-sm text-muted-foreground">{new Date(order.created_at).toLocaleString()}</div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-start justify-between gap-4">
                    <div><div className="font-semibold">{order.customer_name || 'Counter customer'}</div>{order.customer_phone && <div className="mt-1 flex items-center gap-1.5 text-sm text-muted-foreground"><Phone className="h-3.5 w-3.5" />{order.customer_phone}</div>}</div>
                    <div className="text-right"><div className="font-bold">{formatMoney(order.total_amount)}</div><div className="mt-1 text-xs text-muted-foreground">{fulfillmentLabels[order.fulfillment_status] || order.fulfillment_status}</div></div>
                  </div>
                  <div className="space-y-1 rounded-lg bg-slate-50 p-3 text-sm">
                    {order.items?.map((item) => <div key={item.id} className="flex justify-between gap-3"><span>{item.quantity} × {item.product?.name || 'Product'}</span><span>{formatMoney(item.total_price)}</span></div>)}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {order.fulfillment_status === 'pending_packing' && <Button size="sm" onClick={() => updateFulfillment.mutate({ orderId: order.id, status: 'packed' })}><Box className="mr-2 h-4 w-4" />Mark packed</Button>}
                    {order.fulfillment_status === 'packed' && <Button size="sm" onClick={() => updateFulfillment.mutate({ orderId: order.id, status: 'with_courier' })}><PackageCheck className="mr-2 h-4 w-4" />Handed to courier</Button>}
                    {order.fulfillment_status === 'with_courier' && <Button size="sm" onClick={() => updateFulfillment.mutate({ orderId: order.id, status: 'delivered' })}><CheckCircle2 className="mr-2 h-4 w-4" />Mark delivered</Button>}
                    {order.fulfillment_type === 'cash_on_delivery' && !isPaid && <Button size="sm" variant="outline" onClick={() => recordPayment.mutate(order)}><Banknote className="mr-2 h-4 w-4" />Payment received</Button>}
                    {(order.fulfillment_status === 'completed' || (order.fulfillment_status === 'delivered' && isPaid)) && <span className="inline-flex items-center gap-1.5 text-sm font-medium text-emerald-700"><CheckCircle2 className="h-4 w-4" />Order complete</span>}
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

function paidAmount(order: Order) {
  return order.payments?.filter((payment) => payment.status === 'completed').reduce((sum, payment) => sum + payment.amount, 0) || 0
}
