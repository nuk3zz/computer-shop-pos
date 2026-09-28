import { createFileRoute, Navigate, useRouter } from '@tanstack/react-router'
import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import apiClient from '@/api/client'
import type { LoginRequest, LoginResponse, APIResponse } from '@/types'
import { Eye, EyeOff, Store, Package, Wrench, BarChart3, ShieldCheck } from 'lucide-react'

export const Route = createFileRoute('/login')({
  component: LoginPage,
})

function LoginPage() {
  const router = useRouter()
  const [formData, setFormData] = useState<LoginRequest>({ username: '', password: '' })
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const { data: setupProfile, isLoading: isSetupLoading } = useQuery({
    queryKey: ['setup-status'],
    queryFn: () => apiClient.getSetupStatus().then((response) => response.data),
  })

  const loginMutation = useMutation({
    mutationFn: async (credentials: LoginRequest) => {
      const response: APIResponse<LoginResponse> = await apiClient.login(credentials)
      return response
    },
    onSuccess: (data) => {
      console.log('Login success:', data)
      console.log('Current API URL:', import.meta.env.VITE_API_URL)
      if (data.success && data.data) {
        apiClient.setAuthToken(data.data.token)
        localStorage.setItem('pos_user', JSON.stringify(data.data.user))
        console.log('Auth token set, redirecting to home...')
        setTimeout(() => {
          router.navigate({ to: '/' })
        }, 100)
      } else {
        console.error('Login failed:', data)
        setError(data.message || 'Login failed')
      }
    },
    onError: (error: any) => {
      setError(error.message || 'Login failed')
    },
  })

  // Check setup and authentication after declaring hooks.
  if (apiClient.isAuthenticated()) {
    return <Navigate to="/" />
  }
  if (isSetupLoading) {
    return <div className="flex min-h-screen items-center justify-center text-muted-foreground">Loading…</div>
  }
  if (setupProfile && !setupProfile.setup_completed) {
    return <Navigate to="/setup" />
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    
    if (!formData.username || !formData.password) {
      setError('Username and password are required')
      return
    }

    loginMutation.mutate(formData)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex">
      {/* Left Panel - Branding */}
      <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-blue-600 to-indigo-700 p-12 text-white relative overflow-hidden">
        <div className="relative z-10 flex flex-col justify-center max-w-lg">
          <div className="flex items-center gap-3 mb-8">
            <div className="w-12 h-12 bg-white/20 rounded-lg flex items-center justify-center">
              <Store className="w-7 h-7" />
            </div>
            <h1 className="text-3xl font-bold">Universal Repair POS</h1>
          </div>
          
          <h2 className="text-4xl font-bold mb-6 leading-tight">
            Sales and Repair Management
            <br />
            <span className="text-blue-200">in One Place</span>
          </h2>
          
          <p className="text-xl text-blue-100 mb-12 leading-relaxed">
            Manage computer parts, product sales, repair jobs, services, inventory,
            invoices, and business reports from one secure workspace.
          </p>

          <div className="grid grid-cols-2 gap-6">
            {[
              { icon: Package, title: 'Inventory', desc: 'Products, costs, and stock' },
              { icon: Wrench, title: 'Repair Jobs', desc: 'Track service progress' },
              { icon: BarChart3, title: 'Real-time Analytics', desc: 'Business insights' },
              { icon: ShieldCheck, title: 'Secure Access', desc: 'Protected staff accounts' },
            ].map((feature, idx) => (
              <div key={idx} className="flex items-start gap-3">
                <div className="w-8 h-8 bg-white/20 rounded-lg flex items-center justify-center flex-shrink-0">
                  <feature.icon className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="font-semibold text-sm">{feature.title}</h3>
                  <p className="text-blue-200 text-xs">{feature.desc}</p>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-12 p-4 bg-white/10 backdrop-blur-sm rounded-xl border border-white/20">
            <p className="text-sm text-blue-100">
              Private self-hosted system for authorized shop staff.
            </p>
          </div>
        </div>

        {/* Background Pattern */}
        <div className="absolute inset-0 opacity-10">
          <div className="absolute top-0 left-0 w-full h-full"
               style={{
                 backgroundImage: 'radial-gradient(circle at 1px 1px, white 1px, transparent 0)',
                 backgroundSize: '50px 50px'
               }} />
        </div>
      </div>

      {/* Right Panel - Login Form */}
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-md">
          <Card className="shadow-xl border-0">
            <CardHeader className="text-center pb-8">
              <div className="mx-auto w-16 h-16 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl flex items-center justify-center mb-4 shadow-lg">
                <Store className="w-8 h-8 text-white" />
              </div>
              <CardTitle className="text-2xl font-bold">Universal Repair POS</CardTitle>
              <CardDescription className="text-base">
                Sign in with your staff account
              </CardDescription>
            </CardHeader>

            <CardContent className="space-y-6">
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Username</label>
                  <Input
                    type="text"
                    value={formData.username}
                    onChange={(e) => setFormData(prev => ({ ...prev, username: e.target.value }))}
                    className="h-11"
                    autoComplete="username"
                    disabled={loginMutation.isPending}
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">Password</label>
                  <div className="relative">
                    <Input
                      type={showPassword ? 'text' : 'password'}
                      value={formData.password}
                      onChange={(e) => setFormData(prev => ({ ...prev, password: e.target.value }))}
                      className="h-11 pr-10"
                      autoComplete="current-password"
                      disabled={loginMutation.isPending}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700"
                    >
                      {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>

                {error && (
                  <div className="bg-gradient-to-r from-red-50 to-red-25 border border-red-200 text-red-700 p-4 rounded-lg text-sm shadow-sm">
                    <div className="flex items-center gap-2">
                      <div className="w-4 h-4 bg-red-500 rounded-full flex items-center justify-center flex-shrink-0">
                        <div className="w-1.5 h-1.5 bg-white rounded-full"></div>
                      </div>
                      <span className="font-medium">Login Failed</span>
                    </div>
                    <div className="mt-1 text-xs text-red-600">{error}</div>
                  </div>
                )}

                <Button 
                  type="submit" 
                  className="w-full h-11 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white text-base font-medium transition-all duration-200 shadow-md hover:shadow-lg"
                  disabled={loginMutation.isPending}
                >
                  {loginMutation.isPending ? (
                    <div className="flex items-center justify-center gap-2">
                      <div className="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin" />
                      Signing In...
                    </div>
                  ) : (
                    'Sign In to POS System'
                  )}
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
