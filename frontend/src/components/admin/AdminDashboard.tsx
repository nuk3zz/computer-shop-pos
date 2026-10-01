import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import apiClient from '@/api/client'
import { formatMoney } from '@/lib/shop-settings'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { 
  DollarSign, 
  ShoppingCart, 
  Wrench,
  UserCog,
  TrendingUp,
  Plus,
  Settings,
  BarChart3
} from 'lucide-react'

interface IncomeBreakdownItem {
  period: string
  orders: number
  gross: number
  tax: number
  net: number
}

interface SalesActivityItem {
  id: string
  order_number: string
  customer_name: string
  customer_phone: string
  status: string
  fulfillment_status: string
  total: number
  paid: number
  created_at: string
  item_name: string
  quantity: number
  payment_method: string
}

function SalesActivity({ items }: { items: SalesActivityItem[] }) {
  const orders = new Map<string, { order: SalesActivityItem; items: string[] }>()
  for (const item of items) {
    const entry = orders.get(item.id) ?? { order: item, items: [] }
    entry.items.push(`${item.item_name} × ${item.quantity}`)
    orders.set(item.id, entry)
  }
  const label = (value: string) => value.replace(/_/g, ' ')
  const methods: Record<string, string> = { cash: 'Cash', debit_card: 'Debit card', digital_wallet: 'Bank transfer', credit_card: 'Credit card' }
  return <section>
    <h3 className="mb-1 text-sm font-semibold">Sales & order activity</h3>
    <p className="mb-3 text-xs text-muted-foreground">Includes pending orders. Only paid sales contribute to the income totals above.</p>
    <div className="max-h-80 overflow-auto" tabIndex={0} aria-label="Sales and order activity">
      <table className="w-full min-w-[680px] text-left text-sm">
        <thead className="sticky top-0 bg-white text-xs text-muted-foreground"><tr>{['Date / reference', 'Items', 'Customer', 'Status', 'Amount'].map(text => <th key={text} className="p-2 font-medium">{text}</th>)}</tr></thead>
        <tbody>{Array.from(orders.values()).map(({ order, items }) => <tr key={order.id} className="border-t align-top">
          <td className="p-2"><div>{new Date(order.created_at).toLocaleString()}</div><div className="text-xs text-muted-foreground">{order.order_number}</div></td>
          <td className="p-2">{items.map((name, index) => <div key={index}>{name}</div>)}</td>
          <td className="p-2"><div>{order.customer_name || 'Walk-in customer'}</div><div className="text-xs text-muted-foreground">{order.customer_phone || 'No phone provided'}</div></td>
          <td className="p-2"><div className="capitalize">{label(order.status)} · {label(order.fulfillment_status)}</div><div className={`text-xs ${order.status === 'cancelled' ? 'text-red-600' : order.paid >= order.total ? 'text-green-600' : 'text-amber-600'}`}>{order.status === 'cancelled' ? 'Cancelled' : order.paid >= order.total ? `${methods[order.payment_method] || 'Payment'} received` : order.paid > 0 ? 'Partially paid' : 'Payment pending'}</div></td>
          <td className="p-2 whitespace-nowrap"><div>{formatMoney(order.total)}</div>{order.paid < order.total && order.status !== 'cancelled' && <div className="text-xs text-amber-600">Due {formatMoney(order.total - order.paid)}</div>}</td>
        </tr>)}</tbody>
      </table>
      {orders.size === 0 && <p className="py-5 text-center text-sm text-muted-foreground">No orders in this period.</p>}
    </div>
  </section>
}

export function AdminDashboard() {
  const [selectedPeriod, setSelectedPeriod] = useState<'today' | 'week' | 'month'>('today')

  // Fetch dashboard stats
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['dashboardStats'],
    queryFn: () => apiClient.getDashboardStats().then(res => res.data)
  })

  // Fetch income report
  const { data: income, isLoading: incomeLoading } = useQuery({
    queryKey: ['incomeReport', selectedPeriod],
    queryFn: () => apiClient.getIncomeReport(selectedPeriod).then(res => res.data)
  })

  if (statsLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Business Dashboard</h1>
          <p className="text-muted-foreground">
            Monitor sales, service jobs, and daily business activity
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link to="/admin/settings">
              <Settings className="w-4 h-4 mr-2" />
              Settings
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to="/admin/reports">
              <BarChart3 className="w-4 h-4 mr-2" />
              Reports
            </Link>
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Today's Orders</CardTitle>
            <ShoppingCart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.today_orders || 0}</div>
            <p className="text-xs text-muted-foreground">
              Sales and repair tickets created today
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Today's Sales Collected</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-purple-600">{formatMoney(stats?.today_revenue || 0)}</div>
            <p className="text-xs text-muted-foreground">
              Full customer payments received today
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Transactions</CardTitle>
            <ShoppingCart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.active_orders || 0}</div>
            <p className="text-xs text-muted-foreground">
              Currently being processed
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Open Repairs</CardTitle>
            <Wrench className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.open_repairs || 0}</div>
            <p className="text-xs text-muted-foreground">
              Waiting, diagnosing, or repairing
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Income Report */}
      <Card className="col-span-4">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                Income Report
              </CardTitle>
              <CardDescription>
                Detailed breakdown of revenue and performance
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <Button 
                variant={selectedPeriod === 'today' ? 'default' : 'outline'} 
                size="sm"
                onClick={() => setSelectedPeriod('today')}
              >
                Today
              </Button>
              <Button 
                variant={selectedPeriod === 'week' ? 'default' : 'outline'} 
                size="sm"
                onClick={() => setSelectedPeriod('week')}
              >
                Week
              </Button>
              <Button 
                variant={selectedPeriod === 'month' ? 'default' : 'outline'} 
                size="sm"
                onClick={() => setSelectedPeriod('month')}
              >
                Month
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {incomeLoading ? (
            <div className="flex justify-center py-8">
              <div className="w-6 h-6 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
            </div>
          ) : income ? (
            <div className="space-y-6">
              {/* Summary */}
              <div className="grid gap-4 md:grid-cols-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">
                    {income.summary.total_orders}
                  </div>
                  <div className="text-sm text-muted-foreground">Total Orders</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-purple-600">
                    {formatMoney(income.summary.gross_income)}
                  </div>
                  <div className="text-sm text-muted-foreground">Sales Collected</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">
                    {formatMoney(income.summary.cost_of_goods)}
                  </div>
                  <div className="text-sm text-muted-foreground">Product / Service Cost</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {formatMoney(income.summary.gross_profit)}
                  </div>
                  <div className="text-sm text-muted-foreground">Gross Profit</div>
                </div>
              </div>

              {/* Breakdown Table */}
              {income.breakdown && income.breakdown.length > 0 && (
                <div className="max-h-64 overflow-auto border rounded-lg">
                  <div className="grid grid-cols-5 gap-4 p-4 bg-muted/50 font-medium text-sm">
                    <div>Period</div>
                    <div className="text-center">Orders</div>
                    <div className="text-center">Gross</div>
                    <div className="text-center">Tax</div>
                    <div className="text-center">Net</div>
                  </div>
                  {income.breakdown.map((item: IncomeBreakdownItem, index: number) => (
                    <div key={index} className="grid grid-cols-5 gap-4 p-4 border-t text-sm">
                      <div className="font-medium">
                        {new Date(item.period).toLocaleDateString()}
                      </div>
                      <div className="text-center">{item.orders}</div>
                      <div className="text-center">{formatMoney(item.gross)}</div>
                      <div className="text-center">{formatMoney(item.tax)}</div>
                      <div className="text-center font-medium">{formatMoney(item.net)}</div>
                    </div>
                  ))}
                </div>
              )}
              <SalesActivity items={income.activity ?? []} />
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              No income data available
            </div>
          )}
        </CardContent>
      </Card>

      {/* Quick Actions */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Link to="/admin/catalog" className="rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
          <Card className="h-full cursor-pointer transition-shadow hover:shadow-lg">
            <CardHeader className="text-center">
              <Plus className="h-8 w-8 mx-auto text-blue-600" />
              <CardTitle className="text-lg">Catalog & Inventory</CardTitle>
              <CardDescription>Add products, services, images, prices, and categories</CardDescription>
            </CardHeader>
          </Card>
        </Link>

        <Link to="/admin/repairs" className="rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
          <Card className="h-full cursor-pointer transition-shadow hover:shadow-lg">
            <CardHeader className="text-center">
              <Wrench className="h-8 w-8 mx-auto text-green-600" />
              <CardTitle className="text-lg">Repair Tickets</CardTitle>
              <CardDescription>Track diagnostics, active repairs, and pickups</CardDescription>
            </CardHeader>
          </Card>
        </Link>

        <Link to="/admin/staff" className="rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
          <Card className="h-full cursor-pointer transition-shadow hover:shadow-lg">
            <CardHeader className="text-center">
              <UserCog className="h-8 w-8 mx-auto text-purple-600" />
              <CardTitle className="text-lg">Manage Staff</CardTitle>
              <CardDescription>Add, edit staff accounts and manage permissions</CardDescription>
            </CardHeader>
          </Card>
        </Link>

        <Link to="/admin/reports" className="rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
          <Card className="h-full cursor-pointer transition-shadow hover:shadow-lg">
            <CardHeader className="text-center">
              <BarChart3 className="h-8 w-8 mx-auto text-orange-600" />
              <CardTitle className="text-lg">View Reports</CardTitle>
              <CardDescription>Detailed analytics and performance reports</CardDescription>
            </CardHeader>
          </Card>
        </Link>
      </div>
    </div>
  )
}
