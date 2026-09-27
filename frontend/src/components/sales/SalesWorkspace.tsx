import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import apiClient from '@/api/client'
import { getMediaUrl } from '@/lib/media'
import { getPreparationTimeDisplay } from '@/lib/utils'
import { formatMoney } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'
import type { Product } from '@/types'
import { Clock, Minus, Package, Plus, Search, ShoppingCart, User, Wrench, ZoomIn } from 'lucide-react'

interface CartLine {
  product: Product
  quantity: number
}

export function SalesWorkspace() {
  const [selectedCategory, setSelectedCategory] = useState('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [orderType, setOrderType] = useState<'sale' | 'service'>('sale')
  const [customerName, setCustomerName] = useState('')
  const [customerPhone, setCustomerPhone] = useState('')
  const [deviceNotes, setDeviceNotes] = useState('')
  const [cart, setCart] = useState<CartLine[]>([])
  const [tileSize, setTileSize] = useState(180)
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

  const createOrder = useMutation({
    mutationFn: () => apiClient.createOrder({
      order_type: orderType,
      customer_name: customerName.trim() || undefined,
      customer_phone: customerPhone.trim() || undefined,
      notes: deviceNotes.trim() || undefined,
      items: cart.map((line) => ({
        product_id: line.product.id,
        quantity: line.quantity,
      })),
    }),
    onSuccess: (response) => {
      toastHelpers.orderCreated(response.data?.order_number)
      setCart([])
      setCustomerName('')
      setCustomerPhone('')
      setDeviceNotes('')
      queryClient.invalidateQueries({ queryKey: ['orders'] })
      queryClient.invalidateQueries({ queryKey: ['repair-orders'] })
      queryClient.invalidateQueries({ queryKey: ['dashboardStats'] })
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
    if (product.item_type === 'service') setOrderType('service')
    setCart((lines) => {
      const existing = lines.find((line) => line.product.id === product.id)
      if (!existing) return [...lines, { product, quantity: 1 }]
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

  const subtotal = cart.reduce((total, line) => total + line.product.price * line.quantity, 0)
  const itemCount = cart.reduce((total, line) => total + line.quantity, 0)
  const canSubmit = cart.length > 0 && (orderType === 'sale' || customerName.trim().length > 0)

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="border-b bg-white px-6 py-5">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Sales & Services</h1>
            <p className="mt-1 text-muted-foreground">Sell computer products or create a customer repair ticket.</p>
          </div>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <label className="flex items-center gap-2 text-sm text-muted-foreground">
              <ZoomIn className="h-4 w-4" /> Tile size
              <input type="range" min="150" max="280" step="10" value={tileSize} onChange={(event) => setTileSize(Number(event.target.value))} className="w-28" />
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
                  <Card key={product.id} className="overflow-hidden transition-shadow hover:shadow-md">
                    <div className="aspect-square overflow-hidden bg-slate-100">
                      {product.image_url ? (
                        <img src={getMediaUrl(product.image_url)} alt={product.name} className="h-full w-full object-cover" />
                      ) : (
                        <div className="flex h-full items-center justify-center">
                          <Package className="h-14 w-14 text-slate-300" />
                        </div>
                      )}
                    </div>
                    <CardContent className="space-y-3 p-4">
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <h2 className="font-semibold leading-tight">{product.name}</h2>
                          <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{product.description || 'No description'}</p>
                        </div>
                        <span className="whitespace-nowrap font-bold">{formatMoney(product.price)}</span>
                      </div>
                      <div className="flex min-h-6 flex-wrap items-center gap-2">
                        {product.category && <Badge variant="outline">{product.category.name}</Badge>}
                        <Badge variant={product.item_type === 'service' ? 'default' : 'secondary'}>{product.item_type === 'service' ? 'Service' : 'Product'}</Badge>
                        {product.preparation_time > 0 && (
                          <Badge variant="secondary" className="gap-1">
                            <Clock className="h-3 w-3" /> {getPreparationTimeDisplay(product.preparation_time)}
                          </Badge>
                        )}
                      </div>
                      <div className="flex items-center justify-end gap-2">
                        {quantity > 0 && (
                          <>
                            <Button variant="outline" size="icon" onClick={() => removeFromCart(product.id)}><Minus className="h-4 w-4" /></Button>
                            <span className="w-6 text-center font-semibold">{quantity}</span>
                          </>
                        )}
                        <Button size={quantity > 0 ? 'icon' : 'default'} onClick={() => addToCart(product)} disabled={!product.is_available}>
                          <Plus className="h-4 w-4" />{quantity === 0 && <span className="ml-2">Add</span>}
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
                {orderType === 'service' ? 'Customer name is required for service work.' : 'Customer name is optional for product sales.'}
              </p>
            </div>

            <div className="space-y-3">
              <div className="relative">
                <User className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={customerName}
                  onChange={(event) => setCustomerName(event.target.value)}
                  placeholder={orderType === 'service' ? 'Customer name *' : 'Customer name (optional)'}
                  className="pl-10"
                />
              </div>
              <Input
                value={customerPhone}
                onChange={(event) => setCustomerPhone(event.target.value)}
                placeholder="WhatsApp phone (example: 0771234567)"
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
                    <div className="text-sm text-muted-foreground">{line.quantity} × {formatMoney(line.product.price)}</div>
                  </div>
                  <div className="font-semibold">{formatMoney(line.product.price * line.quantity)}</div>
                </div>
              ))}
            </div>

            <div className="space-y-3 border-t pt-4">
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
    </div>
  )
}
