import { useState } from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { RepairQueue } from '@/components/repairs/RepairQueue'
import { SalesWorkspace } from '@/components/sales/SalesWorkspace'
import apiClient from '@/api/client'
import type { User } from '@/types'
import { LayoutDashboard, LogOut, ShoppingCart, UserRound, Wrench } from 'lucide-react'

interface RoleBasedLayoutProps {
  user: User
}

export function RoleBasedLayout({ user }: RoleBasedLayoutProps) {
  const defaultView = user.role === 'technician' ? 'repairs' : 'sales'
  const [currentView, setCurrentView] = useState<'sales' | 'repairs'>(defaultView)

  const canUseSales = ['admin', 'manager', 'sales'].includes(user.role)
  const canUseRepairs = ['admin', 'manager', 'technician'].includes(user.role)

  const roleLabel = {
    admin: 'Administrator',
    manager: 'Manager',
    sales: 'Sales Staff',
    technician: 'Repair Technician',
  }[user.role]

  const logout = () => {
    apiClient.clearAuth()
    window.location.href = '/login'
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="flex items-center justify-between border-b bg-white px-6 py-3">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2 font-bold"><LayoutDashboard className="h-5 w-5" /> Universal Repair POS</div>
          <nav className="flex gap-2">
            {canUseSales && <Button size="sm" variant={currentView === 'sales' ? 'default' : 'ghost'} onClick={() => setCurrentView('sales')}><ShoppingCart className="mr-2 h-4 w-4" /> Sales & Services</Button>}
            {canUseRepairs && <Button size="sm" variant={currentView === 'repairs' ? 'default' : 'ghost'} onClick={() => setCurrentView('repairs')}><Wrench className="mr-2 h-4 w-4" /> Repair Tickets</Button>}
          </nav>
        </div>
        <div className="flex items-center gap-3">
          <Badge variant="secondary" className="gap-1"><UserRound className="h-3 w-3" /> {roleLabel}</Badge>
          <Button size="sm" variant="outline" onClick={logout}><LogOut className="mr-2 h-4 w-4" /> Logout</Button>
        </div>
      </header>
      {currentView === 'repairs' ? <RepairQueue /> : <SalesWorkspace />}
    </div>
  )
}
