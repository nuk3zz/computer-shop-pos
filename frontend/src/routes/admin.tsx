import { createFileRoute, Navigate, Outlet } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import apiClient from '@/api/client'
import type { User } from '@/types'
import { AdminSidebar } from '@/components/admin/AdminSidebar'
import { loadShopSettings, saveShopSettings } from '@/lib/shop-settings'

export const Route = createFileRoute('/admin')({
  component: AdminLayout,
})

function AdminLayout() {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const storedUser = localStorage.getItem('pos_user')
    const token = localStorage.getItem('pos_token')
    
    if (storedUser && token) {
      try {
        const parsedUser = JSON.parse(storedUser)
        setUser(parsedUser)
      } catch (error) {
        console.error('Failed to parse stored user:', error)
        localStorage.removeItem('pos_user')
        localStorage.removeItem('pos_token')
      }
    }
    setIsLoading(false)
  }, [])

  useEffect(() => {
    const refreshUser = (event: Event) => setUser((event as CustomEvent<User>).detail)
    window.addEventListener('user-profile-changed', refreshUser)
    return () => window.removeEventListener('user-profile-changed', refreshUser)
  }, [])

  const { data: shopProfile, isLoading: isProfileLoading } = useQuery({
    queryKey: ['shop-profile'],
    queryFn: async () => {
      const response = await apiClient.getShopProfile()
      if (!response.data) throw new Error('Shop profile is missing')
      return response.data
    },
    enabled: Boolean(user && apiClient.isAuthenticated()),
  })

  useEffect(() => {
    if (!shopProfile?.setup_completed) return
    saveShopSettings({ ...loadShopSettings(), shop_name: shopProfile.company_name })
  }, [shopProfile])

  // Show loading while checking auth
  if (isLoading || (user && isProfileLoading)) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading Admin Panel...</p>
        </div>
      </div>
    )
  }

  // Check authentication
  if (!apiClient.isAuthenticated() || !user) {
    return <Navigate to="/login" />
  }

  // Check admin role
  if (user.role !== 'admin') {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-destructive mb-2">Access Denied</h1>
          <p className="text-muted-foreground mb-4">You don't have admin privileges.</p>
          <Navigate to="/" />
        </div>
      </div>
    )
  }

  if (shopProfile && !shopProfile.setup_completed) {
    return <Navigate to="/setup" />
  }

  return (
    <div className="min-h-screen bg-background flex">
      <AdminSidebar user={user} shopProfile={shopProfile} />
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
