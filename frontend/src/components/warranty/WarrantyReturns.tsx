import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, ClipboardCheck, MessageCircle, PackageCheck, Plus, RotateCcw, Search, ShieldCheck } from 'lucide-react'
import apiClient from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { buildWhatsAppUrl, formatMoney, loadShopSettings, renderWhatsAppTemplate } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'
import type { WarrantyClaim, WarrantyClaimInput, WarrantyClaimUpdate, WarrantyReplacementSource, WarrantyResolution, WarrantyStatus, WarrantySupplierStatus } from '@/types'

const statusLabels: Record<WarrantyStatus, string> = {
  received: 'Received',
  checking: 'Being checked',
  awaiting_supplier: 'Awaiting supplier',
  ready_for_customer: 'Ready for customer',
  returned: 'Returned to customer',
  cancelled: 'Cancelled',
}

const statusClasses: Record<WarrantyStatus, string> = {
  received: 'bg-slate-100 text-slate-700', checking: 'bg-blue-100 text-blue-700',
  awaiting_supplier: 'bg-violet-100 text-violet-700',
  ready_for_customer: 'bg-amber-100 text-amber-800', returned: 'bg-emerald-100 text-emerald-700',
  cancelled: 'bg-red-100 text-red-700',
}

const resolutionLabels: Record<WarrantyResolution, string> = {
  pending: 'Pending inspection', no_fault_found: 'Working / no fault found', replacement: 'Replacement', refund: 'Refund',
}
const supplierLabels: Record<WarrantySupplierStatus, string> = {
  not_sent: 'Not sent to supplier', waiting: 'Waiting for supplier', replaced: 'Supplier replacement received',
  refunded: 'Supplier refunded shop', rejected: 'Supplier rejected claim',
}

type ClaimDraft = WarrantyClaimInput & { order_item_id: string }
const emptyDraft: ClaimDraft = { order_item_id: '', customer_name: '', customer_phone: '', serial_number: '', issue_description: '', received_condition: '', notes: '' }

export function WarrantyReturns() {
  const queryClient = useQueryClient()
  const [showCreate, setShowCreate] = useState(false)
  const [draft, setDraft] = useState<ClaimDraft>(emptyDraft)
  const [filter, setFilter] = useState<'active' | 'all' | WarrantyStatus>('active')
  const [search, setSearch] = useState('')
  const [resolving, setResolving] = useState<string | null>(null)
  const [resolution, setResolution] = useState<WarrantyResolution>('no_fault_found')
  const [supplierStatus, setSupplierStatus] = useState<WarrantySupplierStatus>('not_sent')
  const [supplierRecovery, setSupplierRecovery] = useState('')
  const [replacementSource, setReplacementSource] = useState<WarrantyReplacementSource>('shop_stock')
  const [replacementProductId, setReplacementProductId] = useState('')
  const [refundAmount, setRefundAmount] = useState('')
  const [resolutionNotes, setResolutionNotes] = useState('')

  const { data: claims = [], isLoading } = useQuery({ queryKey: ['warranty-claims'], queryFn: () => apiClient.getWarrantyClaims().then((r) => r.data || []), refetchInterval: 30_000 })
  const { data: orders = [] } = useQuery({ queryKey: ['warranty-sale-orders'], queryFn: () => apiClient.getOrders({ order_type: 'sale', per_page: 100 }).then((r) => r.data || []) })
  const { data: products = [] } = useQuery({ queryKey: ['warranty-products'], queryFn: () => apiClient.getProducts({ per_page: 100 }).then((r) => r.data || []) })

  const soldItems = useMemo(() => orders.flatMap((order) => (order.items || []).map((item) => ({ order, item }))), [orders])
  const visibleClaims = claims.filter((claim) => {
    if (filter === 'active' && ['returned', 'cancelled'].includes(claim.status)) return false
    if (filter !== 'active' && filter !== 'all' && claim.status !== filter) return false
    const needle = search.trim().toLowerCase()
    return !needle || [claim.claim_number, claim.order_number, claim.customer_name, claim.customer_phone, claim.product_name, claim.serial_number || ''].some((v) => v.toLowerCase().includes(needle))
  })

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['warranty-claims'] })
    queryClient.invalidateQueries({ queryKey: ['warranty-products'] })
    queryClient.invalidateQueries({ queryKey: ['products'] })
  }
  const createClaim = useMutation({
    mutationFn: () => apiClient.createWarrantyClaim(draft),
    onSuccess: (response) => { refresh(); setDraft(emptyDraft); setShowCreate(false); toastHelpers.success('Warranty return created', `${response.data?.claim_number || 'Return'} is now in Received status.`) },
    onError: (error) => toastHelpers.apiError('Create warranty return', error),
  })
  const updateClaim = useMutation({
    mutationFn: ({ id, update }: { id: string; update: WarrantyClaimUpdate }) => apiClient.updateWarrantyClaim(id, update),
    onSuccess: () => { refresh(); setResolving(null); setResolutionNotes(''); setReplacementProductId(''); setRefundAmount(''); setSupplierRecovery(''); toastHelpers.success('Warranty return updated', 'The status and history were saved.') },
    onError: (error) => toastHelpers.apiError('Update warranty return', error),
  })

  const chooseSoldItem = (orderItemId: string) => {
    const selected = soldItems.find(({ item }) => item.id === orderItemId)
    setDraft((current) => ({ ...current, order_item_id: orderItemId, customer_name: selected?.order.customer_name || current.customer_name, customer_phone: selected?.order.customer_phone || current.customer_phone }))
  }
  const openWhatsApp = (claim: WarrantyClaim) => {
    const settings = loadShopSettings()
    const template = ['checking', 'awaiting_supplier'].includes(claim.status) ? settings.whatsapp_warranty_checking : claim.status === 'returned' ? settings.whatsapp_warranty_returned : settings.whatsapp_warranty_ready
    window.open(buildWhatsAppUrl(claim.customer_phone, renderWhatsAppTemplate(template, { customer: claim.customer_name, jobNumber: claim.claim_number })), '_blank', 'noopener,noreferrer')
  }
  const openResolution = (claim: WarrantyClaim) => {
    setResolving(claim.id); setSupplierStatus(claim.supplier_status); setSupplierRecovery(String(claim.supplier_recovery_amount || ''))
    setResolution('no_fault_found'); setReplacementSource('shop_stock'); setReplacementProductId(''); setRefundAmount(''); setResolutionNotes('')
  }
  const resolveClaim = (claim: WarrantyClaim) => updateClaim.mutate({ id: claim.id, update: {
    status: 'ready_for_customer', resolution, supplier_status: supplierStatus,
    supplier_recovery_amount: supplierStatus === 'refunded' ? Number(supplierRecovery) : 0,
    replacement_source: resolution === 'replacement' ? replacementSource : 'none',
    replacement_product_id: resolution === 'replacement' && replacementSource === 'shop_stock' ? replacementProductId : undefined,
    refund_amount: resolution === 'refund' ? Number(refundAmount) : 0, notes: resolutionNotes || undefined,
  } })

  return <div className="min-h-screen space-y-6 bg-slate-50 p-6">
    <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div><h1 className="flex items-center gap-3 text-3xl font-bold tracking-tight"><ShieldCheck className="h-8 w-8" />Warranty Returns</h1><p className="mt-1 text-muted-foreground">Track returned products from intake and checking through replacement, refund, or customer collection.</p></div>
      <Button onClick={() => setShowCreate((v) => !v)}><Plus className="mr-2 h-4 w-4" />New warranty return</Button>
    </div>

    {showCreate && <Card><CardHeader><CardTitle>Receive returned item</CardTitle></CardHeader><CardContent className="grid gap-4 md:grid-cols-2">
      <Field label="Original sold item"><select className="h-10 w-full rounded-md border bg-white px-3 text-sm" value={draft.order_item_id} onChange={(e) => chooseSoldItem(e.target.value)}><option value="">Choose sale and item…</option>{soldItems.map(({ order, item }) => <option key={item.id} value={item.id}>{order.order_number} · {item.product?.name || 'Product'} · {order.customer_name || 'Counter customer'}</option>)}</select></Field>
      <Field label="Serial number (optional)"><Input value={draft.serial_number} onChange={(e) => setDraft({ ...draft, serial_number: e.target.value })} /></Field>
      <Field label="Customer name"><Input value={draft.customer_name} onChange={(e) => setDraft({ ...draft, customer_name: e.target.value })} /></Field>
      <Field label="WhatsApp / contact number"><Input value={draft.customer_phone} onChange={(e) => setDraft({ ...draft, customer_phone: e.target.value })} /></Field>
      <Field label="Reported problem"><Textarea rows={3} value={draft.issue_description} onChange={(e) => setDraft({ ...draft, issue_description: e.target.value })} /></Field>
      <Field label="Condition when received (optional)"><Textarea rows={3} value={draft.received_condition} onChange={(e) => setDraft({ ...draft, received_condition: e.target.value })} /></Field>
      <div className="md:col-span-2"><Field label="Internal notes (optional)"><Textarea rows={2} value={draft.notes} onChange={(e) => setDraft({ ...draft, notes: e.target.value })} /></Field></div>
      <div className="flex gap-2 md:col-span-2"><Button onClick={() => createClaim.mutate()} disabled={createClaim.isPending || !draft.order_item_id || !draft.customer_name.trim() || !draft.customer_phone.trim() || !draft.issue_description.trim()}><ClipboardCheck className="mr-2 h-4 w-4" />Receive item</Button><Button variant="outline" onClick={() => setShowCreate(false)}>Cancel</Button></div>
    </CardContent></Card>}

    <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between"><div className="relative max-w-md flex-1"><Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" /><Input className="pl-9" placeholder="Search return, customer, item, or serial…" value={search} onChange={(e) => setSearch(e.target.value)} /></div><div className="flex flex-wrap gap-2">{(['active','received','checking','awaiting_supplier','ready_for_customer','returned','all'] as const).map((value) => <Button key={value} size="sm" variant={filter === value ? 'default' : 'outline'} onClick={() => setFilter(value)}>{value === 'active' ? 'Active' : value === 'all' ? 'All' : statusLabels[value]}</Button>)}</div></div>

    {isLoading ? <div className="text-muted-foreground">Loading warranty returns…</div> : visibleClaims.length === 0 ? <div className="rounded-xl border border-dashed bg-white p-12 text-center text-muted-foreground">No warranty returns in this view.</div> : <div className="grid gap-4 xl:grid-cols-2">{visibleClaims.map((claim) => <Card key={claim.id}><CardHeader className="pb-3"><div className="flex items-start justify-between gap-3"><div><CardTitle className="text-lg">{claim.product_name}</CardTitle><p className="mt-1 font-mono text-xs text-muted-foreground">{claim.claim_number} · Sale {claim.order_number}</p></div><Badge className={statusClasses[claim.status]}>{statusLabels[claim.status]}</Badge></div></CardHeader><CardContent className="space-y-4">
      <div className="grid grid-cols-2 gap-3 text-sm"><div><span className="text-muted-foreground">Customer</span><div className="font-medium">{claim.customer_name}</div><div>{claim.customer_phone}</div></div><div><span className="text-muted-foreground">Received</span><div>{new Date(claim.received_at).toLocaleString()}</div>{claim.serial_number && <div className="mt-1">Serial: {claim.serial_number}</div>}</div></div>
      <div className="rounded-lg bg-slate-100 p-3 text-sm"><span className="font-medium">Reported issue:</span> {claim.issue_description}{claim.received_condition && <div className="mt-2 text-muted-foreground">Received condition: {claim.received_condition}</div>}</div>
      {claim.supplier_status !== 'not_sent' && <div className="text-sm"><span className="text-muted-foreground">Supplier: </span><span className="font-medium">{supplierLabels[claim.supplier_status]}</span>{claim.supplier_recovery_amount > 0 && <span> · recovered {formatMoney(claim.supplier_recovery_amount)}</span>}</div>}
      {claim.resolution !== 'pending' && <div className="text-sm"><span className="text-muted-foreground">Resolution: </span><span className="font-medium">{resolutionLabels[claim.resolution]}</span>{claim.replacement_source === 'supplier' && <span> · supplier-provided unit</span>}{claim.replacement_product_name && <span> · {claim.replacement_product_name}</span>}{claim.refund_amount > 0 && <span> · {formatMoney(claim.refund_amount)}</span>}</div>}
      {resolving === claim.id && <div className="space-y-3 rounded-lg border p-3"><Field label="Supplier outcome"><select className="h-10 w-full rounded-md border bg-white px-3 text-sm" value={supplierStatus} onChange={(e) => setSupplierStatus(e.target.value as WarrantySupplierStatus)}><option value="not_sent">Not sent to supplier</option><option value="waiting">Still waiting for supplier</option><option value="replaced">Supplier gave replacement</option><option value="refunded">Supplier refunded shop</option><option value="rejected">Supplier rejected claim</option></select></Field>{supplierStatus === 'refunded' && <Field label="Amount recovered from supplier"><Input type="number" min="0.01" step="0.01" value={supplierRecovery} onChange={(e) => setSupplierRecovery(e.target.value)} /></Field>}<Field label="Customer resolution"><select className="h-10 w-full rounded-md border bg-white px-3 text-sm" value={resolution} onChange={(e) => setResolution(e.target.value as WarrantyResolution)}><option value="no_fault_found">Working / no fault found</option><option value="replacement">Provide replacement</option><option value="refund">Refund customer</option></select></Field>{resolution === 'replacement' && <><Field label="Replacement source"><select className="h-10 w-full rounded-md border bg-white px-3 text-sm" value={replacementSource} onChange={(e) => { const value = e.target.value as WarrantyReplacementSource; setReplacementSource(value); if (value === 'supplier') setSupplierStatus('replaced') }}><option value="shop_stock">Use shop stock</option><option value="supplier">Supplier-provided replacement</option></select></Field>{replacementSource === 'shop_stock' && <Field label="Replacement item"><select className="h-10 w-full rounded-md border bg-white px-3 text-sm" value={replacementProductId} onChange={(e) => setReplacementProductId(e.target.value)}><option value="">Choose in-stock product…</option>{products.filter((p) => p.item_type === 'product' && p.stock_quantity > 0).map((p) => <option key={p.id} value={p.id}>{p.name} · {p.stock_quantity} in stock</option>)}</select></Field>}<p className="text-xs text-muted-foreground">Shop stock deducts one unit and its saved cost from profit. A supplier-provided unit does not consume shop stock.</p></>}{resolution === 'refund' && <><Field label="Customer refund amount"><Input type="number" min="0.01" step="0.01" value={refundAmount} onChange={(e) => setRefundAmount(e.target.value)} /></Field><p className="text-xs text-muted-foreground">The refund reduces Sales Collected and Gross Profit when the item is returned to the customer.</p></>}<Field label="Resolution notes (optional)"><Textarea rows={2} value={resolutionNotes} onChange={(e) => setResolutionNotes(e.target.value)} /></Field><div className="flex gap-2"><Button size="sm" onClick={() => resolveClaim(claim)} disabled={supplierStatus === 'waiting' || (resolution === 'replacement' && replacementSource === 'shop_stock' && !replacementProductId) || (resolution === 'refund' && Number(refundAmount) <= 0) || (supplierStatus === 'refunded' && Number(supplierRecovery) <= 0)}>Mark ready for customer</Button><Button size="sm" variant="outline" onClick={() => setResolving(null)}>Cancel</Button></div></div>}
      <div className="flex flex-wrap gap-2">{claim.status === 'received' && <Button size="sm" onClick={() => updateClaim.mutate({ id: claim.id, update: { status: 'checking' } })}><ClipboardCheck className="mr-2 h-4 w-4" />Start checking</Button>}{claim.status === 'checking' && <><Button size="sm" onClick={() => openResolution(claim)}><PackageCheck className="mr-2 h-4 w-4" />Set result</Button><Button size="sm" variant="outline" onClick={() => updateClaim.mutate({ id: claim.id, update: { status: 'awaiting_supplier' } })}>Send to supplier / wait</Button></>}{claim.status === 'awaiting_supplier' && <Button size="sm" onClick={() => openResolution(claim)}><PackageCheck className="mr-2 h-4 w-4" />Record supplier result</Button>}{claim.status === 'ready_for_customer' && <Button size="sm" onClick={() => updateClaim.mutate({ id: claim.id, update: { status: 'returned' } })}><CheckCircle2 className="mr-2 h-4 w-4" />Returned to customer</Button>}{['checking','awaiting_supplier','ready_for_customer','returned'].includes(claim.status) && <Button size="sm" variant="outline" onClick={() => openWhatsApp(claim)}><MessageCircle className="mr-2 h-4 w-4" />WhatsApp</Button>}{['received','checking','awaiting_supplier'].includes(claim.status) && <Button size="sm" variant="ghost" onClick={() => updateClaim.mutate({ id: claim.id, update: { status: 'cancelled' } })}><RotateCcw className="mr-2 h-4 w-4" />Cancel return</Button>}</div>
    </CardContent></Card>)}</div>}
  </div>
}

function Field({ label, children }: { label: string; children: React.ReactNode }) { return <div><label className="mb-1.5 block text-sm font-medium">{label}</label>{children}</div> }
