import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import apiClient from '@/api/client'
import { ClientForm } from '@/components/clients/ClientForm'
import { getMediaUrl } from '@/lib/media'
import { formatMoney } from '@/lib/shop-settings'
import type { Customer } from '@/types'
import { Clock3, Edit, Phone, Plus, Search, ShoppingCart, UserRound, UsersRound, Wrench } from 'lucide-react'

export function ClientManagement() {
  const [search, setSearch] = useState('')
  const [selectedCustomer, setSelectedCustomer] = useState<Customer>()
  const [showClientForm, setShowClientForm] = useState(false)
  const [editingCustomer, setEditingCustomer] = useState<Customer>()

  const { data: customers = [], isLoading } = useQuery({
    queryKey: ['customers', search],
    queryFn: () => apiClient.getCustomers({ search: search.trim(), per_page: 100 }).then((response) => response.data || []),
  })

  const { data: history = [], isLoading: isHistoryLoading } = useQuery({
    queryKey: ['customer-orders', selectedCustomer?.id],
    queryFn: () => apiClient.getOrders({ customer_id: selectedCustomer!.id, per_page: 100 }).then((response) => response.data || []),
    enabled: Boolean(selectedCustomer),
  })

  const closeClientForm = () => {
    setShowClientForm(false)
    setEditingCustomer(undefined)
    setSelectedCustomer(undefined)
  }

  if (showClientForm || editingCustomer) {
    return (
      <div className="min-h-screen bg-slate-50 p-6">
        <ClientForm customer={editingCustomer} onSuccess={closeClientForm} onCancel={closeClientForm} />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-slate-50 p-6">
      <div className="mx-auto max-w-7xl space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Manage Clients</h1>
            <p className="mt-1 text-muted-foreground">Add clients manually, update their contact details, and review every linked transaction.</p>
          </div>
          <Button onClick={() => setShowClientForm(true)} className="gap-2"><Plus className="h-4 w-4" /> Add Client</Button>
        </div>

        <div className="relative max-w-xl">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search by client name or phone..." className="h-11 bg-white pl-10" />
        </div>

        <div className="grid gap-6 lg:grid-cols-[360px_1fr]">
          <Card className="h-fit">
            <CardHeader className="pb-3">
              <CardTitle className="flex items-center justify-between text-lg">
                <span className="flex items-center gap-2"><UsersRound className="h-5 w-5" /> Clients</span>
                <Badge variant="secondary">{customers.length}</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {isLoading ? (
                <div className="py-10 text-center text-sm text-muted-foreground">Loading clients...</div>
              ) : customers.length === 0 ? (
                <div className="rounded-lg border border-dashed px-4 py-12 text-center">
                  <UserRound className="mx-auto mb-3 h-10 w-10 text-slate-300" />
                  <div className="font-medium">No clients yet</div>
                  <p className="mt-1 text-sm text-muted-foreground">A client is saved automatically when a sale or service includes both name and phone.</p>
                </div>
              ) : customers.map((customer) => (
                <button
                  key={customer.id}
                  type="button"
                  onClick={() => setSelectedCustomer(customer)}
                  className={`w-full rounded-lg border p-3 text-left transition-colors ${selectedCustomer?.id === customer.id ? 'border-slate-900 bg-slate-900 text-white' : 'bg-white hover:bg-slate-50'}`}
                >
                  <div className="flex items-center gap-3">
                    <div className={`flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full ${selectedCustomer?.id === customer.id ? 'bg-slate-700' : 'bg-slate-100'}`}>
                      {customer.image_url ? <img src={getMediaUrl(customer.image_url)} alt="" className="h-full w-full object-cover" /> : <UserRound className="h-5 w-5 text-slate-400" />}
                    </div>
                    <div className="min-w-0">
                      <div className="truncate font-semibold">{customer.name}</div>
                      <div className={`mt-1 flex items-center gap-1.5 text-sm ${selectedCustomer?.id === customer.id ? 'text-slate-300' : 'text-muted-foreground'}`}>
                        <Phone className="h-3.5 w-3.5" /> {customer.phone}
                      </div>
                    </div>
                  </div>
                  <div className={`mt-2 text-xs ${selectedCustomer?.id === customer.id ? 'text-slate-300' : 'text-muted-foreground'}`}>
                    {customer.order_count} transaction{customer.order_count === 1 ? '' : 's'} · {formatMoney(customer.total_spent)}
                  </div>
                </button>
              ))}
            </CardContent>
          </Card>

          <Card className="min-h-[480px]">
            {!selectedCustomer ? (
              <CardContent className="flex min-h-[480px] flex-col items-center justify-center text-center">
                <Clock3 className="mb-3 h-12 w-12 text-slate-300" />
                <h2 className="font-semibold">Select a client</h2>
                <p className="mt-1 text-sm text-muted-foreground">Their product purchases and service history will appear here.</p>
              </CardContent>
            ) : (
              <>
                <CardHeader className="border-b">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-center gap-3">
                      <div className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-full bg-slate-100">
                        {selectedCustomer.image_url ? <img src={getMediaUrl(selectedCustomer.image_url)} alt="" className="h-full w-full object-cover" /> : <UserRound className="h-7 w-7 text-slate-400" />}
                      </div>
                      <div>
                        <CardTitle>{selectedCustomer.name}</CardTitle>
                        <div className="mt-1 text-sm text-muted-foreground">{selectedCustomer.phone}</div>
                      </div>
                    </div>
                    <Button variant="outline" size="sm" onClick={() => setEditingCustomer(selectedCustomer)} className="gap-2"><Edit className="h-4 w-4" /> Edit Client</Button>
                  </div>
                  <div className="flex flex-wrap gap-x-5 gap-y-1 text-sm text-muted-foreground">
                    {selectedCustomer.email && <span>{selectedCustomer.email}</span>}
                    <span>{selectedCustomer.order_count} total transactions</span>
                    <span>{formatMoney(selectedCustomer.total_spent)} total value</span>
                  </div>
                  {selectedCustomer.notes && <p className="text-sm text-muted-foreground">{selectedCustomer.notes}</p>}
                </CardHeader>
                <CardContent className="space-y-3 pt-5">
                  {isHistoryLoading ? (
                    <div className="py-12 text-center text-sm text-muted-foreground">Loading history...</div>
                  ) : history.length === 0 ? (
                    <div className="rounded-lg border border-dashed py-12 text-center text-sm text-muted-foreground">No linked history yet.</div>
                  ) : history.map((order) => (
                    <div key={order.id} className="rounded-lg border bg-white p-4">
                      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                        <div>
                          <div className="flex flex-wrap items-center gap-2">
                            {order.order_type === 'service' ? <Wrench className="h-4 w-4" /> : <ShoppingCart className="h-4 w-4" />}
                            <span className="font-semibold">{order.order_number}</span>
                            <Badge variant={order.order_type === 'service' ? 'default' : 'secondary'}>{order.order_type === 'service' ? 'Service' : 'Product sale'}</Badge>
                            <Badge variant="outline" className="capitalize">{order.status}</Badge>
                          </div>
                          <div className="mt-2 text-sm text-muted-foreground">
                            {order.items?.map((item) => `${item.quantity} × ${item.product?.name || 'Catalog item'}`).join(', ')}
                          </div>
                          {order.notes && <div className="mt-1 text-sm text-muted-foreground">{order.notes}</div>}
                        </div>
                        <div className="shrink-0 text-left sm:text-right">
                          <div className="font-bold">{formatMoney(order.total_amount)}</div>
                          <div className="mt-1 text-xs text-muted-foreground">{new Date(order.created_at).toLocaleString()}</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </CardContent>
              </>
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
