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
                <div className="border rounded-lg">
                  <div className="grid grid-cols-5 gap-4 p-4 bg-muted/50 font-medium text-sm">
                    <div>Period</div>
                    <div className="text-center">Orders</div>
                    <div className="text-center">Gross</div>
                    <div className="text-center">Tax</div>
                    <div className="text-center">Net</div>
                  </div>
                  {income.breakdown.slice(0, 10).map((item: IncomeBreakdownItem, index: number) => (
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
