import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import apiClient from '@/api/client'
import { getMediaUrl } from '@/lib/media'
import { formatMoney } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'
import { ProductImageGallery } from '@/components/catalog/ProductImageGallery'
import type { Customer, ProcessPaymentRequest, Product } from '@/types'
import { Check, Minus, Package, Plus, Search, ShoppingCart, User, Wrench, X, ZoomIn } from 'lucide-react'

interface CartLine {
  product: Product
  quantity: number
  sellingPrice?: string
}

export function SalesWorkspace() {
  const [selectedCategory, setSelectedCategory] = useState('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [orderType, setOrderType] = useState<'sale' | 'service'>('sale')
  const [fulfillmentType, setFulfillmentType] = useState<'in_store' | 'pickup' | 'delivery' | 'cash_on_delivery'>('in_store')
  const [paymentMethod, setPaymentMethod] = useState<ProcessPaymentRequest['payment_method']>('cash')
  const [customerName, setCustomerName] = useState('')
  const [customerPhone, setCustomerPhone] = useState('')
  const [selectedCustomerId, setSelectedCustomerId] = useState<string>()
  const [showCustomerSuggestions, setShowCustomerSuggestions] = useState(false)
  const [deviceNotes, setDeviceNotes] = useState('')
  const [cart, setCart] = useState<CartLine[]>([])
  const [tileSize, setTileSize] = useState(150)
  const [galleryProduct, setGalleryProduct] = useState<Product | null>(null)
  const queryClient = useQueryClient()

  const { data: categories = [] } = useQuery({
    queryKey: ['categories'],
    queryFn: () => apiClient.getCategories().then((response) => response.data || []),
  })

  const { data: products = [], isLoading } = useQuery({
    queryKey: ['products', selectedCategory],
    queryFn: async () => {
      const response = selectedCategory === 'all'
        ? await apiClient.getProducts({ available: true, per_page: 100 })
        : await apiClient.getProductsByCategory(selectedCategory)
      return response.data || []
    },
  })

  const customerSearch = customerName.trim()
  const { data: customerSuggestions = [] } = useQuery({
    queryKey: ['customer-suggestions', customerSearch],
    queryFn: () => apiClient.getCustomers({ search: customerSearch, per_page: 6 }).then((response) => response.data || []),
    enabled: customerSearch.length >= 2 && !selectedCustomerId,
  })

  const createOrder = useMutation({
    mutationFn: () => apiClient.createOrder({
      order_type: orderType,
      fulfillment_type: orderType === 'sale' ? fulfillmentType : undefined,
      payment_method: orderType === 'sale' && fulfillmentType !== 'cash_on_delivery' ? paymentMethod : undefined,
      customer_id: selectedCustomerId,
      customer_name: customerName.trim() || undefined,
      customer_phone: customerPhone.trim() || undefined,
      notes: deviceNotes.trim() || undefined,
      items: cart.map((line) => ({
        product_id: line.product.id,
        quantity: line.quantity,
        selling_price: line.sellingPrice?.trim() ? Number(line.sellingPrice) : undefined,
      })),
    }),
    onSuccess: (response) => {
      toastHelpers.orderCreated(response.data?.order_number)
      setCart([])
      setCustomerName('')
      setCustomerPhone('')
      setSelectedCustomerId(undefined)
      setShowCustomerSuggestions(false)
      setDeviceNotes('')
      queryClient.invalidateQueries({ queryKey: ['orders'] })
      queryClient.invalidateQueries({ queryKey: ['repair-orders'] })
      queryClient.invalidateQueries({ queryKey: ['dashboardStats'] })
      queryClient.invalidateQueries({ queryKey: ['customers'] })
      queryClient.invalidateQueries({ queryKey: ['products'] })
      queryClient.invalidateQueries({ queryKey: ['salesReport'] })
      queryClient.invalidateQueries({ queryKey: ['incomeReport'] })
    },
    onError: (error) => toastHelpers.apiError('Create transaction', error),
  })

  const filteredProducts = products.filter((product) => {
    const search = searchTerm.trim().toLowerCase()
    if (!search) return true
    return product.name.toLowerCase().includes(search)
      || product.description?.toLowerCase().includes(search)
      || product.sku?.toLowerCase().includes(search)
  })

  const addToCart = (product: Product) => {
    if (product.item_type === 'product' && product.stock_quantity <= 0) return
    if (product.item_type === 'service') setOrderType('service')
    setCart((lines) => {
      const existing = lines.find((line) => line.product.id === product.id)
      if (!existing) return [...lines, { product, quantity: 1 }]
      if (product.item_type === 'product' && existing.quantity >= product.stock_quantity) return lines
      return lines.map((line) => line.product.id === product.id
        ? { ...line, quantity: line.quantity + 1 }
        : line)
    })
  }

  const removeFromCart = (productId: string) => {
    setCart((lines) => lines.flatMap((line) => {
      if (line.product.id !== productId) return [line]
      if (line.quantity === 1) return []
      return [{ ...line, quantity: line.quantity - 1 }]
    }))
  }

  const linePrice = (line: CartLine) => line.sellingPrice?.trim() ? Number(line.sellingPrice) : line.product.price
  const validPrices = cart.every((line) => !line.sellingPrice?.trim() || (Number.isFinite(linePrice(line)) && linePrice(line) >= 0 && linePrice(line) <= line.product.price && /^\d+(\.\d{1,2})?$/.test(line.sellingPrice.trim())))
  const subtotal = cart.reduce((total, line) => total + Math.round(linePrice(line) * 100) * line.quantity, 0) / 100
  const itemCount = cart.reduce((total, line) => total + line.quantity, 0)
  const hasServiceContact = customerName.trim().length > 0 && customerPhone.trim().length > 0
  const needsDeliveryContact = orderType === 'sale' && (fulfillmentType === 'delivery' || fulfillmentType === 'cash_on_delivery')
  const canSubmit = cart.length > 0 && validPrices && (orderType === 'sale' ? !needsDeliveryContact || hasServiceContact : hasServiceContact)

  const selectCustomer = (customer: Customer) => {
    setSelectedCustomerId(customer.id)
    setCustomerName(customer.name)
    setCustomerPhone(customer.phone)
    setShowCustomerSuggestions(false)
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="border-b bg-white px-6 py-5">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Sales & Services</h1>
            <p className="mt-1 text-muted-foreground">Sell products, arrange delivery, or create a customer repair ticket.</p>
          </div>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <label className="flex items-center gap-2 text-sm text-muted-foreground">
              <ZoomIn className="h-4 w-4" /> Card width
              <input type="range" min="130" max="220" step="10" value={tileSize} onChange={(event) => setTileSize(Number(event.target.value))} className="w-24" />
            </label>
            <div className="flex rounded-lg border bg-slate-50 p-1">
              <Button variant={orderType === 'sale' ? 'default' : 'ghost'} onClick={() => setOrderType('sale')} className="gap-2">
                <ShoppingCart className="h-4 w-4" /> Product sale
              </Button>
              <Button variant={orderType === 'service' ? 'default' : 'ghost'} onClick={() => setOrderType('service')} className="gap-2">
                <Wrench className="h-4 w-4" /> Service / repair
              </Button>
            </div>
          </div>
        </div>

        <div className="relative mt-5">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={searchTerm}
            onChange={(event) => setSearchTerm(event.target.value)}
            placeholder="Search by item, service, or SKU..."
            className="h-11 pl-10"
          />
        </div>

        <div className="mt-4 flex gap-2 overflow-x-auto pb-1">
          <Button variant={selectedCategory === 'all' ? 'default' : 'outline'} onClick={() => setSelectedCategory('all')}>
            All items
          </Button>
          {categories.map((category) => (
            <Button
              key={category.id}
              variant={selectedCategory === category.id ? 'default' : 'outline'}
              onClick={() => setSelectedCategory(category.id)}
              className="whitespace-nowrap"
            >
              {category.name}
            </Button>
          ))}
        </div>
      </div>

      <div className="grid min-h-[calc(100vh-210px)] grid-cols-1 xl:grid-cols-[1fr_390px]">
        <div className="p-5">
          {isLoading ? (
            <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${tileSize}px, 1fr))` }}>
              {Array.from({ length: 6 }).map((_, index) => <div key={index} className="h-64 animate-pulse rounded-xl bg-slate-200" />)}
            </div>
          ) : filteredProducts.length === 0 ? (
            <div className="flex min-h-80 flex-col items-center justify-center rounded-xl border border-dashed bg-white text-center">
              <Package className="mb-3 h-12 w-12 text-slate-300" />
              <h2 className="font-semibold">No matching catalog items</h2>
              <p className="mt-1 text-sm text-muted-foreground">Add products or services from Catalog & Inventory.</p>
            </div>
          ) : (
            <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${tileSize}px, 1fr))` }}>
              {filteredProducts.map((product) => {
                const quantity = cart.find((line) => line.product.id === product.id)?.quantity || 0
                return (
                  <Card key={product.id} className="flex h-full flex-col overflow-hidden shadow-sm transition-colors hover:border-slate-400">
                    <button type="button" onClick={() => setGalleryProduct(product)} className="aspect-[16/10] w-full overflow-hidden bg-slate-100 text-left" aria-label={`View details for ${product.name}`}>
                      {product.image_url ? (
                        <img src={getMediaUrl(product.image_url)} alt={product.name} className="h-full w-full object-cover" />
                      ) : (
                        <div className="flex h-full items-center justify-center">
                          <Package className="h-9 w-9 text-slate-300" />
                        </div>
                      )}
                    </button>
                    <CardContent className="flex flex-1 flex-col p-2.5">
                      <h2 className="line-clamp-2 text-sm font-semibold leading-4">{product.name}</h2>
                      <div className="mt-1 text-sm font-bold">{formatMoney(product.price)}</div>
                      {product.description && <p className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">{product.description}</p>}
                      <div className="mt-1.5 truncate text-[11px] text-muted-foreground">
                        {product.category?.name || 'Uncategorized'}
                        {product.item_type === 'service' && ` · Service${product.preparation_time > 0 ? ` · ${Math.max(1, Math.ceil(product.preparation_time / 1440))}d` : ''}`}
                      </div>
                      {product.item_type === 'product' && (
                        <div className={`mt-0.5 text-[11px] font-semibold ${product.stock_quantity > 0 ? 'text-emerald-600' : product.preorder_enabled ? 'text-amber-600' : 'text-red-600'}`}>
                          {product.stock_quantity > 0 ? `In stock · ${product.stock_quantity}` : product.preorder_enabled ? 'Pre-order' : 'Out of stock'}
                        </div>
                      )}
                      <div className="mt-auto flex items-center justify-end gap-1.5 pt-2">
                        {quantity > 0 && (
                          <>
                            <Button variant="outline" size="icon" className="h-8 w-8" onClick={() => removeFromCart(product.id)}><Minus className="h-3.5 w-3.5" /></Button>
                            <span className="w-5 text-center text-sm font-semibold">{quantity}</span>
                          </>
                        )}
                        <Button size="sm" className={quantity > 0 ? 'h-8 w-8 p-0' : 'h-8 px-3'} onClick={() => addToCart(product)} disabled={!product.is_available || (product.item_type === 'product' && (product.stock_quantity <= 0 || quantity >= product.stock_quantity))}>
                          <Plus className="h-3.5 w-3.5" />{quantity === 0 && <span className="ml-1.5">Add</span>}
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </div>

        <aside className="border-l bg-white p-5">
          <div className="sticky top-5 space-y-5">
            <div>
              <div className="flex items-center justify-between">
                <h2 className="flex items-center gap-2 text-lg font-semibold">
                  {orderType === 'service' ? <Wrench className="h-5 w-5" /> : <ShoppingCart className="h-5 w-5" />}
                  {orderType === 'service' ? 'Repair ticket' : 'Current sale'}
                </h2>
                <Badge variant="secondary">{itemCount} item{itemCount === 1 ? '' : 's'}</Badge>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {orderType === 'service' ? 'Customer name and phone are required for service work.' : needsDeliveryContact ? 'Customer name and phone are required for delivery.' : 'Customer details are optional for counter sales.'}
              </p>
            </div>

            {orderType === 'sale' && (
              <div className="space-y-3 rounded-lg border bg-slate-50 p-3">
                <div className="text-sm font-semibold">How will the customer receive it?</div>
                <div className="grid grid-cols-2 gap-2">
                  {[
                    ['in_store', 'Instant sale'],
                    ['pickup', 'Customer pickup'],
                    ['delivery', 'Delivery · paid now'],
                    ['cash_on_delivery', 'Cash on delivery'],
                  ].map(([value, label]) => (
                    <button key={value} type="button" onClick={() => setFulfillmentType(value as typeof fulfillmentType)} className={`rounded-md border px-2.5 py-2 text-left text-xs font-medium ${fulfillmentType === value ? 'border-slate-900 bg-slate-900 text-white' : 'bg-white hover:border-slate-400'}`}>{label}</button>
                  ))}
                </div>
                {fulfillmentType !== 'cash_on_delivery' ? (
                  <label className="block text-xs font-medium text-slate-700">Payment received by
                    <select value={paymentMethod} onChange={(event) => setPaymentMethod(event.target.value as ProcessPaymentRequest['payment_method'])} className="mt-1 h-9 w-full rounded-md border bg-white px-2 text-sm">
                      <option value="cash">Cash</option>
                      <option value="credit_card">Credit card</option>
                      <option value="debit_card">Debit card</option>
                      <option value="digital_wallet">Bank transfer / digital wallet</option>
                    </select>
                  </label>
                ) : <p className="text-xs text-amber-700">Payment remains due until you mark it received in Product Orders.</p>}
              </div>
            )}

            <div className="space-y-3">
              <div className="relative">
                <User className="absolute left-3 top-1/2 z-10 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={customerName}
                  onFocus={() => setShowCustomerSuggestions(true)}
                  onChange={(event) => {
                    setCustomerName(event.target.value)
                    setSelectedCustomerId(undefined)
                    setShowCustomerSuggestions(true)
                  }}
                  placeholder={orderType === 'service' ? 'Customer name *' : 'Customer name (optional)'}
                  className="pl-10 pr-10"
                />
                {(customerName || selectedCustomerId) && (
                  <button
                    type="button"
                    aria-label="Clear selected client"
                    onClick={() => {
                      setCustomerName('')
                      setCustomerPhone('')
                      setSelectedCustomerId(undefined)
                    }}
                    className="absolute right-3 top-1/2 z-10 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
                {showCustomerSuggestions && !selectedCustomerId && customerSearch.length >= 2 && customerSuggestions.length > 0 && (
                  <div className="absolute z-30 mt-1 w-full overflow-hidden rounded-md border bg-white shadow-lg">
                    {customerSuggestions.map((customer) => (
                      <button
                        key={customer.id}
                        type="button"
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => selectCustomer(customer)}
                        className="flex w-full items-center justify-between gap-3 border-b px-3 py-2 text-left last:border-0 hover:bg-slate-50"
                      >
                        <span className="min-w-0">
                          <span className="block truncate text-sm font-medium">{customer.name}</span>
                          <span className="block truncate text-xs text-muted-foreground">{customer.phone}</span>
                        </span>
                        <span className="shrink-0 text-xs text-muted-foreground">{customer.order_count} past</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
              {selectedCustomerId && (
                <div className="flex items-center gap-2 text-xs font-medium text-emerald-700">
                  <Check className="h-3.5 w-3.5" /> Existing client selected
                </div>
              )}
              <Input
                value={customerPhone}
                onChange={(event) => {
                  setCustomerPhone(event.target.value)
                  setSelectedCustomerId(undefined)
                }}
                placeholder={orderType === 'service' ? 'WhatsApp phone * (example: 0771234567)' : 'WhatsApp phone (optional)'}
                inputMode="tel"
              />
              <Textarea
                value={deviceNotes}
                onChange={(event) => setDeviceNotes(event.target.value)}
                placeholder={orderType === 'service' ? 'Device, fault, password handling note, requested work...' : 'Sale notes (optional)'}
                rows={4}
              />
            </div>

            <div className="max-h-[38vh] space-y-2 overflow-y-auto">
              {cart.length === 0 ? (
                <div className="rounded-lg border border-dashed px-4 py-10 text-center text-sm text-muted-foreground">Select an item or service to begin.</div>
              ) : cart.map((line) => (
                <div key={line.product.id} className="flex items-center gap-3 rounded-lg border p-3">
                  <div className="h-12 w-12 shrink-0 overflow-hidden rounded-md bg-slate-100">
                    {line.product.image_url
                      ? <img src={getMediaUrl(line.product.image_url)} alt="" className="h-full w-full object-cover" />
                      : <Package className="m-3 h-6 w-6 text-slate-300" />}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">{line.product.name}</div>
                    <div className="text-sm text-muted-foreground">{line.quantity} × {formatMoney(linePrice(line))}</div>
                    <label className="mt-1 block text-xs text-muted-foreground">
                      Selling price per item (optional)
                      <Input type="number" min="0" max={line.product.price} step="0.01" className="mt-1 h-8" aria-label={`Selling price for ${line.product.name}`} placeholder={String(line.product.price)} value={line.sellingPrice || ''} onChange={(event) => setCart((lines) => lines.map((entry) => entry.product.id === line.product.id ? { ...entry, sellingPrice: event.target.value } : entry))} />
                    </label>
                  </div>
                  <div className="font-semibold">{formatMoney(linePrice(line) * line.quantity)}</div>
                </div>
              ))}
            </div>

            <div className="space-y-3 border-t pt-4">
              <p className="text-xs text-muted-foreground">Leave selling price blank to use the catalog price. Changes apply to this transaction only.</p>
              {!validPrices && <p className="text-xs text-red-600">Enter a price from zero to the catalog price, with up to two decimal places.</p>}
              <div className="flex items-center justify-between text-lg font-bold"><span>Subtotal</span><span>{formatMoney(subtotal)}</span></div>
              <Button
                size="lg"
                className="w-full"
                disabled={!canSubmit || createOrder.isPending}
                onClick={() => createOrder.mutate()}
              >
                {createOrder.isPending ? 'Saving...' : orderType === 'service' ? 'Create trackable repair ticket' : 'Create sale'}
              </Button>
            </div>
          </div>
        </aside>
      </div>
      <ProductImageGallery product={galleryProduct} onClose={() => setGalleryProduct(null)} />
    </div>
  )
}
