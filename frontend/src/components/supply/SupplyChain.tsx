import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Banknote, MapPin, PackagePlus, Pencil, Phone, Plus, Save, Trash2, Warehouse } from 'lucide-react'
import apiClient from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { formatMoney } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'
import type { Supplier, SupplierInput } from '@/types'

type PurchaseLine = { product_id: string; quantity: number; unit_cost: number }
const emptySupplier: SupplierInput = { name: '', phone: '', location: '', notes: '', credit_allowed: false }
const emptyLine: PurchaseLine = { product_id: '', quantity: 1, unit_cost: 0 }

export function SupplyChain() {
  const queryClient = useQueryClient()
  const [supplierForm, setSupplierForm] = useState<SupplierInput>(emptySupplier)
  const [editingSupplier, setEditingSupplier] = useState<Supplier | null>(null)
  const [selectedSupplierID, setSelectedSupplierID] = useState('')
  const [referenceNumber, setReferenceNumber] = useState('')
  const [amountPaid, setAmountPaid] = useState(0)
  const [purchaseNotes, setPurchaseNotes] = useState('')
  const [purchaseLines, setPurchaseLines] = useState<PurchaseLine[]>([{ ...emptyLine }])
  const [payingSupplier, setPayingSupplier] = useState<Supplier | null>(null)
  const [debtPayment, setDebtPayment] = useState(0)

  const { data: suppliers = [], isLoading } = useQuery({ queryKey: ['suppliers'], queryFn: () => apiClient.getSuppliers().then((response) => response.data || []) })
  const { data: purchases = [] } = useQuery({ queryKey: ['supplier-purchases'], queryFn: () => apiClient.getSupplierPurchases().then((response) => response.data || []) })
  const { data: products = [] } = useQuery({ queryKey: ['products', 'supply-chain'], queryFn: () => apiClient.getProducts({ per_page: 100 }).then((response) => (response.data || []).filter((product) => product.item_type === 'product')) })

  useEffect(() => {
    if (!selectedSupplierID && suppliers.length > 0) setSelectedSupplierID(suppliers[0].id)
  }, [selectedSupplierID, suppliers])

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['suppliers'] })
    queryClient.invalidateQueries({ queryKey: ['supplier-purchases'] })
    queryClient.invalidateQueries({ queryKey: ['products'] })
  }

  const saveSupplier = useMutation({
    mutationFn: () => editingSupplier ? apiClient.updateSupplier(editingSupplier.id, supplierForm) : apiClient.createSupplier(supplierForm),
    onSuccess: () => { refresh(); setSupplierForm(emptySupplier); setEditingSupplier(null); toastHelpers.apiSuccess('Supplier', editingSupplier ? 'Supplier updated' : 'Supplier added') },
    onError: (error) => toastHelpers.apiError('Save supplier', error),
  })

  const createPurchase = useMutation({
    mutationFn: () => apiClient.createSupplierPurchase({ supplier_id: selectedSupplierID, reference_number: referenceNumber.trim() || undefined, amount_paid: amountPaid, notes: purchaseNotes.trim() || undefined, items: purchaseLines }),
    onSuccess: () => { refresh(); setReferenceNumber(''); setAmountPaid(0); setPurchaseNotes(''); setPurchaseLines([{ ...emptyLine }]); toastHelpers.apiSuccess('Stock purchase', 'Inventory and supplier balance updated') },
    onError: (error) => toastHelpers.apiError('Record stock purchase', error),
  })

  const payDebt = useMutation({
    mutationFn: () => apiClient.createSupplierPayment(payingSupplier!.id, debtPayment),
    onSuccess: () => { refresh(); setPayingSupplier(null); setDebtPayment(0); toastHelpers.apiSuccess('Supplier payment', 'Debt payment recorded') },
    onError: (error) => toastHelpers.apiError('Record supplier payment', error),
  })

  const purchaseTotal = purchaseLines.reduce((sum, line) => sum + (Number(line.quantity) || 0) * (Number(line.unit_cost) || 0), 0)
  const selectedSupplier = suppliers.find((supplier) => supplier.id === selectedSupplierID)

  const editSupplier = (supplier: Supplier) => {
    setEditingSupplier(supplier)
    setSupplierForm({ name: supplier.name, phone: supplier.phone || '', location: supplier.location || '', notes: supplier.notes || '', credit_allowed: supplier.credit_allowed })
  }

  return (
    <div className="space-y-6 p-6">
      <div><h1 className="flex items-center gap-3 text-3xl font-bold tracking-tight"><Warehouse className="h-8 w-8" /> Supply Chain</h1><p className="mt-1 text-muted-foreground">Manage suppliers, stock purchases, payments, and outstanding debt.</p></div>

      <div className="grid gap-6 xl:grid-cols-[1fr_420px]">
        <div className="space-y-4">
          <h2 className="text-lg font-semibold">Suppliers</h2>
          {isLoading ? <div className="text-muted-foreground">Loading suppliers…</div> : suppliers.length === 0 ? <div className="rounded-lg border border-dashed p-8 text-center text-muted-foreground">Add your first supplier using the form.</div> : suppliers.map((supplier) => (
            <Card key={supplier.id}>
              <CardContent className="p-4">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div><div className="flex items-center gap-2"><div className="text-lg font-semibold">{supplier.name}</div>{supplier.credit_allowed && <Badge variant="secondary">Credit allowed</Badge>}</div><div className="mt-2 flex flex-wrap gap-4 text-sm text-muted-foreground">{supplier.phone && <span className="flex items-center gap-1"><Phone className="h-3.5 w-3.5" />{supplier.phone}</span>}{supplier.location && <span className="flex items-center gap-1"><MapPin className="h-3.5 w-3.5" />{supplier.location}</span>}</div>{supplier.notes && <p className="mt-2 text-sm text-muted-foreground">{supplier.notes}</p>}</div>
                  <Button variant="outline" size="sm" onClick={() => editSupplier(supplier)}><Pencil className="mr-1.5 h-3.5 w-3.5" />Edit</Button>
                </div>
                <div className="mt-4 grid grid-cols-3 gap-3 border-t pt-3 text-sm"><Metric label="Purchases" value={formatMoney(supplier.total_purchases)} /><Metric label="Paid" value={formatMoney(supplier.total_paid)} /><Metric label="Outstanding debt" value={formatMoney(supplier.outstanding_debt)} danger={supplier.outstanding_debt > 0} /></div>
                {supplier.outstanding_debt > 0 && <Button className="mt-3" size="sm" variant="outline" onClick={() => { setPayingSupplier(supplier); setDebtPayment(supplier.outstanding_debt) }}><Banknote className="mr-2 h-4 w-4" />Pay debt</Button>}
              </CardContent>
            </Card>
          ))}
        </div>

        <Card className="h-fit">
          <CardHeader><CardTitle>{editingSupplier ? 'Edit supplier' : 'Add supplier'}</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <Field label="Supplier name *"><Input value={supplierForm.name} onChange={(event) => setSupplierForm({ ...supplierForm, name: event.target.value })} /></Field>
            <Field label="Contact number"><Input value={supplierForm.phone || ''} onChange={(event) => setSupplierForm({ ...supplierForm, phone: event.target.value })} /></Field>
            <Field label="Location"><Input value={supplierForm.location || ''} onChange={(event) => setSupplierForm({ ...supplierForm, location: event.target.value })} /></Field>
            <Field label="Notes"><Textarea rows={3} value={supplierForm.notes || ''} onChange={(event) => setSupplierForm({ ...supplierForm, notes: event.target.value })} /></Field>
            <label className="flex items-start gap-2 rounded-lg border p-3 text-sm"><input type="checkbox" className="mt-0.5" checked={supplierForm.credit_allowed} onChange={(event) => setSupplierForm({ ...supplierForm, credit_allowed: event.target.checked })} /><span><span className="block font-medium">Can buy before full payment</span><span className="text-xs text-muted-foreground">Allows this supplier to carry an outstanding balance.</span></span></label>
            <div className="flex gap-2"><Button disabled={!supplierForm.name.trim() || saveSupplier.isPending} onClick={() => saveSupplier.mutate()}><Save className="mr-2 h-4 w-4" />{editingSupplier ? 'Update supplier' : 'Add supplier'}</Button>{editingSupplier && <Button variant="outline" onClick={() => { setEditingSupplier(null); setSupplierForm(emptySupplier) }}>Cancel</Button>}</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><PackagePlus className="h-5 w-5" />Record stock purchase</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          {suppliers.length === 0 ? <p className="text-sm text-muted-foreground">Add a supplier before recording stock.</p> : <>
            <div className="grid gap-3 md:grid-cols-3"><Field label="Supplier"><select className="h-10 w-full rounded-md border bg-white px-3 text-sm" value={selectedSupplierID} onChange={(event) => setSelectedSupplierID(event.target.value)}>{suppliers.map((supplier) => <option key={supplier.id} value={supplier.id}>{supplier.name}</option>)}</select></Field><Field label="Invoice / reference"><Input value={referenceNumber} onChange={(event) => setReferenceNumber(event.target.value)} /></Field><Field label="Amount paid now"><Input type="number" min="0" step="0.01" value={amountPaid} onChange={(event) => setAmountPaid(Number(event.target.value))} /></Field></div>
            <div className="space-y-2">{purchaseLines.map((line, index) => <div key={index} className="grid gap-2 rounded-lg border p-3 md:grid-cols-[1fr_110px_150px_40px]"><select className="h-10 rounded-md border bg-white px-3 text-sm" value={line.product_id} onChange={(event) => { const product = products.find((item) => item.id === event.target.value); setPurchaseLines((lines) => lines.map((value, lineIndex) => lineIndex === index ? { ...value, product_id: event.target.value, unit_cost: product?.cost_price || 0 } : value)) }}><option value="">Select product…</option>{products.map((product) => <option key={product.id} value={product.id}>{product.name}</option>)}</select><Input type="number" min="1" value={line.quantity} onChange={(event) => setPurchaseLines((lines) => lines.map((value, lineIndex) => lineIndex === index ? { ...value, quantity: Number(event.target.value) } : value))} /><Input type="number" min="0" step="0.01" value={line.unit_cost} onChange={(event) => setPurchaseLines((lines) => lines.map((value, lineIndex) => lineIndex === index ? { ...value, unit_cost: Number(event.target.value) } : value))} /><Button variant="ghost" size="icon" disabled={purchaseLines.length === 1} onClick={() => setPurchaseLines((lines) => lines.filter((_, lineIndex) => lineIndex !== index))}><Trash2 className="h-4 w-4" /></Button></div>)}</div>
            <Button variant="outline" size="sm" onClick={() => setPurchaseLines((lines) => [...lines, { ...emptyLine }])}><Plus className="mr-2 h-4 w-4" />Add another item</Button>
            <Textarea rows={2} placeholder="Purchase notes (optional)" value={purchaseNotes} onChange={(event) => setPurchaseNotes(event.target.value)} />
            <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4"><div><div className="text-lg font-bold">Total: {formatMoney(purchaseTotal)}</div><div className={`text-sm ${purchaseTotal - amountPaid > 0 ? 'text-amber-700' : 'text-muted-foreground'}`}>Balance: {formatMoney(Math.max(0, purchaseTotal - amountPaid))}{purchaseTotal > amountPaid && selectedSupplier && !selectedSupplier.credit_allowed ? ' · Full payment required' : ''}</div></div><Button disabled={createPurchase.isPending || purchaseLines.some((line) => !line.product_id || line.quantity <= 0 || line.unit_cost < 0) || amountPaid > purchaseTotal || (purchaseTotal > amountPaid && !selectedSupplier?.credit_allowed)} onClick={() => createPurchase.mutate()}><PackagePlus className="mr-2 h-4 w-4" />Record purchase & add stock</Button></div>
          </>}
        </CardContent>
      </Card>

      <Card><CardHeader><CardTitle>Recent purchases</CardTitle></CardHeader><CardContent>{purchases.length === 0 ? <p className="text-sm text-muted-foreground">No supplier purchases yet.</p> : <div className="divide-y rounded-lg border">{purchases.map((purchase) => <div key={purchase.id} className="grid gap-2 p-3 text-sm sm:grid-cols-4"><div><div className="font-medium">{purchase.supplier_name}</div><div className="text-xs text-muted-foreground">{new Date(purchase.purchased_at).toLocaleString()}</div></div><div>Purchase<br /><strong>{formatMoney(purchase.total_amount)}</strong></div><div>Paid<br /><strong>{formatMoney(purchase.amount_paid)}</strong></div><div className={purchase.balance > 0 ? 'text-amber-700' : 'text-emerald-700'}>Balance<br /><strong>{formatMoney(purchase.balance)}</strong></div></div>)}</div>}</CardContent></Card>

      {payingSupplier && <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"><Card className="w-full max-w-md"><CardHeader><CardTitle>Pay {payingSupplier.name}</CardTitle></CardHeader><CardContent className="space-y-4"><p className="text-sm text-muted-foreground">Outstanding debt: {formatMoney(payingSupplier.outstanding_debt)}</p><Field label="Payment amount"><Input type="number" min="0.01" max={payingSupplier.outstanding_debt} step="0.01" value={debtPayment} onChange={(event) => setDebtPayment(Number(event.target.value))} /></Field><div className="flex gap-2"><Button disabled={debtPayment <= 0 || debtPayment > payingSupplier.outstanding_debt || payDebt.isPending} onClick={() => payDebt.mutate()}><Banknote className="mr-2 h-4 w-4" />Record payment</Button><Button variant="outline" onClick={() => setPayingSupplier(null)}>Cancel</Button></div></CardContent></Card></div>}
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) { return <label className="block text-sm font-medium"><span className="mb-1.5 block">{label}</span>{children}</label> }
function Metric({ label, value, danger = false }: { label: string; value: string; danger?: boolean }) { return <div><div className="text-xs text-muted-foreground">{label}</div><div className={`mt-1 font-semibold ${danger ? 'text-amber-700' : ''}`}>{value}</div></div> }
